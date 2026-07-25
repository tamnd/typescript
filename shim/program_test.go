package shim_test

import (
	"testing"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/shim"
)

// findFile returns the program's own input file by name, skipping the bundled
// standard library files the program also loads.
func findFile(t *testing.T, p *shim.Program, name string) *shim.SourceFile {
	t.Helper()
	for _, f := range p.SourceFiles() {
		if f.FileName() == name {
			return f
		}
	}
	t.Fatalf("source file %q not found", name)
	return nil
}

// firstNamed finds the first identifier node with the given text, which the
// tests use to point the checker at a specific declaration.
func firstNamed(root *shim.Node, text string) *shim.Node {
	var found *shim.Node
	var walk func(n *shim.Node) bool
	walk = func(n *shim.Node) bool {
		if found != nil {
			return true
		}
		if n.Kind == ast.KindIdentifier && n.Text() == text {
			found = n
			return true
		}
		shim.ForEachChild(n, walk)
		return found != nil
	}
	walk(root)
	return found
}

// TestCompilesRealTypeScript proves the shim runs the real checker: it infers the
// type of a variable from its initializer, which only a real type checker does.
func TestCompilesRealTypeScript(t *testing.T) {
	t.Parallel()
	p := shim.Compile(map[string]string{
		"/src/main.ts": "const count = 1 + 2;\n",
	}, shim.Options{})
	defer p.Close()

	file := findFile(t, p, "/src/main.ts")
	if diags := p.Diagnostics(file); len(diags) != 0 {
		t.Fatalf("clean program has %d diagnostics, first: %s", len(diags), shim.Message(diags[0]))
	}

	id := firstNamed(file.AsNode(), "count")
	if id == nil {
		t.Fatal("identifier count not found in tree")
	}
	ty := p.Checker().GetTypeAtLocation(id)
	if ty.Flags()&shim.TypeFlagsNumberLike == 0 {
		t.Errorf("type of count is not number-like, flags = %v", ty.Flags())
	}
}

// TestNodeText proves NodeText returns a node's own source text without leading
// trivia, for an identifier, a numeric literal, and a binary operator token, the
// three things a code generator reads straight from the checked source.
func TestNodeText(t *testing.T) {
	t.Parallel()
	p := shim.Compile(map[string]string{
		"/src/main.ts": "const total = width * 2;\n",
	}, shim.Options{})
	defer p.Close()

	file := findFile(t, p, "/src/main.ts")

	id := firstNamed(file.AsNode(), "width")
	if id == nil {
		t.Fatal("identifier width not found")
	}
	if got := shim.NodeText(id); got != "width" {
		t.Errorf("NodeText(identifier) = %q, want %q", got, "width")
	}

	// Walk to the binary expression and read its operator token and right operand.
	var bin *shim.Node
	var walk func(n *shim.Node) bool
	walk = func(n *shim.Node) bool {
		if n.Kind == ast.KindBinaryExpression {
			bin = n
			return true
		}
		shim.ForEachChild(n, walk)
		return bin != nil
	}
	walk(file.AsNode())
	if bin == nil {
		t.Fatal("binary expression not found")
	}

	var kids []*shim.Node
	shim.ForEachChild(bin, func(n *shim.Node) bool {
		kids = append(kids, n)
		return false
	})
	if len(kids) != 3 {
		t.Fatalf("binary expression has %d children, want 3 (left, operator, right)", len(kids))
	}
	if got := shim.NodeText(kids[1]); got != "*" {
		t.Errorf("NodeText(operator) = %q, want %q", got, "*")
	}
	if got := shim.NodeText(kids[2]); got != "2" {
		t.Errorf("NodeText(numeric literal) = %q, want %q", got, "2")
	}
}

// forStatement finds the first for statement in a tree, which the ForClauses
// tests point at.
func forStatement(root *shim.Node) *shim.Node {
	var found *shim.Node
	var walk func(n *shim.Node) bool
	walk = func(n *shim.Node) bool {
		if found != nil {
			return true
		}
		if n.Kind == ast.KindForStatement {
			found = n
			return true
		}
		shim.ForEachChild(n, walk)
		return found != nil
	}
	walk(root)
	return found
}

// TestForClausesReadsEveryPart proves ForClauses returns each of a full for
// statement's four parts by role, so a caller reads the initializer, condition,
// incrementor, and body straight from the node.
func TestForClausesReadsEveryPart(t *testing.T) {
	t.Parallel()
	p := shim.Compile(map[string]string{
		"/src/main.ts": "for (let i = 0; i < 3; i++) { console.log(i); }\n",
	}, shim.Options{})
	defer p.Close()

	f := forStatement(findFile(t, p, "/src/main.ts").AsNode())
	if f == nil {
		t.Fatal("for statement not found")
	}
	init, cond, incr, body := shim.ForClauses(f)
	if init == nil || cond == nil || incr == nil || body == nil {
		t.Fatalf("a full for lost a clause: init=%v cond=%v incr=%v body=%v", init != nil, cond != nil, incr != nil, body != nil)
	}
	if got := shim.NodeText(cond); got != "i < 3" {
		t.Errorf("condition text = %q, want %q", got, "i < 3")
	}
	if got := shim.NodeText(incr); got != "i++" {
		t.Errorf("incrementor text = %q, want %q", got, "i++")
	}
}

// TestForClausesReportsOmittedClauses proves ForClauses returns nil for a clause
// the source omits, the case walking children cannot tell apart because
// ForEachChild skips the omission. Here for(;i<3;) drops the initializer and the
// incrementor but keeps the condition.
func TestForClausesReportsOmittedClauses(t *testing.T) {
	t.Parallel()
	p := shim.Compile(map[string]string{
		"/src/main.ts": "let i = 0; for (; i < 3; ) { i++; }\n",
	}, shim.Options{})
	defer p.Close()

	f := forStatement(findFile(t, p, "/src/main.ts").AsNode())
	if f == nil {
		t.Fatal("for statement not found")
	}
	init, cond, incr, body := shim.ForClauses(f)
	if init != nil {
		t.Errorf("omitted initializer read as %q, want nil", shim.NodeText(init))
	}
	if incr != nil {
		t.Errorf("omitted incrementor read as %q, want nil", shim.NodeText(incr))
	}
	if cond == nil {
		t.Fatal("condition was dropped")
	}
	if got := shim.NodeText(cond); got != "i < 3" {
		t.Errorf("condition text = %q, want %q", got, "i < 3")
	}
	if body == nil {
		t.Error("body was dropped")
	}
}

// TestReadsObjectProperties proves the shim exposes a type's members with their
// resolved types, the query type lowering is built on.
func TestReadsObjectProperties(t *testing.T) {
	t.Parallel()
	p := shim.Compile(map[string]string{
		"/src/main.ts": "const point = { x: 1, y: 2 };\n",
	}, shim.Options{})
	defer p.Close()

	file := findFile(t, p, "/src/main.ts")
	id := firstNamed(file.AsNode(), "point")
	if id == nil {
		t.Fatal("identifier point not found")
	}
	ty := p.Checker().GetTypeAtLocation(id)
	if ty.Flags()&shim.TypeFlagsObject == 0 {
		t.Fatalf("type of point is not object, flags = %v", ty.Flags())
	}

	props := p.Checker().GetPropertiesOfType(ty)
	names := map[string]bool{}
	for _, s := range props {
		names[s.Name] = true
		pt := p.Checker().GetTypeOfSymbol(s)
		if pt.Flags()&shim.TypeFlagsNumberLike == 0 {
			t.Errorf("property %q is not number-like, flags = %v", s.Name, pt.Flags())
		}
	}
	if !names["x"] || !names["y"] {
		t.Errorf("want properties x and y, got %v", names)
	}
}

// hasFile reports whether the program lists an input file by name among its
// source files, skipping the bundled standard library.
func hasFile(p *shim.Program, name string) bool {
	for _, f := range p.SourceFiles() {
		if f.FileName() == name {
			return true
		}
	}
	return false
}

// TestAllowJSAdmitsJavaScriptRoot proves the AllowJS option admits a .js root to
// the program as a source file. Without allowJs the checker parses a JavaScript
// root but never lists it, so a caller handed a .js entry gets an empty program;
// with it the file is present and type-checked. The two compiles differ only in
// the flag, so the flag is what the presence turns on.
func TestAllowJSAdmitsJavaScriptRoot(t *testing.T) {
	t.Parallel()
	files := map[string]string{
		"/src/main.js": "function twice(x) { return x + x; }\nconsole.log(twice(21));\n",
	}

	off := shim.Compile(files, shim.Options{})
	defer off.Close()
	if hasFile(off, "/src/main.js") {
		t.Fatal("a .js root was admitted without allowJs; the flag should gate it")
	}

	on := shim.Compile(files, shim.Options{AllowJS: true})
	defer on.Close()
	if !hasFile(on, "/src/main.js") {
		t.Fatal("allowJs did not admit the .js root as a source file")
	}
}

// TestAllowJSWithoutCheckJSSuppressesJSDiagnostics proves that a .js file allowJs
// admits is typed but its JavaScript-specific type errors are not reported unless
// checkJs is also on. A type-incompatible assignment that a .ts file would reject
// produces no diagnostic in a .js file under allowJs alone, which is what a front
// end lowering untyped JavaScript wants: the resolved types without the
// diagnostics gating the build.
func TestAllowJSWithoutCheckJSSuppressesJSDiagnostics(t *testing.T) {
	t.Parallel()
	files := map[string]string{
		"/src/main.js": "const n = 1;\nn();\n",
	}

	p := shim.Compile(files, shim.Options{AllowJS: true})
	defer p.Close()
	file := findFile(t, p, "/src/main.js")
	if diags := p.Diagnostics(file); len(diags) != 0 {
		t.Fatalf("allowJs without checkJs reported %d JS diagnostics, first: %s",
			len(diags), shim.Message(diags[0]))
	}
}

// TestReportsTypeErrors proves the checker is real by making it reject a program
// a real type checker rejects.
func TestReportsTypeErrors(t *testing.T) {
	t.Parallel()
	p := shim.Compile(map[string]string{
		"/src/main.ts": "const n: number = \"not a number\";\n",
	}, shim.Options{})
	defer p.Close()

	file := findFile(t, p, "/src/main.ts")
	diags := p.Diagnostics(file)
	if len(diags) == 0 {
		t.Fatal("assigning a string to a number produced no diagnostic")
	}
}
