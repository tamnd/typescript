package shim

import "github.com/microsoft/typescript-go/internal/ast"

// EarlyModuleError is a static-semantics early error a module carries that the
// fork's checker does not surface as a diagnostic, but that the spec makes a
// SyntaxError for the whole module. The caller turns it into the SyntaxError an
// engine would throw before running a line of the module.
type EarlyModuleError struct {
	// Message is a human-readable description of the violation, without the
	// "SyntaxError:" prefix a caller adds when it reports the error.
	Message string
}

// EarlyModuleErrors walks a parsed module for the import-declaration early
// errors the fork's parser and checker leave unreported, so a front end that
// AOT-compiles the module rejects it the way an engine would rather than emit
// code that silently runs.
//
// It flags two things. First, an imported binding named eval or arguments:
// module code is strict, and an ImportedBinding may not be eval or arguments
// (ImportSpecifier, NamespaceImport, and default-import forms all bind a local
// and so all count). Export specifiers are deliberately not walked, because
// `export { x as eval }` binds no local name and is legal. Second, a WithClause
// (import attributes) whose keys are not all distinct: the key is compared by
// its cooked value, so a string-literal key with an escape collides with the
// plain identifier it spells.
func EarlyModuleErrors(file *SourceFile) []EarlyModuleError {
	var out []EarlyModuleError
	var walk func(n *Node)
	walk = func(n *Node) {
		if n == nil {
			return
		}
		switch n.Kind {
		case ast.KindImportClause:
			// The default-import binding, e.g. `import eval from "./m"`. The
			// named and namespace bindings are separate nodes the walk reaches
			// on its own.
			if name := n.AsImportClause().Name(); name != nil {
				checkBindingName(name, &out)
			}
		case ast.KindNamespaceImport:
			checkBindingName(n.AsNamespaceImport().Name(), &out)
		case ast.KindImportSpecifier:
			// The local binding of a named import: `{ x }` or `{ x as eval }`.
			// Name() is the local name in both forms.
			checkBindingName(n.AsImportSpecifier().Name(), &out)
		case ast.KindImportAttributes:
			checkDuplicateAttributes(n.AsImportAttributes(), &out)
		}
		n.ForEachChild(func(c *Node) bool { walk(c); return false })
	}
	walk(file.AsNode())
	return out
}

// checkBindingName flags a local import binding named eval or arguments, which
// module (strict) code may not bind.
func checkBindingName(name *Node, out *[]EarlyModuleError) {
	if name == nil || name.Kind != ast.KindIdentifier {
		return
	}
	switch name.Text() {
	case "eval", "arguments":
		*out = append(*out, EarlyModuleError{
			Message: "Cannot use '" + name.Text() + "' as an imported binding name",
		})
	}
}

// checkDuplicateAttributes flags a WithClause whose keys are not all distinct,
// comparing each key by its cooked value so an escaped string-literal key
// collides with the identifier it spells.
func checkDuplicateAttributes(attrs *ast.ImportAttributes, out *[]EarlyModuleError) {
	if attrs == nil || attrs.Attributes == nil {
		return
	}
	seen := make(map[string]bool)
	for _, node := range attrs.Attributes.Nodes {
		key := node.AsImportAttribute().Name()
		if key == nil {
			continue
		}
		text := key.Text()
		if seen[text] {
			*out = append(*out, EarlyModuleError{
				Message: "Duplicate import attribute key '" + text + "'",
			})
			continue
		}
		seen[text] = true
	}
}
