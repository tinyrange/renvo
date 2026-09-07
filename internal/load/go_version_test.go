package load

import "testing"

func TestModuleGoVersion(t *testing.T) {
	for _, version := range []string{"1.22", "1.26.0", "1.26rc1", "1.27beta2", "2.0", "1.999999999999999999999999"} {
		module := ParseModule("/repo", []byte("module example.com/app\ngo "+version+" // language version\n"))
		if !module.Ok || module.GoVersion != version {
			t.Fatalf("%s: %#v", version, module)
		}
	}
	module := ParseModule("/repo", []byte("module example.com/app\n"))
	if !module.Ok || module.GoVersion != "" {
		t.Fatalf("absent directive: %#v", module)
	}
}

func TestInvalidModuleGoVersion(t *testing.T) {
	for _, directive := range []string{"go", "go 1", "go go1.26", "go 0.26", "go 01.26", "go 1.026", "go 1.26.00", "go 1.26.", "go 1.26rc", "go 1.26rc1junk", "go 1.26\ngo 1.26", "go 1.26 extra", "go 1.26 /* comment */ extra"} {
		module := ParseModule("/repo", []byte("module example.com/app\n"+directive+"\n"))
		if module.Ok || module.Error != ModuleErrDirective {
			t.Fatalf("%s: %#v", directive, module)
		}
	}
}

func TestWorkspaceRetainsGoVersion(t *testing.T) {
	workspace := LoadWorkspace("/repo", "/std", ".", []SourceFile{
		{Path: "/repo/go.mod", Src: []byte("module example.com/app\ngo 1.26.0\n")},
		{Path: "/repo/main.go", Src: []byte("package main\nfunc main() {}\n")},
	})
	if !workspace.Ok || workspace.Module.GoVersion != "1.26.0" || workspace.Graph.Module.GoVersion != "1.26.0" {
		t.Fatalf("version lost during loading: %#v", workspace)
	}
}

func TestPackageGoVersionOwnership(t *testing.T) {
	module := Module{Path: "example.com/app", GoVersion: "1.26"}
	dependencies := []ModuleDependency{
		{Path: "example.com/lib", GoVersion: "1.20"},
		{Path: "example.com/lib/sub", GoVersion: "1.22"},
		{Path: "example.com/legacy"},
	}
	for _, test := range []struct {
		kind       int
		path, want string
	}{
		{PackageInModule, "example.com/app/pkg", "1.26"},
		{PackageDependency, "example.com/lib/pkg", "1.20"},
		{PackageDependency, "example.com/lib/sub/pkg", "1.22"},
		{PackageDependency, "example.com/legacy", ""},
		{PackageStandard, "fmt", ""},
	} {
		got := packageModuleGoVersion(module, PackageRef{Kind: test.kind, ImportPath: test.path}, dependencies)
		if got != test.want {
			t.Fatalf("%s: %q, want %q", test.path, got, test.want)
		}
	}
}

func TestLegacyDependencyManifest(t *testing.T) {
	workspace := LoadWorkspace("/repo/app", "/std", ".", []SourceFile{
		{Path: "/repo/app/go.mod", Src: []byte("module example.com/app\ngo 1.26\n")},
		{Path: "/repo/app/main.go", Src: []byte("package main\nimport \"example.com/lib\"\nfunc main(){_=lib.Value()}\n")},
		{Path: "/repo/lib/go.mod", Src: []byte("example.com/lib")},
		{Path: "/repo/lib/lib.go", Src: []byte("package lib\nfunc Value()int{return 1}\n")},
	})
	if !workspace.Ok || len(workspace.Graph.Packages) != 2 || workspace.Graph.Packages[0].GoVersion != "" {
		t.Fatalf("legacy manifest failed: %#v", workspace)
	}
}
