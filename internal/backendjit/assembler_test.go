//go:build !renvo

package backendjit

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/driver"
)

func TestCompilerJITConventionalAssemblerAcrossTargets(t *testing.T) {
	if hostTarget() == "" {
		t.Skip("no in-process backend for this host")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, definition, target, register, kind string
		machine                                  []byte
	}{
		{"8086", "backends/msdos.rtg", "msdos/8086", "ax", "int", []byte{0xe9, 3, 0, 0xb8, 99, 0, 0xb8, 42, 0, 0xc3}},
		{"riscv32", "backends/esp32c6.rtg", "esp32c6/riscv32", "a0", "int64", []byte{0x6f, 0, 0x80, 0, 0x13, 0x05, 0x30, 0x06, 0x13, 0x05, 0xa0, 0x02, 0x67, 0x80, 0, 0}},
		{"amd64-narrow", "backend/definitions/linux_amd64.rtg", "linux/amd64", "rax", "int64", []byte{0xb8, 40, 0, 0, 0, 0xbf, 2, 0, 0, 0, 0x01, 0xf8, 0xc3}},
		{"amd64", "backend/definitions/linux_amd64.rtg", "linux/amd64", "rax", "int64", []byte{0xe9, 10, 0, 0, 0, 0x48, 0xb8, 99, 0, 0, 0, 0, 0, 0, 0, 0x48, 0xb8, 42, 0, 0, 0, 0, 0, 0, 0, 0xc3}},
	} {
		t.Run(test.name, func(t *testing.T) {
			project := t.TempDir()
			definition := filepath.Join(root, test.definition)
			backend := New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"), backendJITTestCacheDir, backendcompiled.Backend{})
			prepared := backend.prepare(test.target)
			if !prepared.Ok {
				t.Fatalf("prepare: %+v", prepared.Diagnostic)
			}
			var body string
			switch test.name {
			case "8086":
				body = "jmp 1f; movw $99,%ax\n1: movw $42,%ax; ret"
			case "amd64-narrow":
				body = "mov eax,40; mov edi,2; add eax,edi; ret"
			case "amd64":
				body = "jmp 1f; movq $99,%rax\n1: movq $42,%rax; ret"
			case "riscv32":
				body = "j 1f; li a0,99\n1: li a0,42; ret"
			}
			prefix := ".text\n"
			if test.name == "amd64-narrow" {
				prefix += ".intel_syntax noprefix\n"
			}
			assembly := []byte(prefix + ".globl answer\nanswer: " + body + "\n")
			for name, source := range map[string][]byte{
				"go.mod":   []byte("module example.com/typedfrontend\n"),
				"main.go":  []byte("package main\nfunc answer() int\nfunc appMain() int { return answer()-42 }\n"),
				"answer.s": assembly,
			} {
				if err := os.WriteFile(filepath.Join(project, name), source, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			compile := func(backendPath string, selected *Backend) driver.CompileResult {
				return driver.CompileFromFS([]string{"-backend", backendPath, "-t", test.target, "-s", "-o", "image", "."}, project, filepath.Join(root, "std"), driver.OSFS{}, selected)
			}
			result := compile(definition, backend)
			if !result.Ok || !bytes.Contains(result.Binary, test.machine) {
				t.Fatalf("assembler %s compile: %+v; expected body %x", test.target, result.Diagnostic, test.machine)
			}
			// Preserve both the selected vocabulary and the assembly through an
			// RTGB round trip, with no source definition loader at evaluation time.
			artifact := filepath.Join(project, "selected.rtgb")
			if err := os.WriteFile(artifact, prepared.Encoded, 0o600); err != nil {
				t.Fatal(err)
			}
			persisted := New(artifact, filepath.Join(root, "backend"), filepath.Join(root, "std"), backendJITTestCacheDir, backendcompiled.Backend{})
			again := compile(artifact, persisted)
			if !again.Ok || !bytes.Equal(result.Binary, again.Binary) {
				t.Fatalf("RTGB changed typed output: %+v", again.Diagnostic)
			}
			// Source validation is mandatory on repeat compilation too.
			invalid := bytes.Replace(assembly, []byte("ret"), []byte("undeclared_helper"), 1)
			if err := os.WriteFile(filepath.Join(project, "answer.s"), invalid, 0o600); err != nil {
				t.Fatal(err)
			}
			failed := compile(definition, backend)
			if failed.Ok || failed.Diagnostic.Code != "RENVO-RTGASM-003" || failed.Diagnostic.Path != "answer.s" || failed.Diagnostic.Start == 0 {
				t.Fatalf("typed diagnostic: %+v", failed.Diagnostic)
			}
		})
	}
}
