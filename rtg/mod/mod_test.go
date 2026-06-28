package mod

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseModulePath(t *testing.T) {
	got, err := ParseModulePath("\n// comment\nmodule example.com/app // tail\n\ngo 1.25\n")
	if err != nil {
		t.Fatalf("ParseModulePath failed: %v", err)
	}
	if got != "example.com/app" {
		t.Fatalf("module path = %q, want example.com/app", got)
	}
}

func TestFindRelativeSubdirectoryUsesNearestModule(t *testing.T) {
	root := t.TempDir()
	writeModTestFile(t, root, "go.mod", "module example.com/outer\n")
	writeModTestFile(t, root, "nested/go.mod", "module example.com/nested\n")
	writeModTestFile(t, root, "nested/cmd/app/main.go", "package main\n")
	t.Chdir(root)

	got, err := Find("./nested/cmd/app")
	if err != nil {
		t.Fatalf("Find failed: %v", err)
	}
	if got.Path != "example.com/nested" {
		t.Fatalf("module path = %q, want example.com/nested", got.Path)
	}
	if got.Root != filepath.Join(root, "nested") {
		t.Fatalf("module root = %q, want nested root", got.Root)
	}
}

func writeModTestFile(t *testing.T, root string, name string, data string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
}

func TestParseModulePathMissing(t *testing.T) {
	if _, err := ParseModulePath("go 1.25\n"); err == nil {
		t.Fatalf("ParseModulePath succeeded without module directive")
	}
}

func TestParseFileRejectsMalformedModuleDirective(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{name: "missing path", data: `module
`},
		{name: "extra field", data: `module example.com/app extra
`},
		{name: "duplicate", data: `module example.com/app
module example.com/other
`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseFile(tt.data)
			if err == nil {
				t.Fatalf("ParseFile accepted malformed module directive %s", tt.name)
			}
			if err.Error() != "malformed module directive" {
				t.Fatalf("error = %q", err)
			}
		})
	}
}

func TestParseFileReplaces(t *testing.T) {
	module, err := ParseFile(`module example.com/app

replace example.com/lib => ../lib
replace example.com/other v1.2.3 => ./other
replace example.com/external => example.com/fork v1.2.3

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
		{Old: "example.com/external", New: "example.com/fork"},
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

func TestParseFileRequires(t *testing.T) {
	module, err := ParseFile(`module example.com/app

require example.com/lib v1.2.3

require (
	example.com/one v0.1.0
	example.com/two v2.0.0
)
`)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	want := []Require{
		{Path: "example.com/lib", Version: "v1.2.3"},
		{Path: "example.com/one", Version: "v0.1.0"},
		{Path: "example.com/two", Version: "v2.0.0"},
	}
	if len(module.Requires) != len(want) {
		t.Fatalf("requires = %#v, want %#v", module.Requires, want)
	}
	for i := range want {
		if module.Requires[i] != want[i] {
			t.Fatalf("require %d = %#v, want %#v", i, module.Requires[i], want[i])
		}
	}
}

func TestParseFileUnquotesDirectiveFields(t *testing.T) {
	module, err := ParseFile(`module "example.com/app"

require "example.com/lib" "v1.2.3"
replace "example.com/lib" => "../lib"
replace "example.com/other" "v1.0.0" => "./other"
replace "example.com/spaced" => "../my lib"
`)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	if module.Path != "example.com/app" {
		t.Fatalf("module path = %q, want example.com/app", module.Path)
	}
	if len(module.Requires) != 1 || module.Requires[0] != (Require{Path: "example.com/lib", Version: "v1.2.3"}) {
		t.Fatalf("requires = %#v", module.Requires)
	}
	want := []Replace{
		{Old: "example.com/lib", New: "../lib"},
		{Old: "example.com/other", New: "./other"},
		{Old: "example.com/spaced", New: "../my lib"},
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

func TestParseFileStripsBlockComments(t *testing.T) {
	module, err := ParseFile(`/* leading */
module example.com/app /* tail */

require (
	/* before */ example.com/one v0.1.0
	/*
		middle
	*/
	example.com/two v2.0.0 /* after */
)

replace (
	example.com/one => ./one /* local one */
	/* comment */ example.com/two v2.0.0 => ../two
)
`)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	if module.Path != "example.com/app" {
		t.Fatalf("module path = %q, want example.com/app", module.Path)
	}
	wantRequires := []Require{
		{Path: "example.com/one", Version: "v0.1.0"},
		{Path: "example.com/two", Version: "v2.0.0"},
	}
	if len(module.Requires) != len(wantRequires) {
		t.Fatalf("requires = %#v, want %#v", module.Requires, wantRequires)
	}
	for i := range wantRequires {
		if module.Requires[i] != wantRequires[i] {
			t.Fatalf("require %d = %#v, want %#v", i, module.Requires[i], wantRequires[i])
		}
	}
	wantReplaces := []Replace{
		{Old: "example.com/one", New: "./one"},
		{Old: "example.com/two", New: "../two"},
	}
	if len(module.Replaces) != len(wantReplaces) {
		t.Fatalf("replaces = %#v, want %#v", module.Replaces, wantReplaces)
	}
	for i := range wantReplaces {
		if module.Replaces[i] != wantReplaces[i] {
			t.Fatalf("replace %d = %#v, want %#v", i, module.Replaces[i], wantReplaces[i])
		}
	}
}

func TestParseFileRejectsUnterminatedBlockComment(t *testing.T) {
	_, err := ParseFile(`module example.com/app

/* unterminated
require example.com/lib v1.2.3
`)
	if err == nil {
		t.Fatalf("ParseFile accepted unterminated block comment")
	}
	if err.Error() != "malformed comment" {
		t.Fatalf("error = %q", err)
	}
}

func TestParseFileRejectsMalformedRequires(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{name: "line", data: `module example.com/app

require example.com/lib
`},
		{name: "block", data: `module example.com/app

require (
	example.com/lib
)
`},
		{name: "too many fields", data: `module example.com/app

require example.com/lib v1.0.0 extra
`},
		{name: "bad block opener", data: `module example.com/app

require ( extra
`},
		{name: "unterminated block", data: `module example.com/app

require (
	example.com/lib v1.0.0
`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseFile(tt.data)
			if err == nil {
				t.Fatalf("ParseFile accepted malformed require %s", tt.name)
			}
			if err.Error() != "malformed require directive" {
				t.Fatalf("error = %q", err)
			}
		})
	}
}

func TestParseFileRejectsMalformedReplaces(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{name: "line", data: `module example.com/app

replace example.com/lib ../lib
`},
		{name: "block", data: `module example.com/app

replace (
	example.com/lib ../lib
)
`},
		{name: "missing target", data: `module example.com/app

replace example.com/lib =>
`},
		{name: "too many target fields", data: `module example.com/app

replace example.com/lib => ../lib extra
`},
		{name: "local target version", data: `module example.com/app

replace example.com/lib => ../lib v1.2.3
`},
		{name: "bad block opener", data: `module example.com/app

replace ( extra
`},
		{name: "unterminated block", data: `module example.com/app

replace (
	example.com/lib => ../lib
`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseFile(tt.data)
			if err == nil {
				t.Fatalf("ParseFile accepted malformed replace %s", tt.name)
			}
			if err.Error() != "malformed replace directive" {
				t.Fatalf("error = %q", err)
			}
		})
	}
}
