package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWASIFixedTargetValueMatchesBranchSpecialization(t *testing.T) {
	target := compilerTarget{name: "wasi/wasm32", runner: []string{"wasmtime", "run"}}
	skipIfTargetRunnerMissing(t, target)
	source, err := os.ReadFile("tests/compiler_fixed_target_value.go")
	if err != nil {
		t.Fatal(err)
	}
	image, ok := RenvoCompileSourceToBytes(source, target.name)
	if !ok {
		t.Fatal("compile fixed-target regression")
	}
	path := filepath.Join(t.TempDir(), "fixed-target.wasm")
	if err := os.WriteFile(path, image, 0600); err != nil {
		t.Fatal(err)
	}
	result, err := runTargetCommand(t, target, path)
	if err != nil {
		t.Fatal(err)
	}
	compareCommandResult(t, expectedCommandResult(t, "tests/compiler_fixed_target_value.go"), result)
}
