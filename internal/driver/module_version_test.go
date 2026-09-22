package driver

import (
	"testing"

	"renvo.dev/internal/load"
)

func TestCollectedDependencyGoVersions(t *testing.T) {
	fs := memorySourceFS{files: []load.SourceFile{
		{Path: "/repo/app/go.mod", Src: []byte("module example.com/app\ngo 1.26\nrequire example.com/lib v1.0.0\nreplace example.com/lib => ../lib\nreplace example.com/value => ../value\n")},
		{Path: "/repo/app/main.go", Src: []byte("package main\nimport \"example.com/lib\"\nfunc main(){_=lib.Value()}\n")},
		{Path: "/repo/lib/go.mod", Src: []byte("module example.com/replacement\ngo 1.22.3\nrequire example.com/value v1.0.0\n")},
		{Path: "/repo/lib/lib.go", Src: []byte("package lib\nimport \"example.com/value\"\nfunc Value()int{return value.Number()}\n")},
		{Path: "/repo/value/go.mod", Src: []byte("module example.com/value\ngo 1.23rc1\n")},
		{Path: "/repo/value/value.go", Src: []byte("package value\nfunc Number()int{return 42}\n")},
	}}
	collected := CollectSources("/repo/app", "/std", ".", fs)
	if !collected.Ok {
		t.Fatalf("collection failed: %#v", collected)
	}
	workspace := load.LoadWorkspace("/repo/app", "/std", ".", collected.Files)
	if !workspace.Ok {
		t.Fatalf("workspace failed: %#v", workspace)
	}
	want := map[string]string{"example.com/app": "1.26", "example.com/lib": "1.22.3", "example.com/value": "1.23rc1"}
	if len(workspace.Graph.Packages) != len(want) {
		t.Fatalf("packages: %#v", workspace.Graph.Packages)
	}
	for _, pkg := range workspace.Graph.Packages {
		version, ok := want[pkg.Ref.ImportPath]
		if !ok || pkg.GoVersion != version {
			t.Fatalf("%s version %q, want %q", pkg.Ref.ImportPath, pkg.GoVersion, version)
		}
	}
}
