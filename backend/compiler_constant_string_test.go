package main

import (
	"os"
	"strings"
	"testing"

	"renvo.dev/std/vm"
)

func TestVM32ConstantStringConcatenation(t *testing.T) {
	source, err := os.ReadFile("tests/constant_string_concatenation.go")
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("tests/constant_string_concatenation.expected")
	if err != nil {
		t.Fatal(err)
	}
	for _, multiline := range []bool{false, true} {
		text := string(source)
		if multiline {
			text = strings.ReplaceAll(text, " + ", " +\n")
		}
		image, ok := RenvoCompileSourceToBytesWithOptions([]byte(text), "vm/vm32", RenvoCompileOptions{ArenaSize: 8192, StripSymbols: true})
		if !ok {
			t.Fatalf("compile (multiline=%v)", multiline)
		}
		result := vm.Run(image, vm.Limits{Steps: 100000, Memory: 32768})
		if result.Trap != vm.TrapNone || result.ExitCode != 0 || string(result.Output) != string(want) || len(result.Stderr) != 0 {
			t.Fatalf("multiline=%v trap=%d exit=%d output=%q stderr=%q", multiline, result.Trap, result.ExitCode, result.Output, result.Stderr)
		}
	}
}
