package main

import (
	"os"
	"testing"

	"renvo.dev/std/vm"
)

func TestVM32CompiledShiftWidth(t *testing.T) {
	source, err := os.ReadFile("tests/vm32_shift_width.go")
	if err != nil {
		t.Fatal(err)
	}
	image, ok := RenvoCompileSourceToBytesWithOptions(source, "vm/vm32", RenvoCompileOptions{ArenaSize: 8192, StripSymbols: true})
	if !ok {
		t.Fatal("compile shift regression")
	}
	result := vm.Run(image, vm.Limits{Steps: 100000, Memory: 32768})
	if result.Trap != vm.TrapNone || result.ExitCode != 0 || string(result.Output) != "PASS\n" {
		t.Fatalf("shift regression: trap=%v exit=%d output=%q", result.Trap, result.ExitCode, result.Output)
	}
}
