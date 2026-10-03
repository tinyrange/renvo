package main

import (
	"os"
	"testing"

	"renvo.dev/std/vm"
)

func TestVM32ZeroSizeAndArrayAppendValues(t *testing.T) {
	for _, name := range []string{"zero_size_values", "array_append_values", "make_slice_bounds_recover", "vm_immediate_pop_opcode", "vm_signed_compare_extremes", "vm_compare_literal_opcodes"} {
		t.Run(name, func(t *testing.T) {
			source, err := os.ReadFile("tests/" + name + ".go")
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
		})
	}
}
