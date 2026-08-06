package shim

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/diagnostics"
	"github.com/microsoft/typescript-go/internal/module"
)

// The facade re-exports the compiler's own types as aliases rather than wrapping
// them. An alias is the identical type, so a caller holds the real *checker.Type
// and calls its real methods, with no copy, no adapter object, and no drift from
// upstream. The only thing this module adds is the ability to name these types
// from outside, which the internal/ boundary otherwise forbids.
type (
	// Type is a resolved type produced by the checker.
	Type = checker.Type
	// Symbol is a named program entity: a variable, a property, a function.
	Symbol = ast.Symbol
	// Signature is one call or construct signature of a type.
	Signature = checker.Signature
	// Node is a node in the syntax tree.
	Node = ast.Node
	// SourceFile is one parsed file.
	SourceFile = ast.SourceFile
	// Checker answers type questions about a program.
	Checker = checker.Checker
	// Diagnostic is one error or warning with its source location.
	Diagnostic = ast.Diagnostic
	// ResolvedModule is the result of resolving one import specifier.
	ResolvedModule = module.ResolvedModule
	// TupleElementExport is one positional element of a tuple type: its element
	// type, whether the position is optional or the rest tail, and its label.
	TupleElementExport = checker.TupleElementExport

	// TypeFlags classifies a type: number, string, object, union, and so on.
	TypeFlags = checker.TypeFlags
	// SymbolFlags classifies a symbol: property, method, class, optional.
	SymbolFlags = ast.SymbolFlags
	// Kind is a syntax node kind.
	Kind = ast.Kind
	// SignatureKind selects call or construct signatures.
	SignatureKind = checker.SignatureKind
	// DiagnosticCategory is the severity of a diagnostic.
	DiagnosticCategory = diagnostics.Category
)

// TypeFlags values, re-exported so a caller can classify a Type without importing
// internal/checker. The set is the one an out-of-module type mapper needs; more
// can be added the same way without touching upstream.
const (
	TypeFlagsAny            = checker.TypeFlagsAny
	TypeFlagsUnknown        = checker.TypeFlagsUnknown
	TypeFlagsString         = checker.TypeFlagsString
	TypeFlagsNumber         = checker.TypeFlagsNumber
	TypeFlagsBigInt         = checker.TypeFlagsBigInt
	TypeFlagsBoolean        = checker.TypeFlagsBoolean
	TypeFlagsESSymbol       = checker.TypeFlagsESSymbol
	TypeFlagsVoid           = checker.TypeFlagsVoid
	TypeFlagsUndefined      = checker.TypeFlagsUndefined
	TypeFlagsNull           = checker.TypeFlagsNull
	TypeFlagsNever          = checker.TypeFlagsNever
	TypeFlagsObject         = checker.TypeFlagsObject
	TypeFlagsUnion          = checker.TypeFlagsUnion
	TypeFlagsIntersection   = checker.TypeFlagsIntersection
	TypeFlagsEnum           = checker.TypeFlagsEnum
	TypeFlagsTypeParameter  = checker.TypeFlagsTypeParameter
	TypeFlagsStringLiteral  = checker.TypeFlagsStringLiteral
	TypeFlagsNumberLiteral  = checker.TypeFlagsNumberLiteral
	TypeFlagsBigIntLiteral  = checker.TypeFlagsBigIntLiteral
	TypeFlagsBooleanLiteral = checker.TypeFlagsBooleanLiteral
	TypeFlagsLiteral        = checker.TypeFlagsLiteral
	TypeFlagsUnit           = checker.TypeFlagsUnit
	TypeFlagsStringLike     = checker.TypeFlagsStringLike
	TypeFlagsNumberLike     = checker.TypeFlagsNumberLike
	TypeFlagsBooleanLike    = checker.TypeFlagsBooleanLike
	TypeFlagsEnumLike       = checker.TypeFlagsEnumLike
	TypeFlagsNonPrimitive   = checker.TypeFlagsNonPrimitive
)

// SymbolFlags values, re-exported for the same reason.
const (
	SymbolFlagsProperty   = ast.SymbolFlagsProperty
	SymbolFlagsFunction   = ast.SymbolFlagsFunction
	SymbolFlagsClass      = ast.SymbolFlagsClass
	SymbolFlagsInterface  = ast.SymbolFlagsInterface
	SymbolFlagsMethod     = ast.SymbolFlagsMethod
	SymbolFlagsOptional   = ast.SymbolFlagsOptional
	SymbolFlagsVariable   = ast.SymbolFlagsVariable
	SymbolFlagsTypeAlias  = ast.SymbolFlagsTypeAlias
	SymbolFlagsEnum       = ast.SymbolFlagsEnum
	SymbolFlagsEnumMember = ast.SymbolFlagsEnumMember
	SymbolFlagsAlias      = ast.SymbolFlagsAlias
	SymbolFlagsNamespace  = ast.SymbolFlagsNamespace
)

// SignatureKind values.
const (
	SignatureKindCall      = checker.SignatureKindCall
	SignatureKindConstruct = checker.SignatureKindConstruct
)

// DiagnosticCategory values.
const (
	DiagnosticCategoryError      = diagnostics.CategoryError
	DiagnosticCategoryWarning    = diagnostics.CategoryWarning
	DiagnosticCategorySuggestion = diagnostics.CategorySuggestion
	DiagnosticCategoryMessage    = diagnostics.CategoryMessage
)

// ResolutionModeNone is the unspecified module resolution mode, the default a
// caller passes when it does not track a file's own module mode.
const ResolutionModeNone = core.ResolutionModeNone
