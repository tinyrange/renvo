package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestAmd64BranchRelaxationMapsOutOfOrderRelocations(t *testing.T) {
	a := renvoAsm{code: bytes.Repeat([]byte{0x90}, 128)}
	short := renvoAsmNewLabel(&a)
	a.labelPos[short] = 106
	copy(a.code[100:], []byte{0xe9, 0, 0, 0, 0})
	renvoAsmAddReloc(&a, 101, short)
	// Helper emission may append its own relocation before the caller's.
	caller := renvoAsmNewLabel(&a)
	a.labelPos[caller] = 127
	renvoAsmAddReloc(&a, 3, caller)
	renvoAsmAddAbsReloc(&a, 110, 0, 0)
	renvoAsmAddAbsReloc(&a, 8, 0, 0)
	renvoAmd64RelaxBranches(&a)
	if len(a.relocs) != 2 || a.relocs[0] != 3 || a.relocs[1] != int32(caller) {
		t.Fatalf("caller relocation mapped incorrectly: %v", a.relocs)
	}
	if a.absRelocs[0] != 107 || a.absRelocs[3] != 8 {
		t.Fatalf("absolute relocations mapped incorrectly: %v", a.absRelocs)
	}
	if a.code[100] != 0xeb || a.code[101] != 1 {
		t.Fatalf("short branch changed: %x", a.code[100:102])
	}
}

func TestAmd64StringBoundsHelperRelocations(t *testing.T) {
	for _, target := range supportedCompilerTargets(t) {
		if target.name != "linux/amd64" {
			continue
		}
		skipIfTargetRunnerMissing(t, target)
		source, err := os.ReadFile("tests/string_bounds_helper_relocations.go")
		if err != nil {
			t.Fatal(err)
		}
		image, ok := RenvoCompileSourceToBytes(source, target.name)
		if !ok {
			t.Fatal("failed to compile bounds helper regression")
		}
		output := filepath.Join(t.TempDir(), "string-bounds")
		if err := os.WriteFile(output, image, 0755); err != nil {
			t.Fatal(err)
		}
		for _, mode := range []string{"high", "negative"} {
			result, err := runTargetCommand(t, target, output, mode)
			if err != nil || result.exitCode != 2 || result.stdout != "" || result.stderr != "panic\n" {
				t.Fatalf("%s index: result=%+v error=%v", mode, result, err)
			}
		}
	}
}
