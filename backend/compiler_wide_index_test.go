package main

import (
	"os"
	"path/filepath"
	"testing"

	"renvo.dev/internal/driver"
	"renvo.dev/internal/load"
	"renvo.dev/std/vm"
)

// Target-resident corpus compilers and the Go-hosted compiler must agree on
// complete wide values. Exercise the real frontend's compact-unit path too.
func TestStage0WideIndexCompactUnit(t *testing.T) {
	for _, target := range []string{"vm/vm32", "wasi/wasm32"} {
		t.Run(target, func(t *testing.T) {
			runner := compilerTarget{name: target, runner: []string{"wasmtime", "run"}}
			if target == "wasi/wasm32" {
				skipIfTargetRunnerMissing(t, runner)
			}
			for _, path := range []string{"tests/wide_call_array_index.go", "tests/wide_index_high_pc_publication.go", "tests/named_constant_wide_arithmetic.go"} {
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
