package main

import (
	"os"
	"testing"

	"renvo.dev/std/vm"
)

func TestCompilerFunctionValueTupleInterfaceVM(t *testing.T) {
	source, err := os.ReadFile("tests/function_value_tuple_interface.go")
	if err != nil {
		t.Fatal(err)
	}
	resetRuntime()
	image, ok := RenvoCompileSourceToBytesWithOptions(source, "vm/vm32", RenvoCompileOptions{
		ArenaSize: 8 * 1024 * 1024, StripSymbols: true,
	})
	if !ok {
		t.Fatal("compile VM tuple callback")
	}
	result := vm.RunConfig(image, vm.Config{Limits: vm.Limits{Steps: 500 * 1000 * 1000, Memory: 16 * 1024 * 1024}})
	if result.Trap != vm.TrapNone || result.ExitCode != 0 || string(result.Output)+string(result.Stderr) != "PASS\n" {
		t.Fatalf("exit %d, trap %d, stdout %q, stderr %q", result.ExitCode, result.Trap, result.Output, result.Stderr)
	}
}
