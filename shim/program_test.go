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

// TestReadsObjectProperties proves the shim exposes a type's members with their
// resolved types, the query type lowering is built on.
func TestReadsObjectProperties(t *testing.T) {
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

// TestReportsTypeErrors proves the checker is real by making it reject a program
// a real type checker rejects.
func TestReportsTypeErrors(t *testing.T) {
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
