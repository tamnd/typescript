package shim

import (
	"context"

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
		Target:           core.ScriptTargetESNext,
		Module:           core.ModuleKindESNext,
		ModuleResolution: core.ModuleResolutionKindBundler,
	}
	if !opts.Loose {
		co.Strict = core.TSTrue
	}

	inner := compiler.NewProgram(compiler.ProgramOptions{
		Config: &tsoptions.ParsedCommandLine{
			ParsedConfig: &core.ParsedOptions{
				FileNames:       roots,
				CompilerOptions: co,
			},
		},
		Host: compiler.NewCompilerHost("/", fs, bundled.LibPath(), nil, nil),
	})

	chk, release := inner.GetTypeChecker(context.Background())
	return &Program{inner: inner, checker: chk, release: release}
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
