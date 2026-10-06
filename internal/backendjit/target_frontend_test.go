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
	"renvo.dev/internal/rtg"
)

func TestCompilerJITTypedFrontendAcrossTargets(t *testing.T) {
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
			v := rtg.FrontendOperations(prepared.Resolved, test.target)
			blocks := []rtg.TargetBlock{{Name: "answer", Instructions: []rtg.TargetInstruction{
				{Operation: "new_label", Result: "done"},
				{Operation: "jump", Operands: []rtg.TargetOperand{{Kind: "value", Value: "done"}}},
				{Operation: "move_immediate", Operands: []rtg.TargetOperand{{Kind: "register", Value: test.register}, {Kind: test.kind, Value: "99"}}},
				{Operation: "bind_label", Operands: []rtg.TargetOperand{{Kind: "value", Value: "done"}}},
				{Operation: "move_immediate", Operands: []rtg.TargetOperand{{Kind: "register", Value: test.register}, {Kind: test.kind, Value: "42"}}},
				{Operation: "return"},
			}}}
			assembly, diagnostics := rtg.EncodeTargetAssembly(v, blocks, "answer.rtgasm")
			if len(diagnostics) != 0 {
				t.Fatalf("frontend: %+v", diagnostics)
			}
			for name, source := range map[string][]byte{
				"go.mod":        []byte("module example.com/typedfrontend\n"),
				"main.go":       []byte("package main\nfunc answer() int\nfunc appMain() int { return answer()-42 }\n"),
				"answer.rtgasm": assembly,
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
				t.Fatalf("typed %s compile: %+v; expected body %x", test.target, result.Diagnostic, test.machine)
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
			invalid := bytes.Replace(assembly, []byte("return()"), []byte("emitReturn(out)"), 1)
			if err := os.WriteFile(filepath.Join(project, "answer.rtgasm"), invalid, 0o600); err != nil {
				t.Fatal(err)
			}
			failed := compile(definition, backend)
			if failed.Ok || failed.Diagnostic.Code != "RENVO-RTGASM-003" || failed.Diagnostic.Path != "answer.rtgasm" || failed.Diagnostic.Start == 0 {
				t.Fatalf("typed diagnostic: %+v", failed.Diagnostic)
			}
		})
	}
}

func TestCompilerJITPhysicalAssemblyPreservesFlags(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("executes linux/amd64 code")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	assembly := []byte(`rtgasm 2
assembly {
answer(out:emitter) {
 let done = new_label()
 move_immediate(register(rax), int64(40))
 move_immediate(register(rdx), int64(2))
 compare(register(rax), register(rax))
 move_immediate(register(rcx), int64(0))
 branch(condition(eq), done)
 move_immediate(register(rax), int64(99))
 bind_label(done)
 add(register(rax), register(rdx))
 return()
}
}
`)
	for name, source := range map[string][]byte{
		"go.mod":        []byte("module example.com/physical\n"),
		"main.go":       []byte("package main\nfunc answer() int\nfunc appMain() int { return answer()-42 }\n"),
		"answer.rtgasm": assembly,
	} {
		if err := os.WriteFile(filepath.Join(project, name), source, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	definition := filepath.Join(root, "backend", "definitions", "linux_amd64.rtg")
	backend := New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"), backendJITTestCacheDir, backendcompiled.Backend{})
	result := driver.CompileFromFS([]string{"-backend", definition, "-t", "linux/amd64", "-s", "-o", "image", "."}, project, filepath.Join(root, "std"), driver.OSFS{}, backend)
	if !result.Ok {
		t.Fatalf("physical compile: %+v", result.Diagnostic)
	}
	image := filepath.Join(project, "image")
	if err := os.WriteFile(image, result.Binary, 0o700); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command(image).CombinedOutput(); err != nil || len(output) != 0 {
		t.Fatalf("physical flags/order/branch semantics: %s %v", output, err)
	}
}
