package main

import (
	"os"
	"testing"

	"renvo.dev/std/vm"
)

func TestGlobalDeclarationSemicolonVM(t *testing.T) {
	for _, name := range []string{"global_declaration_semicolon", "global_declaration_same_line"} {
		t.Run(name, func(t *testing.T) {
			testGlobalDeclarationVM(t, name)
		})
	}
}

func testGlobalDeclarationVM(t *testing.T, name string) {
	t.Helper()
	source, err := os.ReadFile("tests/" + name + ".go")
	if err != nil {
		t.Fatal(err)
	}
	resetRuntime()
	image, ok := RenvoCompileSourceToBytesWithOptions(source, "vm/vm32", RenvoCompileOptions{
		ArenaSize: 8 * 1024 * 1024, StripSymbols: true,
	})
	if !ok {
		t.Fatal("compile vm/vm32")
	}
	result := vm.RunConfig(image, vm.Config{Limits: vm.Limits{
		Steps: 500 * 1000 * 1000, Memory: 16 * 1024 * 1024,
	}})
	if result.Trap != vm.TrapNone || result.ExitCode != 0 || string(result.Output)+string(result.Stderr) != "PASS\n" {
		t.Fatalf("exit %d, trap %d, stdout %q, stderr %q", result.ExitCode, result.Trap, result.Output, result.Stderr)
	}
}
