package shim

import "testing"

func earlyErrs(t *testing.T, src string) []EarlyModuleError {
	t.Helper()
	p := Compile(map[string]string{"/src/main.ts": src}, Options{Loose: true})
	defer p.Close()
	for _, f := range p.SourceFiles() {
		if f.FileName() == "/src/main.ts" {
			return EarlyModuleErrors(f)
		}
	}
	t.Fatal("source file /src/main.ts not found")
	return nil
}

func TestEarlyModuleErrors_Detects(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{"default eval", `import eval from "./m";`},
		{"default arguments", `import arguments from "./m";`},
		{"named eval", `import { eval } from "./m";`},
		{"named as eval", `import { x as eval } from "./m";`},
		{"named as arguments", `import { x as arguments } from "./m";`},
		{"namespace eval", `import * as eval from "./m";`},
		{"namespace arguments", `import * as arguments from "./m";`},
		{"dup attr plain", `import x from "./m" with { type: "json", type: "js" };`},
		{"dup attr escaped", `import x from "./m" with { "type": "json", type: "js" };`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := earlyErrs(t, tc.src); len(got) == 0 {
				t.Fatalf("expected an early error for %q, got none", tc.src)
			}
		})
	}
}

func TestEarlyModuleErrors_NoFalsePositive(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{"plain named import", `import { x } from "./m";`},
		{"rename to safe local", `import { eval as x } from "./m";`},
		{"default safe", `import def from "./m";`},
		{"namespace safe", `import * as ns from "./m";`},
		// export binds no local, so `as eval` here is legal.
		{"export as eval decoy", `const x = 1; export { x as eval };`},
		{"export from as eval decoy", `export { x as eval } from "./m";`},
		{"distinct attrs", `import x from "./m" with { type: "json", other: "js" };`},
		{"single attr", `import x from "./m" with { type: "json" };`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := earlyErrs(t, tc.src); len(got) != 0 {
				t.Fatalf("expected no early error for %q, got %+v", tc.src, got)
			}
		})
	}
}
