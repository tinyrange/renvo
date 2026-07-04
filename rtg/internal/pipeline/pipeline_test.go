package pipeline

import (
	"testing"

	"j5.nz/rtg/rtg/internal/load"
	"j5.nz/rtg/rtgunit"
)

func TestBuildUnitLinksWorkspace(t *testing.T) {
	result := BuildUnit("/repo/case", "/std", "./cmd/app", []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import "example.com/case/pkg/lib"

func appMain() int { return lib.Value() }
`)},
		{Path: "/repo/case/pkg/lib/lib.go", Src: []byte(`package lib

const answer = 42

func Value() int { return answer }
`)},
	})
	if !result.Ok {
		t.Fatalf("BuildUnit failed: err=%d pkg=%d file=%d tok=%d", result.Error, result.ErrorPackage, result.ErrorFile, result.ErrorToken)
	}
	if len(result.Link.Data) == 0 {
		t.Fatal("BuildUnit returned empty linked unit data")
	}
	decoded, err := rtgunit.Unmarshal(result.Link.Data)
	if err != nil {
		t.Fatalf("linked unit did not decode: %v", err)
	}
	if decoded.Package != "main" {
		t.Fatalf("linked package = %q, want main", decoded.Package)
	}
	if len(decoded.Decls) != 1 || len(decoded.Funcs) != 2 {
		t.Fatalf("linked decls/funcs = %d/%d, want 1/2", len(decoded.Decls), len(decoded.Funcs))
	}
	if result.Build.Root != 1 || result.Build.Units[result.Build.Root].ImportPath != "example.com/case/cmd/app" {
		t.Fatalf("root unit = %d %#v", result.Build.Root, result.Build.Units)
	}
}

func TestBuildUnitReportsLoadError(t *testing.T) {
	result := BuildUnit("/repo/case", "/std", "./cmd/app", []load.SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nfunc appMain() int { return 0 }\n")},
	})
	if result.Ok || result.Error != PipelineErrLoad {
		t.Fatalf("missing module result = %#v", result)
	}
	if result.ErrorFile != -1 || result.ErrorPackage != -1 || result.ErrorToken != -1 {
		t.Fatalf("load error location = pkg %d file %d tok %d", result.ErrorPackage, result.ErrorFile, result.ErrorToken)
	}
}

func TestBuildUnitReportsBuildError(t *testing.T) {
	result := BuildUnit("/repo/case", "/std", "./cmd/app", []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/a.go", Src: []byte("package main\nvar value int\n")},
		{Path: "/repo/case/cmd/app/b.go", Src: []byte("package main\nfunc value() {}\n")},
	})
	if result.Ok || result.Error != PipelineErrBuild {
		t.Fatalf("duplicate symbol result = %#v", result)
	}
	if result.ErrorPackage != 0 || result.ErrorFile != 1 || result.ErrorToken < 0 {
		t.Fatalf("build error location = pkg %d file %d tok %d", result.ErrorPackage, result.ErrorFile, result.ErrorToken)
	}
}
