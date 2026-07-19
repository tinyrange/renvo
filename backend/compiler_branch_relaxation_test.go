package main

import "testing"

func TestAmd64BranchRelaxationRunsForFixedTarget(t *testing.T) {
	oldArch := renvoTargetArch
	oldFixed := renvoFixedTarget
	defer func() {
		renvoTargetArch = oldArch
		renvoFixedTarget = oldFixed
	}()

	renvoTargetArch = renvoArchAmd64
	renvoFixedTarget = renvoTargetLinuxAmd64
	var asm renvoAsm
	renvoAsmInit(&asm)
	target := renvoAsmNewLabel(&asm)
	renvoAsmJmpLabel(&asm, target)
	for i := 0; i < 4; i++ {
		renvoAsmEmit8(&asm, 0x90)
	}
	renvoAsmMarkLabel(&asm, target)
	before := len(asm.code)
	renvoAsmPatch(&asm)

	if len(asm.code) != before-3 {
		t.Fatalf("fixed-target branch length = %d, want %d", len(asm.code), before-3)
	}
	if asm.code[0] != 0xeb || asm.code[1] != 4 {
		t.Fatalf("relaxed branch = % x, want eb 04", asm.code[:2])
	}
}
