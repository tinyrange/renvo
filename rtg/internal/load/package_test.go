package load

import "testing"

func TestLoadPackageSortsFilesAndIgnoresTests(t *testing.T) {
	mod := Module{Root: "/repo/case", Path: "example.com/case", Ok: true}
	ref := ResolvePackageArg(mod, "/repo/case", "./cmd/app")
	files := []SourceFile{
		{Path: "/repo/case/cmd/app/z.go", Src: []byte("package main\nfunc z() {}\n")},
		{Path: "/repo/case/cmd/app/a_test.go", Src: []byte("package other\n")},
		{Path: "/repo/case/pkg/lib/lib.go", Src: []byte("package lib\n")},
		{Path: "/repo/case/cmd/app/a.go", Src: []byte("package main\nfunc a() {}\n")},
	}
	pkg := LoadPackage(mod, "/std", ref, files)
	if !pkg.Ok {
		t.Fatalf("LoadPackage failed: err=%d file=%d import=%d", pkg.Error, pkg.ErrorFile, pkg.ErrorImport)
	}
	if pkg.Name != "main" {
		t.Fatalf("package name = %q, want main", pkg.Name)
	}
	if len(pkg.Files) != 2 {
		t.Fatalf("file count = %d, want 2", len(pkg.Files))
	}
	if pkg.Files[0].Path != "/repo/case/cmd/app/a.go" || pkg.Files[1].Path != "/repo/case/cmd/app/z.go" {
		t.Fatalf("file order = %q %q", pkg.Files[0].Path, pkg.Files[1].Path)
	}
}

func TestLoadGraphLocalPackages(t *testing.T) {
	mod := Module{Root: "/repo/case", Path: "example.com/case", Ok: true}
	files := []SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import "example.com/case/pkg/lib"

func main() { print(lib.Text()) }
`)},
		{Path: "/repo/case/pkg/lib/lib.go", Src: []byte(`package lib

import "example.com/case/pkg/util"

func Value() int { return util.Value() + extra() }
`)},
		{Path: "/repo/case/pkg/lib/extra.go", Src: []byte(`package lib

func extra() int { return 1 }
`)},
		{Path: "/repo/case/pkg/util/util.go", Src: []byte(`package util

func Value() int { return 2 }
`)},
	}
	graph := LoadGraph(mod, "/std", "/repo/case", "./cmd/app", files)
	if !graph.Ok {
		t.Fatalf("LoadGraph failed: err=%d pkg=%d graph=%#v", graph.Error, graph.ErrorPackage, graph)
	}
	want := []string{"example.com/case/pkg/util", "example.com/case/pkg/lib", "example.com/case/cmd/app"}
	if len(graph.Packages) != len(want) {
		t.Fatalf("package count = %d, want %d", len(graph.Packages), len(want))
	}
	for i := 0; i < len(want); i++ {
		if graph.Packages[i].Ref.ImportPath != want[i] {
			t.Fatalf("package %d = %q, want %q", i, graph.Packages[i].Ref.ImportPath, want[i])
		}
	}
	lib := graph.Packages[1]
	if len(lib.Files) != 2 || lib.Files[0].Path != "/repo/case/pkg/lib/extra.go" || lib.Files[1].Path != "/repo/case/pkg/lib/lib.go" {
		t.Fatalf("lib files = %#v", lib.Files)
	}
}

func TestLoadGraphStandardPackage(t *testing.T) {
	mod := Module{Root: "/repo/case", Path: "example.com/case", Ok: true}
	files := []SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import "runtime"

func main() { runtime.KeepAlive(0) }
`)},
		{Path: "/std/runtime/runtime.go", Src: []byte(`package runtime

func KeepAlive(v int) {}
`)},
	}
	graph := LoadGraph(mod, "/std", "/repo/case", "./cmd/app", files)
	if !graph.Ok {
		t.Fatalf("LoadGraph failed: err=%d pkg=%d graph=%#v", graph.Error, graph.ErrorPackage, graph)
	}
	if len(graph.Packages) != 2 {
		t.Fatalf("package count = %d, want 2", len(graph.Packages))
	}
	if graph.Packages[0].Ref.Kind != PackageStandard || graph.Packages[0].Ref.ImportPath != "runtime" {
		t.Fatalf("std package = %#v", graph.Packages[0].Ref)
	}
	if graph.Packages[1].Ref.ImportPath != "example.com/case/cmd/app" {
		t.Fatalf("root package = %#v", graph.Packages[1].Ref)
	}
}

func TestLoadPackageErrors(t *testing.T) {
	mod := Module{Root: "/repo/case", Path: "example.com/case", Ok: true}
	ref := ResolvePackageArg(mod, "/repo/case", "./cmd/app")

	missing := LoadPackage(mod, "/std", ref, nil)
	if missing.Ok || missing.Error != PackageErrNoFiles {
		t.Fatalf("missing package = %#v", missing)
	}

	mismatch := LoadPackage(mod, "/std", ref, []SourceFile{
		{Path: "/repo/case/cmd/app/a.go", Src: []byte("package main\n")},
		{Path: "/repo/case/cmd/app/b.go", Src: []byte("package app\n")},
	})
	if mismatch.Ok || mismatch.Error != PackageErrName || mismatch.ErrorFile != 1 {
		t.Fatalf("mismatched package = %#v", mismatch)
	}

	badSyntax := LoadPackage(mod, "/std", ref, []SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nfunc f()\n")},
	})
	if badSyntax.Ok || badSyntax.Error != PackageErrParse {
		t.Fatalf("bad syntax package = %#v", badSyntax)
	}
}

func TestLoadGraphErrors(t *testing.T) {
	mod := Module{Root: "/repo/case", Path: "example.com/case", Ok: true}

	missingImport := LoadGraph(mod, "/std", "/repo/case", "./cmd/app", []SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import "example.com/case/pkg/lib"
`)},
	})
	if missingImport.Ok || missingImport.Error != GraphErrPackage {
		t.Fatalf("missing import graph = %#v", missingImport)
	}

	cycle := LoadGraph(mod, "/std", "/repo/case", "./cmd/app", []SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import "example.com/case/pkg/lib"
`)},
		{Path: "/repo/case/pkg/lib/lib.go", Src: []byte(`package lib

import "example.com/case/cmd/app"
`)},
	})
	if cycle.Ok || cycle.Error != GraphErrCycle {
		t.Fatalf("cycle graph = %#v", cycle)
	}
}
