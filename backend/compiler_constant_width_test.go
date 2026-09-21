package main

import (
	"os"
	"testing"

	"renvo.dev/std/vm"
)

func TestConstantBranchWidthOn32BitTarget(t *testing.T) {
	for _, name := range []string{"constant_branch_literal_width", "unsigned_word_constant_shift"} {
		t.Run(name, func(t *testing.T) {
			source, err := os.ReadFile("tests/" + name + ".go")
			if err != nil {
				t.Fatal(err)
			}
			image, ok := RenvoCompileSourceToBytesWithOptions(source, "vm/vm32", RenvoCompileOptions{ArenaSize: 8192, StripSymbols: true})
			if !ok {
				t.Fatal("compile vm/vm32")
			}
			result := vm.Run(image, vm.Limits{Steps: 100000, Memory: 32768})
			if result.Trap != vm.TrapNone || result.ExitCode != 0 || string(result.Output) != "PASS\n" {
				t.Fatalf("trap=%d exit=%d output=%q", result.Trap, result.ExitCode, result.Output)
			}
		})
	}
}
