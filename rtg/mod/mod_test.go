package mod

import "testing"

func TestParseModulePath(t *testing.T) {
	got, err := ParseModulePath("\n// comment\nmodule example.com/app // tail\n\ngo 1.25\n")
	if err != nil {
		t.Fatalf("ParseModulePath failed: %v", err)
	}
	if got != "example.com/app" {
		t.Fatalf("module path = %q, want example.com/app", got)
	}
}

func TestParseModulePathMissing(t *testing.T) {
	if _, err := ParseModulePath("go 1.25\n"); err == nil {
		t.Fatalf("ParseModulePath succeeded without module directive")
	}
}

func TestParseFileReplaces(t *testing.T) {
	module, err := ParseFile(`module example.com/app

replace example.com/lib => ../lib
replace example.com/other v1.2.3 => ./other

replace (
	example.com/block => ./block
	example.com/versioned v1.0.0 => ../versioned
)
`)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	if module.Path != "example.com/app" {
		t.Fatalf("module path = %q, want example.com/app", module.Path)
	}
	want := []Replace{
		{Old: "example.com/lib", New: "../lib"},
		{Old: "example.com/other", New: "./other"},
		{Old: "example.com/block", New: "./block"},
		{Old: "example.com/versioned", New: "../versioned"},
	}
	if len(module.Replaces) != len(want) {
		t.Fatalf("replaces = %#v, want %#v", module.Replaces, want)
	}
	for i := range want {
		if module.Replaces[i] != want[i] {
			t.Fatalf("replace %d = %#v, want %#v", i, module.Replaces[i], want[i])
		}
	}
}
