package shim_test

import (
	"testing"

	"github.com/microsoft/typescript-go/shim"
)

// TestNormalizePathTurnsSeparatorsAround pins the property an out-of-module caller
// needs: a Windows path with backslashes and a drive letter comes back slashed,
// with the drive letter kept.
func TestNormalizePathTurnsSeparatorsAround(t *testing.T) {
	cases := []struct{ in, want string }{
		{`C:\Users\x\main.ts`, "C:/Users/x/main.ts"},
		{`C:/Users/x/main.ts`, "C:/Users/x/main.ts"},
		{`C:\Users\x\..\y\main.ts`, "C:/Users/y/main.ts"},
		{"/home/x/main.ts", "/home/x/main.ts"},
		{"/home/x/./main.ts", "/home/x/main.ts"},
	}
	for _, c := range cases {
		if got := shim.NormalizePath(c.in); got != c.want {
			t.Errorf("NormalizePath(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestNormalizedPathsSatisfyTheFileMapPredicate pins the composition callers
// actually depend on: the keys of a shim.Compile file map must be rooted and must
// be unchanged by normalization with the trailing separator removed, which is the
// check vfstest.FromMap makes before it panics.
func TestNormalizedPathsSatisfyTheFileMapPredicate(t *testing.T) {
	for _, in := range []string{`C:\Users\x\main.ts`, `C:\Users\x\`, "/home/x/main.ts", "/home/x/"} {
		p := shim.RemoveTrailingDirectorySeparator(shim.NormalizePath(in))
		if !shim.IsRootedDiskPath(p) {
			t.Errorf("%q normalized to %q, which is not rooted", in, p)
		}
		if again := shim.RemoveTrailingDirectorySeparator(shim.NormalizePath(p)); again != p {
			t.Errorf("%q normalized to %q, which does not normalize to itself but to %q", in, p, again)
		}
	}
}
