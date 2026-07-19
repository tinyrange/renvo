package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"renvo.dev/backend/target"
)

func TestRunReportsValidBoardArtifact(t *testing.T) {
	artifact := validArtifact()
	loader := func(path string, options target.ELFArtifactOptions) (target.Artifact, error) {
		if path != "firmware.elf" {
			t.Fatalf("path = %q", path)
		}
		if options.VectorSymbol != "renvo_vectors" || options.HeapSize != 128 || options.StackSize != 256 {
			t.Fatalf("options = %+v", options)
		}
		artifact.HeapSize = options.HeapSize
		artifact.StackSize = options.StackSize
		return artifact, nil
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"-heap", "128", "-stack", "256", "firmware.elf"}, &stdout, &stderr, loader)
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, stderr.String())
	}
	want := []string{
		"board=wch-ch32v003 object=rv32ec-ilp32e isa=rv32ec abi=ilp32e format=elf32-littleriscv",
		"section=.vectors region=flash address=0x8000000 size=64",
		"section=.text region=flash address=0x8000040 size=1024",
		"section=.data region=ram address=0x20000000 size=128",
		"flash used=1216 free=15168 capacity=16384",
		"ram static=128 heap=128 stack=256 guard=64 free=1472 capacity=2048",
	}
	for _, text := range want {
		if !strings.Contains(stdout.String(), text) {
			t.Errorf("report missing %q:\n%s", text, stdout.String())
		}
	}
}

func TestRunReportsEveryViolation(t *testing.T) {
	artifact := validArtifact()
	artifact.Imports = []string{"printf"}
	artifact.Entry = 0x20000000
	loader := func(string, target.ELFArtifactOptions) (target.Artifact, error) {
		return artifact, nil
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := run([]string{"firmware.elf"}, &stdout, &stderr, loader); code != 1 {
		t.Fatalf("exit = %d", code)
	}
	for _, text := range []string{"bad-entry", "forbidden-import printf"} {
		if !strings.Contains(stderr.String(), text) {
			t.Errorf("diagnostic missing %q:\n%s", text, stderr.String())
		}
	}
	if !strings.Contains(stdout.String(), "flash used=") || !strings.Contains(stdout.String(), "ram static=") {
		t.Fatalf("resource report missing on failure:\n%s", stdout.String())
	}
}

func TestRunRejectsUsageBoardAndInspectionErrors(t *testing.T) {
	loader := func(string, target.ELFArtifactOptions) (target.Artifact, error) {
		return target.Artifact{}, errors.New("broken ELF")
	}
	tests := []struct {
		name string
		args []string
		code int
		want string
	}{
		{name: "help", args: []string{"-h"}, code: 0, want: "usage: renvoboard"},
		{name: "missing", args: nil, code: 2, want: "usage: renvoboard"},
		{name: "board", args: []string{"-board", "unknown", "image.elf"}, code: 2, want: "unsupported board"},
		{name: "vector", args: []string{"-vector", "vectors", "image.elf"}, code: 2, want: "does not match board contract"},
		{name: "inspect", args: []string{"image.elf"}, code: 1, want: "broken ELF"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			if code := run(test.args, &stdout, &stderr, loader); code != test.code {
				t.Fatalf("exit = %d, want %d", code, test.code)
			}
			if !strings.Contains(stderr.String(), test.want) {
				t.Fatalf("stderr missing %q:\n%s", test.want, stderr.String())
			}
		})
	}
}

func validArtifact() target.Artifact {
	composition := target.CH32V003()
	return target.Artifact{
		Format:        composition.Object.ExpectedArtifactFormat(),
		Entry:         0x08000040,
		VectorSymbol:  composition.Board.Startup.VectorSymbol,
		VectorAddress: 0x08000000,
		Sections: []target.Section{
			{Name: ".data", Address: 0x20000000, Size: 128, LoadAddress: 0x08000440, LoadSize: 128, Flags: target.SectionAlloc | target.SectionWrite},
			{Name: ".text", Address: 0x08000040, Size: 1024, LoadAddress: 0x08000040, LoadSize: 1024, Flags: target.SectionAlloc | target.SectionExec},
			{Name: ".vectors", Address: 0x08000000, Size: 64, LoadAddress: 0x08000000, LoadSize: 64, Flags: target.SectionAlloc | target.SectionExec},
		},
	}
}
