package main

import (
	"os"
	"path/filepath"
	"testing"

	"renvo.dev/internal/driver"
	"renvo.dev/internal/load"
	"renvo.dev/std/vm"
)

// Target-resident and Go-hosted compilers must agree on source semantics:
// strings, ordered switch effects, IEEE payload equality, and recover ownership. Exercise actual
// Wasm and VM32 through the frontend's compact-unit path.
func TestStage0ShortStringCasesCompactUnit(t *testing.T) {
	for _, target := range []string{"vm/vm32", "wasi/wasm32"} {
		t.Run(target, func(t *testing.T) {
			runner := compilerTarget{name: target, runner: []string{"wasmtime", "run"}}
			if target == "wasi/wasm32" {
				skipIfTargetRunnerMissing(t, runner)
			}
			for _, path := range []string{"tests/wasm_short_string_cases.go", "tests/wasm_string_key_tree.go", "tests/wasm_integer_case_tree.go", "tests/wasm_dense_integer_dispatch.go", "tests/wasm_interface_equality_shared.go", "tests/interface_float_ieee_payload.go", "tests/string_equal_word_boundaries.go", "tests/regression_wasm_string_compare_reuse.go", "tests/regression_long_nested_struct_string_compare.go", "tests/panic_pending_entry.go", "tests/wasm_array_bounds_proof.go", "tests/wasm_native_wide_params.go", "tests/wasm_trivial_getters.go", "tests/wasm_aggregate_scalar_replacement.go", "tests/function_value_erased_environment.go", "tests/bound_value_receiver_snapshot.go", "tests/named_callback_method_initializer.go"} {
				t.Run(path, func(t *testing.T) {
					resetRuntime()
					source, err := os.ReadFile(path)
					if err != nil {
						t.Fatal(err)
					}
					built := driver.BuildUnit([]string{"-t", target, "-o", "program", "./main.go"}, "/repo/case", "/std", []load.SourceFile{
						{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
						{Path: "/repo/case/main.go", Src: source},
					})
					if !built.Ok {
						t.Fatalf("frontend build: %#v", built.Diagnostic)
					}
					image, ok := RenvoCompileUnitToBytesWithOptions(built.Unit, target, RenvoCompileOptions{ArenaSize: 8 * 1024 * 1024, StripSymbols: true})
					if !ok {
						t.Fatal("compile compact unit")
					}
					want := expectedCommandResult(t, path)
					if target == "wasi/wasm32" {
						output := filepath.Join(t.TempDir(), "program.wasm")
						if err := os.WriteFile(output, image, 0644); err != nil {
							t.Fatal(err)
						}
						actual, err := runTargetCommand(t, runner, output)
						if err != nil {
							t.Fatal(err)
						}
						compareCommandResult(t, want, actual)
					} else {
						result := vm.RunConfig(image, vm.Config{Args: []string{"program"}, Limits: vm.Limits{Steps: 10000000, Memory: 16 * 1024 * 1024}})
						if result.Trap != vm.TrapNone {
							t.Fatalf("trap=%d stderr=%q", result.Trap, result.Stderr)
						}
						compareCommandResult(t, want, commandResult{stdout: string(result.Output), stderr: string(result.Stderr), exitCode: result.ExitCode})
					}
				})
			}
		})
	}
}
