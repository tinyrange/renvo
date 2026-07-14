package driver

import (
	"testing"

	"j5.nz/rtg/rtg/internal/load"
	"j5.nz/rtg/rtg/internal/pipeline"
	"j5.nz/rtg/rtgunit"
)

func TestBuildUnitFromDriverOptions(t *testing.T) {
	result := BuildUnit([]string{"-t", "linux/386", "-s", "-o", "app", "./cmd/app"}, "/repo/case", "/std", driverTestFiles())
	if !result.Ok {
		t.Fatalf("BuildUnit failed: err=%d arg=%q at=%d pkg=%d file=%d tok=%d", result.Error, result.ErrorArg, result.ErrorAt, result.ErrorPackage, result.ErrorFile, result.ErrorToken)
	}
	if result.Options.Target != "linux/386" || result.Options.Output != "app" || !result.Options.Strip {
		t.Fatalf("options = %#v", result.Options)
	}
	if len(result.Unit) == 0 {
		t.Fatal("BuildUnit returned empty linked unit")
	}
	decoded, err := rtgunit.Unmarshal(result.Unit)
	if err != nil {
		t.Fatalf("linked unit did not decode: %v", err)
	}
	if decoded.Package != "main" || len(decoded.Funcs) != 2 {
		t.Fatalf("decoded unit = package %q funcs %d", decoded.Package, len(decoded.Funcs))
	}
}

func TestBuildUnitFiltersBuildTaggedFiles(t *testing.T) {
	result := BuildUnit([]string{"-t", "linux/aarch64", "-o", "app", "./cmd/app"}, "/repo/case", "/std", []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/host.go", Src: []byte(`//go:build !rtg

package main

var value int
`)},
		{Path: "/repo/case/cmd/app/rtg.go", Src: []byte(`//go:build rtg && linux && arm64

package main

func value() int { return 7 }

func appMain() int { return value() }
`)},
	})
	if !result.Ok {
		t.Fatalf("BuildUnit failed: err=%d arg=%q at=%d pkg=%d file=%d tok=%d", result.Error, result.ErrorArg, result.ErrorAt, result.ErrorPackage, result.ErrorFile, result.ErrorToken)
	}
	decoded, err := rtgunit.Unmarshal(result.Unit)
	if err != nil {
		t.Fatalf("linked unit did not decode: %v", err)
	}
	if len(decoded.Decls) != 0 || len(decoded.Funcs) != 2 {
		t.Fatalf("decoded unit decls=%d funcs=%d, want host decl skipped and two funcs", len(decoded.Decls), len(decoded.Funcs))
	}
}

func TestBuildUnitReportsMalformedBuildConstraint(t *testing.T) {
	result := BuildUnit([]string{"-o", "app", "./cmd/app"}, "/repo/case", "/std", []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte("//go:build linux &&\n\npackage main\n")},
	})
	if result.Ok || result.Error != BuildErrSource {
		t.Fatalf("malformed build result = %#v", result)
	}
	if result.Sources.Error != SourceErrBuildConstraint || result.ErrorPath != "/repo/case/cmd/app/main.go" {
		t.Fatalf("malformed build location = %#v", result)
	}
}

func TestBuildUnitReportsOptionError(t *testing.T) {
	result := BuildUnit([]string{"-t", "plan9/amd64", "-o", "app", "./cmd/app"}, "/repo/case", "/std", driverTestFiles())
	if result.Ok || result.Error != BuildErrOptions {
		t.Fatalf("bad option result = %#v", result)
	}
	if result.ErrorArg != "plan9/amd64" || result.ErrorAt != 1 {
		t.Fatalf("option location = arg %q at %d", result.ErrorArg, result.ErrorAt)
	}
	if result.ErrorPackage != -1 || result.ErrorFile != -1 || result.ErrorToken != -1 {
		t.Fatalf("option source location = pkg %d file %d tok %d", result.ErrorPackage, result.ErrorFile, result.ErrorToken)
	}
}

func TestBuildUnitReportsPipelineError(t *testing.T) {
	result := BuildUnit([]string{"-o", "app", "./cmd/app"}, "/repo/case", "/std", []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/a.go", Src: []byte("package main\nvar value int\n")},
		{Path: "/repo/case/cmd/app/b.go", Src: []byte("package main\nfunc value() {}\n")},
	})
	if result.Ok || result.Error != BuildErrPipeline {
		t.Fatalf("bad pipeline result = %#v", result)
	}
	if result.Pipeline.Error != pipeline.PipelineErrBuild {
		t.Fatalf("pipeline error = %d, want build", result.Pipeline.Error)
	}
	if result.ErrorPackage != 0 || result.ErrorFile != 1 || result.ErrorToken < 0 {
		t.Fatalf("pipeline location = pkg %d file %d tok %d", result.ErrorPackage, result.ErrorFile, result.ErrorToken)
	}
}

func driverTestFiles() []load.SourceFile {
	return []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import "example.com/case/pkg/lib"

func appMain() int { return lib.Value() }
`)},
		{Path: "/repo/case/pkg/lib/lib.go", Src: []byte(`package lib

func Value() int { return 42 }
`)},
	}
}
