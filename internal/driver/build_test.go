package driver

import (
	"strings"
	"testing"

	"renvo.dev/backend/unit"
	frontendbuild "renvo.dev/internal/build"
	"renvo.dev/internal/load"
	"renvo.dev/internal/pipeline"
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
	decoded, err := unit.Unmarshal(result.Unit)
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
		{Path: "/repo/case/cmd/app/host.go", Src: []byte(`//go:build !renvo

package main

var value int
`)},
		{Path: "/repo/case/cmd/app/renvo.go", Src: []byte(`//go:build renvo && linux && arm64

package main

func value() int { return 7 }

func appMain() int { return value() }
`)},
	})
	if !result.Ok {
		t.Fatalf("BuildUnit failed: err=%d arg=%q at=%d pkg=%d file=%d tok=%d", result.Error, result.ErrorArg, result.ErrorAt, result.ErrorPackage, result.ErrorFile, result.ErrorToken)
	}
	decoded, err := unit.Unmarshal(result.Unit)
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

func TestBuildUnitReportsStructuredReturnDiagnostic(t *testing.T) {
	result := BuildUnit([]string{"-o", "app", "./cmd/app"}, "/repo/case", "/std", []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n\nfunc broken() (int, int) { return 1 }\nfunc appMain() int { return 0 }\n")},
	})
	if result.Ok {
		t.Fatal("invalid return count accepted")
	}
	want := Diagnostic{
		Phase:   "checker",
		Code:    "RENVO-CHECK-007",
		Message: "return value count does not match function results",
		Path:    "/repo/case/cmd/app/main.go",
		Line:    3,
		Column:  28,
	}
	if result.Diagnostic.Phase != want.Phase || result.Diagnostic.Code != want.Code || result.Diagnostic.Message != want.Message || result.Diagnostic.Path != want.Path || result.Diagnostic.Line != want.Line || result.Diagnostic.Column != want.Column {
		t.Fatalf("diagnostic = %#v, want key fields %#v", result.Diagnostic, want)
	}
	formatted := FormatDiagnostic(result.Diagnostic)
	if !strings.Contains(formatted, "/repo/case/cmd/app/main.go:3:28: error RENVO-CHECK-007 (checker):") {
		t.Fatalf("formatted diagnostic = %q", formatted)
	}
}

func TestBuildUnitReportsStructuredParserDiagnostic(t *testing.T) {
	result := BuildUnit([]string{"-o", "app", "./cmd/app"}, "/repo/case", "/std", []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nfunc broken( {\n")},
	})
	if result.Ok || result.Diagnostic.Code != "RENVO-PARSE-001" || result.Diagnostic.Path != "/repo/case/cmd/app/main.go" || result.Diagnostic.Line < 1 || result.Diagnostic.Column < 1 {
		t.Fatalf("parser diagnostic = %#v", result.Diagnostic)
	}
}

func TestEmbeddedBuildCacheValidatesSelectedSourceContent(t *testing.T) {
	embeddedBuildCacheValid = false
	frontendbuild.InitializePackageProgramCache()
	files := driverTestFiles()
	files = append(files, load.SourceFile{Path: "/repo/case/app", Src: []byte("executable")})
	args := []string{"-t", "darwin/arm64", "-s", "-o", "/repo/case/app", "./cmd/app"}
	first := buildFromFSCompact(args, "/repo/case", "/std", memorySourceFS{files: files})
	if !first.Ok || first.CacheHit {
		t.Fatalf("cold compact build = %#v", first)
	}
	rememberEmbeddedBuild(first)
	second := buildFromFSCompact(args, "/repo/case", "/std", memorySourceFS{files: files})
	if !second.Ok || !second.CacheHit {
		t.Fatalf("unchanged compact build = %#v", second)
	}
	for i := 0; i < len(files); i++ {
		if files[i].Path == "/repo/case/cmd/app/main.go" {
			files[i].Src = []byte("package main\nfunc appMain() int { return 1 }\n")
		}
	}
	changed := buildFromFSCompact(args, "/repo/case", "/std", memorySourceFS{files: files})
	if !changed.Ok || changed.CacheHit {
		t.Fatalf("changed compact build = %#v", changed)
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
