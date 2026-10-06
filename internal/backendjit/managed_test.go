//go:build !renvo

package backendjit

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/driver"
)

func TestCompilerJITManagedBodiesAreInlineAndCallable(t *testing.T) {
	if hostTarget() == "" {
		t.Skip("no in-process backend for this host")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ definition, target, first string }{
		{"backend/definitions/linux_amd64.rtg", "linux/amd64", "rdi"},
		{"backends/esp32c6.rtg", "esp32c6/riscv32", "a0"},
	} {
		t.Run(test.target, func(t *testing.T) {
			project := t.TempDir()
			definition := filepath.Join(root, test.definition)
			backend := New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"), backendJITTestCacheDir, backendcompiled.Backend{})
			assembly := []byte("rtgasm 3\nassembly { sum(out:emitter) { inputs = 2\nintrinsic = word_sub\nyield() } }\n")
			for name, source := range map[string][]byte{
				"go.mod":     []byte("module example.com/managed\n"),
				"main.go":    []byte("package main\nfunc sum(a,b int) int\nvar count int\nfunc arg(n int) int { count++; return n }\nfunc appMain() int { x:=sum(arg(50),arg(8)); y:=sum(arg(67),arg(25)); f:=sum; z:=f(63,21); if x!=42 || y!=42 || z!=42 || count!=4 {return 1}; return 0 }\n"),
				"sum.rtgasm": assembly,
			} {
				if err := os.WriteFile(filepath.Join(project, name), source, 0600); err != nil {
					t.Fatal(err)
				}
			}
			result := driver.CompileFromFS([]string{"-backend", definition, "-t", test.target, "-s", "-o", "image", "."}, project, filepath.Join(root, "std"), driver.OSFS{}, backend)
			if !result.Ok {
				t.Fatalf("managed compilation: %+v", result.Diagnostic)
			}
			prepared := backend.prepare(test.target)
			evaluated, status := backend.evaluateRTGAssembly(result.Build.Unit, prepared)
			if !status.Ok {
				t.Fatal(status.Diagnostic)
			}
			_, bindings, ok := readRTGAssembly(evaluated)
			if !ok || len(bindings) != 1 || bindings[0].Mode != 1 || bindings[0].Inputs != 2 || bindings[0].Outputs != 1 {
				t.Fatalf("managed binding: %+v", bindings)
			}
			// A wrapper is emitted once for function values; each direct use
			// contains its own complete physical body rather than CALLing it.
			if n := bytes.Count(result.Binary, bindings[0].Code); n < 3 {
				t.Fatalf("body emitted %d times; expected two inline sites plus callable wrapper", n)
			}
			if test.target == "linux/amd64" && runtime.GOOS == "linux" && runtime.GOARCH == "amd64" {
				output := filepath.Join(project, "image")
				if err := os.WriteFile(output, result.Binary, 0700); err != nil {
					t.Fatal(err)
				}
				if out, err := exec.Command(output).CombinedOutput(); err != nil || len(out) != 0 {
					t.Fatalf("runtime operands, result, or wrapper: %s %v", out, err)
				}
			}
			// A new source must be checked even with cached/pre-materialized
			// code, and declaration/transport mismatches must fail closed.
			if err := os.WriteFile(filepath.Join(project, "main.go"), []byte("package main\nfunc sum(a int) int\nfunc appMain() int { return sum(42) }\n"), 0600); err != nil {
				t.Fatal(err)
			}
			mismatch := driver.CompileFromFS([]string{"-backend", definition, "-t", test.target, "-s", "-o", "image", "."}, project, filepath.Join(root, "std"), driver.OSFS{}, backend)
			if mismatch.Ok {
				t.Fatal("accepted a mismatched runtime word signature")
			}
		})
	}
}
