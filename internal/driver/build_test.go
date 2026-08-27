package driver

import (
	"bytes"
	"strings"
	"testing"

	"renvo.dev/backend/unit"
	frontendbuild "renvo.dev/internal/build"
	"renvo.dev/internal/load"
	"renvo.dev/internal/pipeline"
	wireunit "renvo.dev/internal/unit"
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

func TestBuildUnitCarriesTargetBoundForeignProgram(t *testing.T) {
	files := []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/main.go", Src: []byte(`package main

//renvo:compile -t linux/386 payloadMain
var payload []byte

func shared() int { return 42 }
func payloadMain() { print(shared()) }
func main() { print(len(payload)) }
`)},
	}
	result := BuildUnit([]string{"-emit-unit", "-t", "linux/amd64", "-o", "app.unit", "."}, "/repo/case", "/std", files)
	if !result.Ok {
		t.Fatalf("foreign build failed: %#v", result.Diagnostic)
	}
	programs, ok := wireunit.ReadForeignPrograms(result.Unit)
	if !ok || len(programs) != 1 {
		t.Fatalf("foreign programs = %#v, ok=%v", programs, ok)
	}
	if programs[0].Name != "payload" || programs[0].Kind != wireunit.ForeignProgramBytes || programs[0].Target != "linux/386" || len(programs[0].Unit) == 0 {
		t.Fatalf("foreign program = %#v", programs[0])
	}
	binding, ok := wireunit.ReadTargetBinding(programs[0].Unit)
	if !ok || binding.Target != "linux/386" {
		t.Fatalf("foreign binding = %#v, ok=%v", binding, ok)
	}
}

func TestBuildUnitRejectsInvalidForeignVariableType(t *testing.T) {
	files := []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/main.go", Src: []byte("package main\n//renvo:compile -t linux/386 payloadMain\nvar payload string\nfunc payloadMain() {}\nfunc main() {}\n")},
	}
	result := BuildUnit([]string{"-t", "linux/amd64", "-o", "app", "."}, "/repo/case", "/std", files)
	if result.Ok || result.Diagnostic.Code != "RENVO-FOREIGN-003" || result.Diagnostic.Line != 2 || result.Diagnostic.Column != 1 {
		t.Fatalf("invalid foreign type = %#v", result)
	}
}

func TestBuildUnitRequiresForeignDirectiveNextToDeclaration(t *testing.T) {
	files := []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/main.go", Src: []byte("package main\n//renvo:compile -t linux/386 payloadMain\n\nvar payload []byte\nfunc payloadMain() {}\nfunc main() {}\n")},
	}
	result := BuildUnit([]string{"-t", "linux/amd64", "-o", "app", "."}, "/repo/case", "/std", files)
	if result.Ok || result.Diagnostic.Code != "RENVO-FOREIGN-002" || result.Diagnostic.Line != 2 {
		t.Fatalf("separated foreign directive = %#v", result)
	}
}

func TestCompactPackageUnitCarriesForeignProgram(t *testing.T) {
	files := []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/main.go", Src: []byte("package main\n//renvo:compile -t linux/386 payloadMain\nvar payload []byte\nfunc payloadMain() {}\nfunc main() { print(len(payload)) }\n")},
	}
	result := BuildPackageUnitCompact(".", "linux/amd64", nil, "/repo/case", "/std", memorySourceFS{files: files})
	if !result.Ok {
		t.Fatalf("compact foreign build = %#v", result)
	}
	programs, ok := wireunit.ReadForeignPrograms(result.Unit)
	if !ok || len(programs) != 1 || programs[0].Target != "linux/386" || len(programs[0].Unit) == 0 {
		t.Fatalf("compact foreign programs = %#v, ok=%v", programs, ok)
	}
}

func TestCompactPackageUnitCarriesFixedTargetBinding(t *testing.T) {
	files := driverTestFiles()
	result := BuildPackageUnitCompact("./cmd/app", "wasi/wasm32", nil, "/repo/case", "/std", memorySourceFS{files: files})
	if !result.Ok {
		t.Fatalf("compact package build = %#v", result)
	}
	binding, ok := wireunit.ReadTargetBinding(result.Unit)
	if !ok || binding.Target != "wasi/wasm32" || len(binding.Definition) != 32 || binding.DescriptorVersion <= 0 {
		t.Fatalf("compact target binding = %#v, ok=%v", binding, ok)
	}
}

func TestCompactKernelModuleCarriesKernelTargetBinding(t *testing.T) {
	files := driverTestFiles()
	result := BuildPackageUnitCompactMode("./cmd/app", "linux/amd64", nil, ModeKernelModule, "/repo/case", "/std", memorySourceFS{files: files})
	if !result.Ok {
		t.Fatalf("compact kernel module build = %#v", result)
	}
	binding, ok := wireunit.ReadTargetBinding(result.Unit)
	if !ok || binding.Target != "linux-kernel/amd64" || len(binding.Definition) != 32 || binding.DescriptorVersion <= 0 {
		t.Fatalf("compact kernel target binding = %#v, ok=%v", binding, ok)
	}
}

func TestCompactPackageUnitPreservesStructuredDiagnostic(t *testing.T) {
	files := []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n\nfunc main() { print(missing) }\n")},
	}
	result := BuildPackageUnitCompact("./cmd/app", "wasi/wasm32", nil, "/repo/case", "/std", memorySourceFS{files: files})
	if result.Ok {
		t.Fatal("compact package build unexpectedly succeeded")
	}
	want := Diagnostic{
		Phase: "checker", Code: "RENVO-CHECK-029", Path: "/repo/case/cmd/app/main.go",
		Line: 3, Column: 21,
	}
	if result.Diagnostic.Phase != want.Phase || result.Diagnostic.Code != want.Code ||
		result.Diagnostic.Path != want.Path || result.Diagnostic.Line != want.Line ||
		result.Diagnostic.Column != want.Column {
		t.Fatalf("compact diagnostic = %#v, want phase=%q code=%q path=%q line=%d column=%d",
			result.Diagnostic, want.Phase, want.Code, want.Path, want.Line, want.Column)
	}
}

func TestOneShotEmitUnitPreservesCanonicalPackageMetadata(t *testing.T) {
	args := []string{"-emit-unit", "-t", "linux/amd64", "-o", "program.unit", "./cmd/app"}
	files := driverTestFiles()
	want := BuildUnit(args, "/repo/case", "/std", files)
	got := buildFromFSOneShotCompactWithModuleCache(args, "/repo/case", "/std", "", memorySourceFS{files: files})
	if !want.Ok || !got.Ok {
		t.Fatalf("persistent ok=%v, one-shot ok=%v", want.Ok, got.Ok)
	}
	if !bytes.Equal(got.Unit, want.Unit) {
		t.Fatal("one-shot -emit-unit output differs from canonical persistent unit")
	}
}

func TestCompactObjectBuildPreservesPersistentSemantics(t *testing.T) {
	command := NormalizeCCompilerCommand([]string{
		"renvo", "cc", "-c", "-nostdinc", "-emit-unit", "-t", "linux/amd64", "-o", "program.unit", "main.c",
	})
	args := command[1:]
	fs := memorySourceFS{files: []load.SourceFile{{
		Path: "/repo/case/main.c",
		Src:  []byte("int value(void) { return 42; }\n"),
	}}}
	want := BuildFromFS(args, "/repo/case", "/std", fs)
	got := buildFromFSCompact(args, "/repo/case", "/std", fs)
	if !want.Ok || !got.Ok {
		t.Fatalf("persistent ok=%v diagnostic=%#v, compact ok=%v diagnostic=%#v",
			want.Ok, want.Diagnostic, got.Ok, got.Diagnostic)
	}
	wantUnit, wantErr := unit.Unmarshal(want.Unit)
	gotUnit, gotErr := unit.Unmarshal(got.Unit)
	if wantErr != nil || gotErr != nil {
		t.Fatalf("decode persistent error=%v, compact error=%v", wantErr, gotErr)
	}
	if !bytes.Equal(gotUnit.Text, wantUnit.Text) || len(gotUnit.Funcs) != len(wantUnit.Funcs) {
		t.Fatalf("compact C object semantics differ: text=%v funcs=%d/%d",
			bytes.Equal(gotUnit.Text, wantUnit.Text), len(gotUnit.Funcs), len(wantUnit.Funcs))
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

func TestBuildUnitKernelModuleLicense(t *testing.T) {
	files := driverTestFiles()
	files[1].Src = append([]byte("// renvo:module-license Dual MIT/GPL\n"), files[1].Src...)
	result := BuildUnit([]string{"-mode=kernel-module", "-o", "app.ko", "./cmd/app"}, "/repo/case", "/std", files)
	if !result.Ok || result.Options.ModuleLicense != "Dual MIT/GPL" {
		t.Fatalf("kernel module license result = %#v", result)
	}

	files[2].Src = append([]byte("// renvo:module-license GPL\n"), files[2].Src...)
	result = BuildUnit([]string{"-mode=kernel-module", "-o", "app.ko", "./cmd/app"}, "/repo/case", "/std", files)
	if result.Ok || result.Error != BuildErrOptions || result.Options.Error != ParseErrConflictingModuleLicense {
		t.Fatalf("conflicting license result = %#v", result)
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
	if result.Ok || result.Diagnostic.Code != "RENVO-PARSE-006" || result.Diagnostic.Path != "/repo/case/cmd/app/main.go" || result.Diagnostic.Line < 1 || result.Diagnostic.Column < 1 {
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
