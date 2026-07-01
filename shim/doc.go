// Package shim is a public, in-process facade over the typescript-go compiler.
//
// Every compiler package in this module lives under internal/, so a program in
// another module cannot import the checker, the binder, or the AST directly.
// This package sits inside the module, so it may import those internal packages,
// and it re-exports the small surface an out-of-module caller needs to compile
// TypeScript in memory and read the checker's answers: the resolved types of
// nodes, the members and signatures of a type, the literal values, the AST, and
// the diagnostics.
//
// The facade is deliberately additive. It adds one new top-level directory and
// touches no existing file, so a rebase onto a newer upstream is a fast-forward
// of everything under internal/ with this directory carried along unchanged. The
// heavy lifting stays in the real compiler; this package only names it.
package shim
