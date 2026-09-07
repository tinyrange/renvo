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
