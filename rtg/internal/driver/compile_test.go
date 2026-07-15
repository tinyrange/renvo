package driver

import (
	"bytes"
	"testing"

	"j5.nz/rtg/rtg/internal/load"
	"j5.nz/rtg/rtg/internal/unit"
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

type recordingBackend struct {
	binary     []byte
	called     bool
	target     string
	strip      bool
	windowsGUI bool
	program    unit.Program
}

func (b *recordingBackend) CompileUnit(data []byte, target string, strip bool, windowsGUI bool) BackendResult {
	b.called = true
	b.target = target
	b.strip = strip
	b.windowsGUI = windowsGUI
	program, ok := unit.Unmarshal(data)
	if !ok {
		return BackendResult{}
	}
	b.program = program
	if len(b.binary) == 0 {
		return BackendResult{Diagnostic: Diagnostic{Phase: "backend", Code: "TEST-BACKEND-001", Message: "intentional backend failure"}}
	}
	return BackendResult{Binary: b.binary, Ok: true}
}
