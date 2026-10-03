package main

import "testing"

func TestAmd64BranchRelaxationLargeCode(t *testing.T) {
	oldArch, oldFixed := renvoTargetArch, renvoFixedTarget
	defer func() { renvoTargetArch, renvoFixedTarget = oldArch, oldFixed }()
	renvoTargetArch, renvoFixedTarget = renvoArchAmd64, renvoTargetLinuxAmd64
	var asm renvoAsm
	renvoAsmInit(&asm)
	target := renvoAsmNewLabel(&asm)
	renvoAsmJmpLabel(&asm, target)
	renvoAsmEmit8(&asm, 0x90)
	renvoAsmMarkLabel(&asm, target)
	for len(asm.code) < 1048600 {
		renvoAsmEmit8(&asm, 0x90)
	}
	before := len(asm.code)
	renvoAsmPatch(&asm)
	if len(asm.code) != before-3 || asm.code[0] != 0xeb || asm.code[1] != 1 {
		t.Fatal("large code did not retain branch relaxation")
	}
}

func TestAarch64ConditionalBranchesCrossOneMiB(t *testing.T) {
	oldArch, oldFixed := renvoTargetArch, renvoFixedTarget
	defer func() { renvoTargetArch, renvoFixedTarget = oldArch, oldFixed }()
	renvoTargetArch, renvoFixedTarget = renvoArchAarch64, renvoTargetLinuxAarch64
	var asm renvoAsm
	renvoAsmInit(&asm)
	backward, forward := renvoAsmNewLabel(&asm), renvoAsmNewLabel(&asm)
	renvoAsmMarkLabel(&asm, backward)
	renvoAarch64AsmBCondLabel(&asm, forward, 0)
	asm.code = append(asm.code, make([]byte, 1048576)...)
	renvoAsmMarkLabel(&asm, forward)
	at := len(asm.code)
	renvoAarch64AsmBCondLabel(&asm, backward, 1)
	renvoAsmPatch(&asm)
	if asm.patchFailed {
		t.Fatal("patch failed")
	}
	if renvoGet32At(asm.code, 0) != 0x54000041 || renvoGet32At(asm.code, at) != 0x54000040 {
		t.Fatal("condition must skip the long branch when inverted")
	}
	if renvoGet32At(asm.code, 4) != 0x14000000|((at-4)/4) || renvoGet32At(asm.code, at+4) != 0x14000000|((-(at+4)/4)&0x03ffffff) {
		t.Fatal("long forward or backward branch was truncated")
	}
}
