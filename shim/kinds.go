package shim

import "github.com/microsoft/typescript-go/internal/ast"

// The syntax kinds a tree walker branches on, re-exported so a caller can
// classify a Node without importing internal/ast. The set is the one a code
// generator needs to recognize declarations, statements, and expressions; the
// full kind enum is large, and more values can be added here the same way
// without touching upstream.
const (
	KindUnknown    = ast.KindUnknown
	KindSourceFile = ast.KindSourceFile
	KindIdentifier = ast.KindIdentifier

	// Declarations.
	KindFunctionDeclaration  = ast.KindFunctionDeclaration
	KindFunctionExpression   = ast.KindFunctionExpression
	KindArrowFunction        = ast.KindArrowFunction
	KindMethodDeclaration    = ast.KindMethodDeclaration
	KindGetAccessor          = ast.KindGetAccessor
	KindSetAccessor          = ast.KindSetAccessor
	KindConstructor          = ast.KindConstructor
	KindClassDeclaration     = ast.KindClassDeclaration
	KindInterfaceDeclaration = ast.KindInterfaceDeclaration
	KindTypeAliasDeclaration = ast.KindTypeAliasDeclaration
	KindEnumDeclaration      = ast.KindEnumDeclaration
	KindVariableStatement    = ast.KindVariableStatement
	KindVariableDeclaration  = ast.KindVariableDeclaration
	KindParameter            = ast.KindParameter
	KindPropertyDeclaration  = ast.KindPropertyDeclaration

	// Statements.
	KindBlock               = ast.KindBlock
	KindReturnStatement     = ast.KindReturnStatement
	KindIfStatement         = ast.KindIfStatement
	KindForStatement        = ast.KindForStatement
	KindForOfStatement      = ast.KindForOfStatement
	KindForInStatement      = ast.KindForInStatement
	KindWhileStatement      = ast.KindWhileStatement
	KindSwitchStatement     = ast.KindSwitchStatement
	KindTryStatement        = ast.KindTryStatement
	KindThrowStatement      = ast.KindThrowStatement
	KindExpressionStatement = ast.KindExpressionStatement
	KindWithStatement       = ast.KindWithStatement

	// Expressions.
	KindCallExpression           = ast.KindCallExpression
	KindNewExpression            = ast.KindNewExpression
	KindPropertyAccessExpression = ast.KindPropertyAccessExpression
	KindElementAccessExpression  = ast.KindElementAccessExpression
	KindBinaryExpression         = ast.KindBinaryExpression
	KindPrefixUnaryExpression    = ast.KindPrefixUnaryExpression
	KindPostfixUnaryExpression   = ast.KindPostfixUnaryExpression
	KindConditionalExpression    = ast.KindConditionalExpression
	KindTemplateExpression       = ast.KindTemplateExpression

	// The tagged template, tag`a${x}b`. It is a call, not a string: the tag is
	// invoked with the template's literal parts and its substitution values, so a
	// walker that reads the template expression alone sees a string where the
	// program wrote a call. Its children are the tag and the template.
	KindTaggedTemplateExpression = ast.KindTaggedTemplateExpression

	KindObjectLiteralExpression = ast.KindObjectLiteralExpression
	KindArrayLiteralExpression  = ast.KindArrayLiteralExpression
	KindAwaitExpression         = ast.KindAwaitExpression
	KindYieldExpression         = ast.KindYieldExpression
	KindSpreadElement           = ast.KindSpreadElement
	KindParenthesizedExpression = ast.KindParenthesizedExpression

	// Type-cast expressions the emitter erases to their inner value: the `as`
	// form and the angle-bracket assertion. Both carry a type the checker used to
	// widen or narrow the expression, and both leave the runtime value untouched.
	KindAsExpression            = ast.KindAsExpression
	KindTypeAssertionExpression = ast.KindTypeAssertionExpression

	// The non-null assertion, x!. It belongs with the casts above: the checker
	// strips null and undefined from the operand's type and the emitter drops the
	// operator, so the runtime value is the operand's.
	KindNonNullExpression = ast.KindNonNullExpression

	// Literals and keyword-valued expressions, the leaves a code generator reads
	// a constant from.
	KindNumericLiteral                = ast.KindNumericLiteral
	KindStringLiteral                 = ast.KindStringLiteral
	KindBigIntLiteral                 = ast.KindBigIntLiteral
	KindNoSubstitutionTemplateLiteral = ast.KindNoSubstitutionTemplateLiteral
	KindTrueKeyword                   = ast.KindTrueKeyword
	KindFalseKeyword                  = ast.KindFalseKeyword
	KindNullKeyword                   = ast.KindNullKeyword

	// Class-body keywords: the receiver reference inside a method and the parent
	// reference inside a subclass.
	KindThisKeyword  = ast.KindThisKeyword
	KindSuperKeyword = ast.KindSuperKeyword
)

// FileName returns the path of the source file a node belongs to, so a caller
// that holds a node deep in a tree can report where it came from without walking
// to the root itself.
func FileName(n *Node) string {
	sf := ast.GetSourceFileOfNode(n)
	if sf == nil {
		return ""
	}
	return sf.FileName()
}

// ImportSpecifiers returns the raw module specifier strings of every import and
// re-export in a file, the edges a caller follows to discover the rest of a
// program's files. The strings are as written in the source, before resolution.
func ImportSpecifiers(file *SourceFile) []string {
	lits := file.Imports()
	out := make([]string, 0, len(lits))
	for _, lit := range lits {
		out = append(out, lit.Text())
	}
	return out
}
