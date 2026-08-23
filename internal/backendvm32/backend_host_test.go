//go:build !renvo

package backendvm32

import (
	"bytes"
	"testing"

	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/driver"
	"renvo.dev/internal/load"
	"renvo.dev/std/vm"
)

func TestSeedCompilesFrontendVM32Unit(t *testing.T) {
	files := []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nfunc main() { print(\"PASS\\n\") }\n")},
	}
	compiled := driver.CompileUnit([]string{
		"-t", Target, "-arena-size", "8192", "-s", "-o", "app.rnvb", "./cmd/app",
	}, "/repo/case", "/std", files, Backend{})
	if !compiled.Ok {
		t.Fatalf("VM32 seed compile failed: %#v", compiled.Diagnostic)
	}
	if len(compiled.Binary) < 4 || string(compiled.Binary[:4]) != "RNVB" {
		t.Fatalf("VM32 seed output prefix = % x", compiled.Binary[:min(4, len(compiled.Binary))])
	}
	legacy := (backendcompiled.Backend{}).CompileUnitWithArena(
		compiled.Build.Unit, Target, true, false, 8192)
	if !legacy.Ok || !bytes.Equal(compiled.Binary, legacy.Binary) {
		t.Fatalf("VM32 seed output differs from the existing backend: legacy=%#v, seed=%d bytes",
			legacy.Diagnostic, len(compiled.Binary))
	}
	imageOptions := driver.BackendCompileOptions{
		Target: Target, Mode: driver.ModeExecutable, Strip: true, EmitImage: true, ArenaSize: 8192,
	}
	seedImage := (Backend{}).CompileUnitWithOptions(compiled.Build.Unit, imageOptions)
	legacyImage := (backendcompiled.Backend{}).CompileUnitWithOptions(compiled.Build.Unit, imageOptions)
	if !seedImage.Ok || !legacyImage.Ok || !bytes.Equal(seedImage.Binary, legacyImage.Binary) ||
		len(seedImage.Binary) < 4 || string(seedImage.Binary[:4]) != "RNVI" {
		t.Fatalf("VM32 seed linked image differs from the existing backend: seed=%#v legacy=%#v",
			seedImage.Diagnostic, legacyImage.Diagnostic)
	}
	result := vm.Run(compiled.Binary, vm.Limits{Steps: 100000, Memory: 32768})
	if result.Trap != vm.TrapNone || result.ExitCode != 0 || string(result.Output) != "PASS\n" {
		t.Fatalf("VM32 seed output: exit %d, trap %d, stdout %q, stderr %q",
			result.ExitCode, result.Trap, result.Output, result.Stderr)
	}
}

func TestSeedRejectsNativeTarget(t *testing.T) {
	result := (Backend{}).CompileUnit(nil, "linux/amd64", false, false)
	if result.Ok || result.Diagnostic.Code != "RENVO-BACKEND-007" {
		t.Fatalf("native target result = %#v", result)
	}
}

func TestSeedMatchesCompilerSources(t *testing.T) {
	if CompilerSourceDigest != backendcompiled.CompilerSourceDigest {
		t.Fatalf("VM32 seed source digest = %s, compiler source digest = %s; run go generate ./internal/backendvm32",
			CompilerSourceDigest, backendcompiled.CompilerSourceDigest)
	}
}
