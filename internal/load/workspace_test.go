package load

import "testing"

func TestLoadWorkspaceRootPackage(t *testing.T) {
	workspace := LoadWorkspace("/repo/case", "/std", "./cmd/app", []SourceFile{
		{Path: "/repo/case/pkg/lib/lib.go", Src: []byte("package lib\nfunc Value() int { return 42 }\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import "example.com/case/pkg/lib"

func appMain() int { return lib.Value() }
`)},
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n\ngo 1.25\n")},
	})
	if !workspace.Ok {
		t.Fatalf("LoadWorkspace failed: err=%d file=%d graph=%#v", workspace.Error, workspace.ErrorFile, workspace.Graph)
	}
	if workspace.Module.Root != "/repo/case" || workspace.Module.Path != "example.com/case" {
		t.Fatalf("module = %#v", workspace.Module)
	}
	if len(workspace.Graph.Packages) != 2 {
		t.Fatalf("package count = %d, want 2", len(workspace.Graph.Packages))
	}
	if workspace.Graph.Packages[0].Ref.ImportPath != "example.com/case/pkg/lib" {
		t.Fatalf("dependency package = %#v", workspace.Graph.Packages[0].Ref)
	}
	if workspace.Graph.Packages[1].Ref.ImportPath != "example.com/case/cmd/app" {
		t.Fatalf("root package = %#v", workspace.Graph.Packages[1].Ref)
	}
}

func TestLoadWorkspaceFromSubdirectory(t *testing.T) {
	workspace := LoadWorkspace("/repo/case/cmd", "/std", "./app", []SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nfunc appMain() int { return 0 }\n")},
	})
	if !workspace.Ok {
		t.Fatalf("LoadWorkspace failed: err=%d file=%d", workspace.Error, workspace.ErrorFile)
	}
	if workspace.Graph.Root != "example.com/case/cmd/app" {
		t.Fatalf("graph root = %q", workspace.Graph.Root)
	}
	if len(workspace.Graph.Packages) != 1 || workspace.Graph.Packages[0].Ref.Dir != "/repo/case/cmd/app" {
		t.Fatalf("packages = %#v", workspace.Graph.Packages)
	}
}

func TestLoadWorkspaceNestedModule(t *testing.T) {
	workspace := LoadWorkspace("/repo/outer/inner/cmd/app", "/std", ".", []SourceFile{
		{Path: "/repo/outer/go.mod", Src: []byte("module example.com/outer\n")},
		{Path: "/repo/outer/inner/go.mod", Src: []byte("module example.com/inner\n")},
		{Path: "/repo/outer/inner/cmd/app/main.go", Src: []byte("package main\nfunc appMain() int { return 0 }\n")},
	})
	if !workspace.Ok {
		t.Fatalf("LoadWorkspace failed: err=%d file=%d", workspace.Error, workspace.ErrorFile)
	}
	if workspace.Module.Root != "/repo/outer/inner" || workspace.Module.Path != "example.com/inner" {
		t.Fatalf("module = %#v", workspace.Module)
	}
	if workspace.Graph.Root != "example.com/inner/cmd/app" {
		t.Fatalf("graph root = %q", workspace.Graph.Root)
	}
}

func TestLoadWorkspaceNormalizesFiles(t *testing.T) {
	workspace := LoadWorkspace("/repo/case", "/std", "./cmd/app", []SourceFile{
		{Path: "/repo/case/cmd/app/../app/main.go", Src: []byte("package main\nfunc appMain() int { return 0 }\n")},
		{Path: "/repo/case/./go.mod", Src: []byte("module example.com/case\n")},
	})
	if !workspace.Ok {
		t.Fatalf("LoadWorkspace failed: err=%d file=%d", workspace.Error, workspace.ErrorFile)
	}
	if len(workspace.Files) != 2 {
		t.Fatalf("file count = %d, want 2", len(workspace.Files))
	}
	if workspace.Files[0].Path != "/repo/case/cmd/app/main.go" || workspace.Files[1].Path != "/repo/case/go.mod" {
		t.Fatalf("normalized files = %#v", workspace.Files)
	}
}

func TestLoadWorkspaceErrors(t *testing.T) {
	duplicate := LoadWorkspace("/repo/case", "/std", "./cmd/app", []SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/./go.mod", Src: []byte("module example.com/case\n")},
	})
	if duplicate.Ok || duplicate.Error != WorkspaceErrDuplicateFile {
		t.Fatalf("duplicate workspace = %#v", duplicate)
	}

	missing := LoadWorkspace("/repo/case", "/std", "./cmd/app", []SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n")},
	})
	if missing.Ok || missing.Error != WorkspaceErrMissingModule {
		t.Fatalf("missing module workspace = %#v", missing)
	}

	badModule := LoadWorkspace("/repo/case", "/std", "./cmd/app", []SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n")},
	})
	if badModule.Ok || badModule.Error != WorkspaceErrModule {
		t.Fatalf("bad module workspace = %#v", badModule)
	}

	badGraph := LoadWorkspace("/repo/case", "/std", "./cmd/app", []SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
	})
	if badGraph.Ok || badGraph.Error != WorkspaceErrGraph {
		t.Fatalf("bad graph workspace = %#v", badGraph)
	}
}
