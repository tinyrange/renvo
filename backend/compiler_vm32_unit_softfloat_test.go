package main

import (
	"os"
	"testing"

	"renvo.dev/backend/unit"
	"renvo.dev/internal/targetinfo"
	internalunit "renvo.dev/internal/unit"
	"renvo.dev/std/vm"
)

func TestVM32UnitComplexConstantComponents(t *testing.T) {
	resetRuntime()
	path := "tests/complex_constant_components.go"
	program, err := unit.ConvertFiles([]string{path})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := unit.Marshal(program)
	if err != nil {
		t.Fatal(err)
	}
	target, definition, version, found := targetinfo.Binding("vm/vm32")
	if !found {
		t.Fatal("VM32 target binding unavailable")
	}
	encoded, bound := internalunit.BindTarget(encoded, internalunit.TargetBinding{Target: target, Definition: definition, DescriptorVersion: version})
	if !bound {
		t.Fatal("bind compact unit")
	}
	image, ok := RenvoCompileUnitToBytesWithOptions(encoded, "vm/vm32", RenvoCompileOptions{ArenaSize: 262144, StripSymbols: true})
	if !ok {
		t.Fatal("compile compact unit")
	}
	want, err := os.ReadFile("tests/complex_constant_components.expected")
	if err != nil {
		t.Fatal(err)
	}
	result := vm.Run(image, vm.Limits{Steps: 10000000, Memory: 2 * 1024 * 1024})
	if result.Trap != vm.TrapNone || result.ExitCode != 0 || string(result.Output) != string(want) || len(result.Stderr) != 0 {
		t.Fatalf("trap=%d exit=%d output=%q stderr=%q", result.Trap, result.ExitCode, result.Output, result.Stderr)
	}
}
