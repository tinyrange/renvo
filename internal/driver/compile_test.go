package driver

import (
	"bytes"
	"testing"

	"renvo.dev/backend/unit"
	"renvo.dev/internal/load"
	wireunit "renvo.dev/internal/unit"
)

func TestCompileUnitInvokesBackend(t *testing.T) {
	backend := &recordingBackend{binary: []byte("binary")}
	result := CompileUnit([]string{"-t", "windows/386", "-s", "-windows-gui", "-o", "app", "./cmd/app"}, "/repo/case", "/std", driverTestFiles(), backend)
	if !result.Ok {
		t.Fatalf("CompileUnit failed: err=%d build=%#v", result.Error, result.Build)
	}
	if !bytes.Equal(result.Binary, []byte("binary")) {
		t.Fatalf("binary = %q", string(result.Binary))
	}
	if backend.target != "windows/386" || !backend.strip || !backend.windowsGUI {
		t.Fatalf("backend target/strip/windowsGUI = %q/%v/%v", backend.target, backend.strip, backend.windowsGUI)
	}
	if backend.program.Package != "main" || len(backend.program.Funcs) != 2 {
		t.Fatalf("backend program = package %q funcs %d", backend.program.Package, len(backend.program.Funcs))
	}
}

func TestCompileFromFSInvokesBackend(t *testing.T) {
	backend := &recordingBackend{binary: []byte("fs-binary")}
	result := CompileFromFS([]string{"-o", "app", "./cmd/app"}, "/repo/case", "/std", memorySourceFS{files: driverTestFiles()}, backend)
	if !result.Ok {
		t.Fatalf("CompileFromFS failed: err=%d build=%#v", result.Error, result.Build)
	}
	if !bytes.Equal(result.Binary, []byte("fs-binary")) {
		t.Fatalf("binary = %q", string(result.Binary))
	}
	if backend.target != DefaultTarget || backend.strip {
		t.Fatalf("backend target/strip = %q/%v", backend.target, backend.strip)
	}
}

func TestCompileResolvesForeignUnitBeforeOuterBackend(t *testing.T) {
	backend := &recordingBackend{binary: []byte("artifact")}
	files := []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/main.go", Src: []byte("package main\n//renvo:compile -t linux/386 payloadMain\nvar payload []byte\nfunc payloadMain() { print(42) }\nfunc main() { print(len(payload)) }\n")},
	}
	result := CompileUnit([]string{"-t", "linux/amd64", "-o", "app", "."}, "/repo/case", "/std", files, backend)
	if !result.Ok {
		t.Fatalf("foreign compile = %#v", result)
	}
	if len(backend.targets) != 2 || backend.targets[0] != "linux/386" || backend.targets[1] != "linux/amd64" {
		t.Fatalf("backend target order = %#v", backend.targets)
	}
	programs, ok := wireunit.ReadForeignPrograms(result.Build.Unit)
	if !ok || len(programs) != 1 || len(programs[0].Unit) != 0 || string(programs[0].Artifact) != "artifact" {
		t.Fatalf("resolved foreign programs = %#v, ok=%v", programs, ok)
	}
}

func TestCompileBuildResultSkipsBackendForCurrentOutput(t *testing.T) {
	backend := &recordingBackend{binary: []byte("unexpected")}
	result := CompileBuildResult(BuildResult{Ok: true, CacheHit: true}, backend)
	if !result.Ok || !result.Build.CacheHit {
		t.Fatalf("cached compile result = %#v", result)
	}
	if backend.called || len(result.Binary) != 0 {
		t.Fatal("backend ran for an unchanged build output")
	}
}

func TestCompileBrowserTargetUsesWASIBackendAndPackagesHTML(t *testing.T) {
	backend := &recordingBackend{binary: []byte{0, 'a', 's', 'm'}}
	result := CompileUnit([]string{"-t", "browser/wasm32", "-o", "app.html", "./cmd/app"}, "/repo/case", "/std", driverTestFiles(), backend)
	if !result.Ok {
		t.Fatalf("browser compile failed: %#v", result)
	}
	if backend.target != "wasi/wasm32" {
		t.Fatalf("backend target = %q", backend.target)
	}
	if !bytes.HasPrefix(result.Binary, []byte("<!doctype html>")) || !bytes.Contains(result.Binary, []byte("AGFzbQ==")) {
		t.Fatalf("browser output was not a self-contained HTML file")
	}
}

func TestCompileKernelModePassesStructuredBackendOptions(t *testing.T) {
	backend := &recordingOptionsBackend{binary: []byte("module")}
	files := driverTestFiles()
	files[1].Src = append([]byte("// renvo:module-license Dual MIT/GPL\n"), files[1].Src...)
	result := CompileUnit([]string{"-mode=kernel-module", "-arena-size", "65536", "-o", "hello-world.ko", "./cmd/app"}, "/repo/case", "/std", files, backend)
	if !result.Ok || !bytes.Equal(result.Binary, backend.binary) {
		t.Fatalf("kernel compile = %#v", result)
	}
	if backend.options.Target != "linux/amd64" || backend.options.Mode != ModeKernelModule || backend.options.Output != "hello-world.ko" || backend.options.ArenaSize != 65536 || backend.options.ModuleLicense != "Dual MIT/GPL" {
		t.Fatalf("backend options = %#v", backend.options)
	}
}

func TestCompileCObjectPassesStructuredBackendOptions(t *testing.T) {
	backend := &recordingOptionsBackend{binary: []byte("object")}
	files := []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/main.c", Src: []byte("int main(void) { return 0; }\n")},
	}
	result := CompileUnit([]string{"-c", "-o", "hello.o", "main.c"}, "/repo/case", "/std", files, backend)
	if !result.Ok || !bytes.Equal(result.Binary, backend.binary) {
		t.Fatalf("object compile = %#v", result)
	}
	if backend.options.Target != "linux/amd64" || backend.options.Mode != ModeObject || backend.options.Output != "hello.o" || !backend.options.ObjectFile || backend.options.ArenaSize != objectDefaultArenaSize {
		t.Fatalf("backend options = %#v", backend.options)
	}
}

func TestCompileM16CObjectPassesCode16BackendOption(t *testing.T) {
	backend := &recordingOptionsBackend{binary: []byte("object")}
	files := []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/boot.c", Src: []byte("int main(void) { return 0; }\n")},
	}
	args := NormalizeCCompilerCommand([]string{"renvo", "cc", "-m16", "-mregparm=3", "-c", "boot.c", "-o", "boot.o"})
	result := CompileUnit(args[1:], "/repo/case", "/std", files, backend)
	if !result.Ok || backend.options.Target != "linux/386" || !backend.options.ObjectFile || !backend.options.Code16 || backend.options.RegParm != 3 {
		t.Fatalf("-m16 object compile = %#v, backend options = %#v", result, backend.options)
	}
}

func TestCompileReportsBuildFailure(t *testing.T) {
	backend := &recordingBackend{binary: []byte("binary")}
	result := CompileUnit([]string{"-t", "invalid", "-o", "app", "./cmd/app"}, "/repo/case", "/std", driverTestFiles(), backend)
	if result.Ok || result.Error != CompileErrBuild {
		t.Fatalf("bad option compile result = %#v", result)
	}
	if backend.called {
		t.Fatal("backend was called after build failure")
	}

	result = CompileFromFS([]string{"-o", "app", "./cmd/app"}, "/repo/case", "/std", memorySourceFS{files: []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
	}}, backend)
	if result.Ok || result.Error != CompileErrBuild {
		t.Fatalf("source failure compile result = %#v", result)
	}
}

func TestCompileReportsBackendFailure(t *testing.T) {
	failing := &recordingBackend{}
	result := CompileUnit([]string{"-o", "app", "./cmd/app"}, "/repo/case", "/std", driverTestFiles(), failing)
	if result.Ok || result.Error != CompileErrBackend {
		t.Fatalf("backend failure result = %#v", result)
	}
	if !failing.called {
		t.Fatal("backend was not called")
	}

	result = CompileUnit([]string{"-o", "app", "./cmd/app"}, "/repo/case", "/std", driverTestFiles(), nil)
	if result.Ok || result.Error != CompileErrBackend {
		t.Fatalf("nil backend result = %#v", result)
	}
}

func TestCompileRejectsRTGAssemblyWithoutCompilerJIT(t *testing.T) {
	backend := &recordingBackend{binary: []byte("unexpected")}
	files := []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/main.go", Src: []byte("package main\nfunc answer() int\nfunc appMain() int { return answer() }\n")},
		{Path: "/repo/case/answer.rtgasm", Src: []byte("rtgasm 1\nassembly { answer(out:emitter) { out.Byte(0xc3) } }\n")},
	}
	result := CompileUnit([]string{"-o", "app", "."}, "/repo/case", "/std", files, backend)
	if result.Ok || result.Diagnostic.Code != "RENVO-RTGASM-010" || backend.called {
		t.Fatalf("built-in RTGASM compile = %#v, backend called=%v", result, backend.called)
	}
}

type recordingBackend struct {
	binary     []byte
	called     bool
	target     string
	strip      bool
	windowsGUI bool
	program    unit.Program
	targets    []string
}

type recordingOptionsBackend struct {
	binary  []byte
	options BackendCompileOptions
}

func (b *recordingOptionsBackend) CompileUnit([]byte, string, bool, bool) BackendResult {
	return BackendResult{}
}

func (b *recordingOptionsBackend) CompileUnitWithOptions(data []byte, options BackendCompileOptions) BackendResult {
	b.options = options
	if _, err := unit.Unmarshal(data); err != nil {
		return BackendResult{}
	}
	return BackendResult{Binary: b.binary, Ok: true}
}

func (b *recordingBackend) CompileUnit(data []byte, target string, strip bool, windowsGUI bool) BackendResult {
	b.called = true
	b.target = target
	b.targets = append(b.targets, target)
	b.strip = strip
	b.windowsGUI = windowsGUI
	program, err := unit.Unmarshal(data)
	if err != nil {
		return BackendResult{}
	}
	b.program = program
	if len(b.binary) == 0 {
		return BackendResult{Diagnostic: Diagnostic{Phase: "backend", Code: "TEST-BACKEND-001", Message: "intentional backend failure"}}
	}
	return BackendResult{Binary: b.binary, Ok: true}
}
