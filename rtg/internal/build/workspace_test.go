package build

import (
	"testing"

	"j5.nz/rtg/rtg/internal/load"
	"j5.nz/rtg/rtgunit"
)

func TestBuildUnitsFromWorkspace(t *testing.T) {
	workspace := load.LoadWorkspace("/repo/case", "/std", "./cmd/app", []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import "example.com/case/pkg/lib"

func appMain() int { return lib.Value() }
`)},
		{Path: "/repo/case/pkg/lib/lib.go", Src: []byte(`package lib

func Value() int { return 42 }
`)},
	})
	if !workspace.Ok {
		t.Fatalf("LoadWorkspace failed: err=%d file=%d", workspace.Error, workspace.ErrorFile)
	}
	result := BuildUnits(workspace.Graph)
	if !result.Ok {
		t.Fatalf("BuildUnits failed: err=%d pkg=%d file=%d tok=%d", result.Error, result.ErrorPackage, result.ErrorFile, result.ErrorToken)
	}
	root := RootUnit(result)
	if root.ImportPath != "example.com/case/cmd/app" {
		t.Fatalf("root = %#v", root)
	}
	decoded, err := rtgunit.Unmarshal(marshalPackageUnit(t, root))
	if err != nil {
		t.Fatalf("root unit decode failed: %v", err)
	}
	if decoded.Package != "main" || len(decoded.Funcs) != 1 {
		t.Fatalf("decoded root = package %q funcs %d", decoded.Package, len(decoded.Funcs))
	}
}
