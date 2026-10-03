package main

import (
	"os"
	"testing"

	"renvo.dev/std/vm"
)

func TestVM32UnsafeLayoutConstants(t *testing.T) {
	source, err := os.ReadFile("tests/unsafe_layout_constants.go")
	if err != nil {
		t.Fatal(err)
	}
	resetRuntime()
	image, ok := RenvoCompileSourceToBytesWithOptions(source, "vm/vm32", RenvoCompileOptions{ArenaSize: 8 * 1024 * 1024, StripSymbols: true})
	if !ok {
		t.Fatal("compile")
	}
	result := vm.RunConfig(image, vm.Config{Limits: vm.Limits{Steps: 10000000, Memory: 16 * 1024 * 1024}})
	if result.Trap != vm.TrapNone || result.ExitCode != 0 || string(result.Output) != "PASS\n" || len(result.Stderr) != 0 {
		t.Fatalf("trap=%d exit=%d output=%q stderr=%q", result.Trap, result.ExitCode, result.Output, result.Stderr)
	}
}
