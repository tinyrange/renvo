package driver

import (
	"renvo.dev/internal/load"
	"testing"
)

// A required parent module must not steal packages from the more-specific main
// module. A more-specific required child still overrides the main module.
func TestParentRequirementKeepsMainModuleOwnership(t *testing.T) {
	fs := memorySourceFS{files: []load.SourceFile{
		{Path: "/repo/app/go.mod", Src: []byte("module example.com/parent/app\nrequire example.com/parent v1.0.0\nreplace example.com/parent => ../parent\n")},
		{Path: "/repo/app/main.go", Src: []byte("package main\nimport (\"example.com/parent/app/local\"; \"example.com/parent/support\")\nfunc appMain() int {return local.Value()+support.Value()}\n")},
		{Path: "/repo/app/local/value.go", Src: []byte("package local\nfunc Value() int {return 19}\n")},
		{Path: "/repo/parent/go.mod", Src: []byte("module example.com/parent\n")},
		{Path: "/repo/parent/support/value.go", Src: []byte("package support\nfunc Value() int {return 23}\n")},
	}}
	result := BuildFromFS([]string{"-o", "app", "."}, "/repo/app", "/std", fs)
	if !result.Ok {
		t.Fatalf("parent requirement shadowed main module: %#v", result.Diagnostic)
	}
	sources := CollectSources("/repo/app", "/std", ".", fs)
	if !sources.Ok {
		t.Fatal(sources)
	}
	assertSourcePaths(t, sources.Files, []string{"/repo/app/go.mod", "/repo/app/main.go", "/repo/app/local/value.go", "/repo/parent/go.mod", "/repo/parent/support/value.go"})
}
