package shim

import (
	"context"
	"strings"
	"sync"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/bundled"
	"github.com/microsoft/typescript-go/internal/compiler"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/locale"
	"github.com/microsoft/typescript-go/internal/scanner"
	"github.com/microsoft/typescript-go/internal/tsoptions"
	"github.com/microsoft/typescript-go/internal/vfs/vfstest"
)

// Options controls how Compile builds a program. The zero value compiles the
// root files under strict type checking against the bundled standard library,
// which is the setting a type-directed caller almost always wants.
type Options struct {
	// RootFiles are the entry files the program is built from. Each must be a key
	// in the Files map passed to Compile. If empty, every file in the map is a
	// root, which is the convenient default for a single-file compile.
	RootFiles []string
	// Loose turns strict type checking off. It is the inverse of the usual
	// default because strict is what a compiler front end wants; the field only
	// exists so a caller can reproduce a non-strict project when it must.
	Loose bool
	// Target names the emit target the checker resolves against, for example
	// "es2015" or "esnext". An empty string keeps the default esnext. It matters
	// for diagnostics that depend on downleveling, such as the helper a
	// pre-es2017 async function needs. Values follow the tsconfig target names.
	Target string
	// NoImplicitAny, when non-nil, sets noImplicitAny independently of Loose, so a
	// caller can reproduce a non-strict project that still asked for the flag. A
	// nil pointer leaves it to Loose: strict implies it, loose leaves it off.
	NoImplicitAny *bool
	// AllowUnreachableCode, when non-nil, sets allowUnreachableCode. A nil pointer
	// keeps the default, under which unreachable code is reported. Setting it
	// false is what makes the checker report unreachable code as an error rather
	// than fold it silently.
	AllowUnreachableCode *bool
	// ImportHelpers turns on importHelpers, under which the checker expects a
	// tslib helper module for downlevel emit and reports its absence. It reproduces
	// a project built with the helper library.
	ImportHelpers bool
	// AllowJS, when true, sets allowJs, under which a .js, .jsx, .mjs, or .cjs root
	// is admitted to the program and type-checked as a source file rather than
	// dropped. Without it the checker parses a JavaScript root but never lists it
	// among the program's source files, so a caller that hands it a .js entry gets
	// an empty program. A caller compiling JavaScript sets this on.
	AllowJS bool
	// CheckJS, when true, sets checkJs, under which the checker reports type errors
	// in the JavaScript files allowJs admitted, the same diagnostics it would in a
	// .ts file. Left off, a .js file is still typed (its expressions resolve to
	// types, unannotated forms widening to any) but its type errors are not
	// reported, which is what a front end that lowers untyped JavaScript wants: the
	// resolved types without the JavaScript-specific diagnostics gating the build.
	CheckJS bool
}

// Program is a compiled, type-checked program with a checker held ready for
// queries. It owns one checker checked out from the program's pool for its whole
// lifetime, so a caller must Close it when done to return that checker.
type Program struct {
	inner   *compiler.Program
	checker *Checker
	release func()
}

// Compile builds and type-checks a program from an in-memory set of files, keyed
// by absolute POSIX path (for example "/src/main.ts"), and returns it with a
// checker ready. The standard library is served from the files bundled into this
// binary, so no on-disk toolchain is needed. The caller owns the returned
// Program and must Close it.
func Compile(files map[string]string, opts Options) *Program {
	m := make(map[string]any, len(files))
	for name, content := range files {
		m[name] = content
	}
	fs := bundled.WrapFS(vfstest.FromMap(m, true /*useCaseSensitiveFileNames*/))

	roots := opts.RootFiles
	if len(roots) == 0 {
		roots = make([]string, 0, len(files))
		for name := range files {
			roots = append(roots, name)
		}
	}

	co := &core.CompilerOptions{
		Target:           targetFromName(opts.Target),
		Module:           core.ModuleKindESNext,
		ModuleResolution: core.ModuleResolutionKindBundler,
	}
	// Strict is set to a decided value either way. Leaving it unknown is not the same
	// as loose: the checker resolves an unset strict sub-flag with `Strict != TSFalse`,
	// so an unknown Strict reads as on and strictNullChecks stays enabled. Setting it
	// false explicitly is what turns the strict family off, so a loose caller gets the
	// non-strict checking it asked for, undefined and null widening to any among it.
	if opts.Loose {
		co.Strict = core.TSFalse
	} else {
		co.Strict = core.TSTrue
	}
	if opts.NoImplicitAny != nil {
		co.NoImplicitAny = triFromBool(*opts.NoImplicitAny)
	}
	// allowJs admits a JavaScript root to the program; checkJs additionally reports
	// its type errors. A caller compiling JavaScript sets allowJs so the .js entry
	// is listed as a source file, and leaves checkJs off so the file is typed
	// without its diagnostics gating the build.
	if opts.AllowJS {
		co.AllowJs = core.TSTrue
	}
	if opts.CheckJS {
		co.CheckJs = core.TSTrue
	}
	if opts.AllowUnreachableCode != nil {
		co.AllowUnreachableCode = triFromBool(*opts.AllowUnreachableCode)
	}
	if opts.ImportHelpers {
		co.ImportHelpers = core.TSTrue
	}

	libPath := bundled.LibPath()
	host := &cachingHost{
		CompilerHost: compiler.NewCompilerHost("/", fs, libPath, nil, nil),
		libPath:      libPath,
	}
	inner := compiler.NewProgram(compiler.ProgramOptions{
		Config: &tsoptions.ParsedCommandLine{
			ParsedConfig: &core.ParsedOptions{
				FileNames:       roots,
				CompilerOptions: co,
			},
		},
		Host: host,
	})

	chk, release := inner.GetTypeChecker(context.Background())
	return &Program{inner: inner, checker: chk, release: release}
}

// triFromBool maps a Go bool onto the checker's tristate, so a caller that knows
// a setting is on or off records it as a decided value rather than leaving it
// unknown, which the checker would resolve from other options.
func triFromBool(b bool) core.Tristate {
	if b {
		return core.TSTrue
	}
	return core.TSFalse
}

// targetFromName maps a tsconfig target name onto the script-target enum, so a
// caller can name the emit target the way a tsconfig does. An empty or
// unrecognized name keeps the esnext default, matching the zero value's meaning.
func targetFromName(name string) core.ScriptTarget {
	switch strings.ToLower(name) {
	case "es6", "es2015":
		return core.ScriptTargetES2015
	case "es2016":
		return core.ScriptTargetES2016
	case "es2017":
		return core.ScriptTargetES2017
	case "es2018":
		return core.ScriptTargetES2018
	case "es2019":
		return core.ScriptTargetES2019
	case "es2020":
		return core.ScriptTargetES2020
	case "es2021":
		return core.ScriptTargetES2021
	case "es2022":
		return core.ScriptTargetES2022
	case "es2023":
		return core.ScriptTargetES2023
	case "es2024":
		return core.ScriptTargetES2024
	case "es2025":
		return core.ScriptTargetES2025
	default:
		return core.ScriptTargetESNext
	}
}

// libFileCache holds parsed standard-library source files so repeated Compile
// calls in one process reparse the multi-megabyte lib.d.ts set only once. The
// lib files are byte-for-byte identical across every compile, and a parsed
// (then bound) source file is safe to share across programs the same way the
// language server's document registry shares it: binding writes symbols onto
// the tree once and is idempotent, while each program's checker keeps its own
// type links off to the side. Input files are never cached here because their
// text changes from call to call.
var libFileCache sync.Map // ast.SourceFileParseOptions -> *ast.SourceFile

// cachingHost wraps a compiler host and serves parsed standard-library files
// from libFileCache. Every non-lib file falls through to the wrapped host, so a
// caller's own inputs are always parsed fresh.
type cachingHost struct {
	compiler.CompilerHost
	libPath string
}

// GetSourceFile returns a cached parse for a lib file and parses everything else
// through the wrapped host. The cache key is the full parse options struct, so a
// change in how a file is parsed (path, module indicator) is a distinct entry.
func (h *cachingHost) GetSourceFile(opts ast.SourceFileParseOptions) *ast.SourceFile {
	if !strings.HasPrefix(opts.FileName, h.libPath) {
		return h.CompilerHost.GetSourceFile(opts)
	}
	if cached, ok := libFileCache.Load(opts); ok {
		return cached.(*ast.SourceFile)
	}
	sf := h.CompilerHost.GetSourceFile(opts)
	if sf == nil {
		return nil
	}
	actual, _ := libFileCache.LoadOrStore(opts, sf)
	return actual.(*ast.SourceFile)
}

// Close returns the checker to the program's pool. After Close the Program's
// Checker must not be used.
func (p *Program) Close() {
	if p.release != nil {
		p.release()
		p.release = nil
	}
}

// Checker returns the program's checker, the entry point for every type and
// symbol query.
func (p *Program) Checker() *Checker { return p.checker }

// SourceFiles returns the program's source files, including the standard library
// files. A caller that only wants its own inputs filters by FileName.
func (p *Program) SourceFiles() []*SourceFile { return p.inner.SourceFiles() }

// Diagnostics returns the syntactic and semantic diagnostics for the given file,
// the errors a caller reports before trusting the checked types.
func (p *Program) Diagnostics(file *SourceFile) []*Diagnostic {
	ctx := context.Background()
	out := p.inner.GetSyntacticDiagnostics(ctx, file)
	return append(out, p.inner.GetSemanticDiagnostics(ctx, file)...)
}

// ResolvedModule resolves an import specifier as written in the given file and
// returns the absolute path of the file it resolves to, or ok=false when the
// import does not resolve. It resolves against the file's own import node so the
// module mode matches the one the checker used, which the cache is keyed on.
func (p *Program) ResolvedModule(from *SourceFile, specifier string) (path string, ok bool) {
	for _, lit := range from.Imports() {
		if lit.Text() != specifier {
			continue
		}
		rm := p.inner.GetResolvedModuleFromModuleSpecifier(from, lit)
		if rm.IsResolved() {
			return rm.ResolvedFileName, true
		}
	}
	return "", false
}

// LineAndCharacter converts a byte position in a file to a zero-based line and
// UTF-16 character offset, using ECMAScript line separators so the numbers match
// what a JavaScript tool reports.
func LineAndCharacter(file *SourceFile, pos int) (line, character int) {
	l, c := scanner.GetECMALineAndUTF16CharacterOfPosition(file, pos)
	return l, int(c)
}

// ForEachChild visits each direct child of a node, stopping early if the visitor
// returns true. It is the traversal primitive a caller walks the tree with.
func ForEachChild(node *Node, visit func(*Node) bool) bool {
	return node.ForEachChild(ast.Visitor(visit))
}

// ForClauses returns the four parts of a for statement by role: the initializer,
// the condition, the incrementor, and the body. Any of the first three is nil
// when the source omits that clause, as in for(;;) or for(let i=0;;i++). A
// caller reads roles this way rather than by walking children, because
// ForEachChild skips an omitted clause and so collapses the positions the roles
// would otherwise sit at. The node must be a for statement.
func ForClauses(node *Node) (init, cond, incr, body *Node) {
	f := node.AsForStatement()
	return f.Initializer, f.Condition, f.Incrementor, f.Statement
}

// NodeText returns the exact source text of a node's own token range, excluding
// leading trivia. It is the primitive a code generator reads an identifier name,
// a numeric or string literal, or a binary operator token with, straight from
// the checked source, so the emitted Go carries the value the source wrote.
func NodeText(node *Node) string {
	return scanner.GetTextOfNode(node)
}

// LiteralValue returns the value of a literal type: a string for a string
// literal, a jsnum.Number for a number literal, a bool for a boolean literal, a
// PseudoBigInt for a bigint literal. It returns nil for a non-literal type.
func LiteralValue(t *Type) any {
	if t.Flags()&TypeFlagsLiteral == 0 {
		return nil
	}
	return t.AsLiteralType().Value()
}

// Message returns the human-readable text of a diagnostic in the default locale.
func Message(d *Diagnostic) string {
	return d.Localize(locale.Locale{})
}
