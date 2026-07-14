package main

import "testing"

func TestAmd64BranchRelaxationRunsForFixedTarget(t *testing.T) {
	oldArch := rtgTargetArch
	oldFixed := rtgCompilerFixedTarget
	defer func() {
		rtgTargetArch = oldArch
		rtgCompilerFixedTarget = oldFixed
	}()

	rtgTargetArch = rtgArchAmd64
	rtgCompilerFixedTarget = rtgTargetLinuxAmd64
	var asm rtgAsm
	rtgAsmInit(&asm)
	target := rtgAsmNewLabel(&asm)
	rtgAsmJmpLabel(&asm, target)
	for i := 0; i < 4; i++ {
		rtgAsmEmit8(&asm, 0x90)
	}
	rtgAsmMarkLabel(&asm, target)
	before := len(asm.code)
	rtgAsmPatch(&asm)

	if len(asm.code) != before-3 {
		t.Fatalf("fixed-target branch length = %d, want %d", len(asm.code), before-3)
	}
	if asm.code[0] != 0xeb || asm.code[1] != 4 {
		t.Fatalf("relaxed branch = % x, want eb 04", asm.code[:2])
	}
}
