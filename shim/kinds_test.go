package shim_test

import (
	"testing"

	"github.com/microsoft/typescript-go/shim"
)

// TestKindConstantsMatch proves the re-exported kind constants name the same
// nodes the checker produces: the source file walks to a function declaration
// and an identifier that carry the expected kinds.
func TestKindConstantsMatch(t *testing.T) {
	t.Parallel()
	p := shim.Compile(map[string]string{
		"/src/main.ts": "function area(r: number) { return r; }\n",
	}, shim.Options{})
	defer p.Close()

	var sawFunc, sawIdent bool
	var walk func(n *shim.Node) bool
	walk = func(n *shim.Node) bool {
		switch n.Kind {
		case shim.KindFunctionDeclaration:
			sawFunc = true
		case shim.KindIdentifier:
			sawIdent = true
		}
		shim.ForEachChild(n, walk)
		return false
	}
	for _, f := range p.SourceFiles() {
		if f.FileName() == "/src/main.ts" {
			walk(f.AsNode())
		}
	}
	if !sawFunc {
		t.Error("did not find a function declaration by its kind constant")
	}
	if !sawIdent {
		t.Error("did not find an identifier by its kind constant")
	}
}

// TestCastExpressionKinds proves the as-cast, angle-bracket assertion, and
// non-null assertion kinds name the nodes the parser produces for each form.
func TestCastExpressionKinds(t *testing.T) {
	t.Parallel()
	p := shim.Compile(map[string]string{
		"/src/main.ts": "const a = 1 as number;\nconst b = <number>2;\nconst c: number | null = 3;\nconst d = c!;\n",
	}, shim.Options{})
	defer p.Close()

	var sawAs, sawAssertion, sawNonNull bool
	var walk func(n *shim.Node) bool
	walk = func(n *shim.Node) bool {
		switch n.Kind {
		case shim.KindAsExpression:
			sawAs = true
		case shim.KindTypeAssertionExpression:
			sawAssertion = true
		case shim.KindNonNullExpression:
			sawNonNull = true
		}
		shim.ForEachChild(n, walk)
		return false
	}
	for _, f := range p.SourceFiles() {
		if f.FileName() == "/src/main.ts" {
			walk(f.AsNode())
		}
	}
	if !sawAs {
		t.Error("did not find an as-cast by its kind constant")
	}
	if !sawAssertion {
		t.Error("did not find an angle-bracket assertion by its kind constant")
	}
	if !sawNonNull {
		t.Error("did not find a non-null assertion by its kind constant")
	}
}

// TestImportSpecifiers proves the file's import edges come back as written.
func TestImportSpecifiers(t *testing.T) {
	t.Parallel()
	p := shim.Compile(map[string]string{
		"/src/main.ts":  "import { x } from \"./other\";\nexport const y = x;\n",
		"/src/other.ts": "export const x = 1;\n",
	}, shim.Options{RootFiles: []string{"/src/main.ts"}})
	defer p.Close()

	var main *shim.SourceFile
	for _, f := range p.SourceFiles() {
		if f.FileName() == "/src/main.ts" {
			main = f
		}
	}
	if main == nil {
		t.Fatal("main file not found")
	}
	specs := shim.ImportSpecifiers(main)
	if len(specs) != 1 || specs[0] != "./other" {
		t.Errorf("import specifiers = %v, want [./other]", specs)
	}
	if resolved, ok := p.ResolvedModule(main, "./other"); !ok || resolved != "/src/other.ts" {
		t.Errorf("resolved ./other to %q ok=%v, want /src/other.ts true", resolved, ok)
	}
}

// TestFileNameOfNode proves a node deep in the tree reports its own file.
func TestFileNameOfNode(t *testing.T) {
	t.Parallel()
	p := shim.Compile(map[string]string{
		"/src/main.ts": "const n = 1;\n",
	}, shim.Options{})
	defer p.Close()

	var ident *shim.Node
	var walk func(n *shim.Node) bool
	walk = func(n *shim.Node) bool {
		if ident == nil && n.Kind == shim.KindIdentifier {
			ident = n
			return true
		}
		shim.ForEachChild(n, walk)
		return ident != nil
	}
	for _, f := range p.SourceFiles() {
		if f.FileName() == "/src/main.ts" {
			walk(f.AsNode())
		}
	}
	if ident == nil {
		t.Fatal("no identifier found")
	}
	if got := shim.FileName(ident); got != "/src/main.ts" {
		t.Errorf("FileName(ident) = %q, want /src/main.ts", got)
	}
}

// TestClassKeywordKinds proves the this and super keyword kinds name the nodes a
// class body carries: this inside a method and super inside a subclass method.
func TestClassKeywordKinds(t *testing.T) {
	t.Parallel()
	p := shim.Compile(map[string]string{
		"/src/main.ts": `class Base {
  x: number = 0;
  get(): number { return this.x; }
}
class Sub extends Base {
  get(): number { return super.get() + 1; }
}
`,
	}, shim.Options{})
	defer p.Close()

	var sawThis, sawSuper bool
	var walk func(n *shim.Node) bool
	walk = func(n *shim.Node) bool {
		switch n.Kind {
		case shim.KindThisKeyword:
			sawThis = true
		case shim.KindSuperKeyword:
			sawSuper = true
		}
		shim.ForEachChild(n, walk)
		return false
	}
	for _, f := range p.SourceFiles() {
		if f.FileName() == "/src/main.ts" {
			walk(f.AsNode())
		}
	}
	if !sawThis {
		t.Error("did not find the this keyword by its kind constant")
	}
	if !sawSuper {
		t.Error("did not find the super keyword by its kind constant")
	}
}
