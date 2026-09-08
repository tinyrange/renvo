package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"

	"renvo.dev/std/vm"
)

func TestWASIHelperFailureAfterReturn(t *testing.T) {
	source, err := os.ReadFile("tests/arena_helper_failure_tail.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"wasi/wasm32", "vm/vm32"} {
		for _, arena := range []int{4096, 8192} {
			t.Run(target+"/"+strconv.Itoa(arena), func(t *testing.T) {
				image, ok := RenvoCompileSourceToBytesWithOptions(source, target, RenvoCompileOptions{ArenaSize: arena, StripSymbols: true})
				if !ok {
					t.Fatal("compile allocation helper regression")
				}
				var stdout, stderr []byte
				exit := 0
				if target == "vm/vm32" {
					result := vm.Run(image, vm.Limits{Memory: 1024 * 1024, Steps: 1000000})
					if result.Trap != vm.TrapNone {
						t.Fatalf("VM trap %d", result.Trap)
					}
					stdout, stderr, exit = result.Output, result.Stderr, result.ExitCode
				} else {
					runner, err := exec.LookPath("wasmtime")
					if err != nil {
						t.Skip("Wasmtime is unavailable")
					}
					path := filepath.Join(t.TempDir(), "program.wasm")
					if err := os.WriteFile(path, image, 0600); err != nil {
						t.Fatal(err)
					}
					cmd := exec.Command(runner, "run", path)
					var out, diagnostic bytes.Buffer
					cmd.Stdout, cmd.Stderr = &out, &diagnostic
					if err := cmd.Run(); err != nil && cmd.ProcessState == nil {
						t.Fatal(err)
					}
					stdout, stderr, exit = out.Bytes(), diagnostic.Bytes(), cmd.ProcessState.ExitCode()
				}
				wantOut, wantErr, wantExit := "PASS\n", "", 0
				if arena == 4096 {
					wantOut, wantErr, wantExit = "", "out of memory\n", 2
				}
				if string(stdout) != wantOut || string(stderr) != wantErr || exit != wantExit {
					t.Fatalf("stdout=%q stderr=%q exit=%d; want %q %q %d", stdout, stderr, exit, wantOut, wantErr, wantExit)
				}
			})
		}
	}
}
