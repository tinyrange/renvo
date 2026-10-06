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

func TestCompilerJITTypedMemoryBlocks(t *testing.T) {
	if hostTarget() == "" {
		t.Skip("no in-process backend for this host")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		definition, target, base, value, temporary string
		machine                                    []byte
	}{
		{"backend/definitions/linux_amd64.rtg", "linux/amd64", "rax", "rdx", "rcx", []byte{0x48, 0x8b, 0x10, 0x48, 0xb9, 2, 0, 0, 0, 0, 0, 0, 0, 0x48, 0x01, 0xca, 0x48, 0x89, 0x10, 0x48, 0x89, 0xd0, 0xc3}},
		{"backends/esp32c6.rtg", "esp32c6/riscv32", "a0", "a1", "a2", []byte{0x83, 0x25, 0x05, 0, 0x13, 0x06, 0x20, 0, 0xb3, 0x85, 0xc5, 0, 0x23, 0x20, 0xb5, 0, 0x13, 0x85, 0x05, 0, 0x67, 0x80, 0, 0}},
	} {
		t.Run(test.target, func(t *testing.T) {
			project := t.TempDir()
			body := "let mem = address(register(" + test.base + "), int(0))\n" +
				"load(register(" + test.value + "), mem)\n" +
				"move_immediate(register(" + test.temporary + "), int64(2))\n" +
				"add(register(" + test.value + "), register(" + test.temporary + "))\n" +
				"store(mem, register(" + test.value + "))\n" +
				"move(register(" + test.base + "), register(" + test.value + "))\nreturn()\n"
			assembly := []byte("rtgasm 2\nassembly { update(out:emitter) {\n" + body + "} }\n")
			for name, source := range map[string][]byte{
				"go.mod":        []byte("module example.com/memory\n"),
				"main.go":       []byte("package main\nfunc update(value *int) int\nfunc appMain() int { value := 40; got := update(&value); if got != 42 || value != 42 { return 1 }; return 0 }\n"),
				"update.rtgasm": assembly,
			} {
				if err := os.WriteFile(filepath.Join(project, name), source, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			definition := filepath.Join(root, test.definition)
			backend := New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"), backendJITTestCacheDir, backendcompiled.Backend{})
			result := driver.CompileFromFS([]string{"-backend", definition, "-t", test.target, "-s", "-o", "image", "."}, project, filepath.Join(root, "std"), driver.OSFS{}, backend)
			if !result.Ok || !bytes.Contains(result.Binary, test.machine) {
				t.Fatalf("memory compile: %+v; expected %x", result.Diagnostic, test.machine)
			}
			if test.target == "linux/amd64" && runtime.GOOS == "linux" && runtime.GOARCH == "amd64" {
				image := filepath.Join(project, "image")
				if err := os.WriteFile(image, result.Binary, 0o700); err != nil {
					t.Fatal(err)
				}
				if out, err := exec.Command(image).CombinedOutput(); err != nil || len(out) != 0 {
					t.Fatalf("memory semantics: %s %v", out, err)
				}
			}
		})
	}
}
