package shim

import "github.com/microsoft/typescript-go/internal/tspath"

// The compiler has one idea of what a path looks like and it is not the operating
// system's. Every path it holds is slash-separated, on Windows as much as on
// Unix, so a Windows path is spelled "C:/Users/x/main.ts" with the drive letter
// kept and the separators turned around. Handing it anything else is not a
// tolerated variation: vfstest.FromMap panics on a path that is not in this form,
// and the module resolver compares paths by bytes.
//
// A caller outside this module has to be able to produce that form, and the only
// correct way to produce it is with the compiler's own code, because "normalized"
// here means exactly what NormalizePath does and nothing else. So these three are
// re-exported, in the same additive spirit as the type aliases: the facade names
// the compiler's functions, it does not restate them.

// NormalizePath turns a path into the compiler's canonical form: separators become
// slashes, and "." and ".." segments are resolved away. A Windows path keeps its
// drive letter and loses its backslashes.
func NormalizePath(path string) string { return tspath.NormalizePath(path) }

// IsRootedDiskPath reports whether a path is absolute in the compiler's sense: it
// starts at "/" or at a DOS root like "c:", "c:/" or "c:\".
func IsRootedDiskPath(path string) bool { return tspath.IsRootedDiskPath(path) }

// RemoveTrailingDirectorySeparator drops one trailing slash, which the canonical
// form does not carry. Combined with NormalizePath this is the exact predicate
// vfstest.FromMap applies to the keys of a file map.
func RemoveTrailingDirectorySeparator(path string) string {
	return tspath.RemoveTrailingDirectorySeparator(path)
}
