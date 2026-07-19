package main

import "testing"

func readStackFrameInstruction(code []byte, at int) int {
	return int(code[at]) | int(code[at+1])<<8 | int(code[at+2])<<16 | int(code[at+3])<<24
}

func TestStackPeakRetainsReleasedTemporaryStorage(t *testing.T) {
	var g renvoLinearGen
	g.stackUsed = 48
	renvoRecordStackPeak(&g)
	g.stackUsed = 16
	renvoRecordStackPeak(&g)

	if g.stackPeak != 48 {
		t.Fatalf("stack peak = %d, want 48", g.stackPeak)
	}
}

func TestAarch64FrameUsesCalculatedStorage(t *testing.T) {
	oldOS := targetOS
	t.Cleanup(func() { targetOS = oldOS })
	targetOS = renvoOSLinux

	var asm renvoAsm
	at := renvoAarch64AsmFrameStart(&asm)
	renvoAarch64AsmPatchFrame(&asm, at, 24)

	if got := readStackFrameInstruction(asm.code, at); got != 0xd503201f {
		t.Fatalf("high frame instruction = %#x, want nop", got)
	}
	if got := readStackFrameInstruction(asm.code, at+4); got != 0xd10083ff {
		t.Fatalf("low frame instruction = %#x, want sub sp, sp, #32", got)
	}
}

func TestWindowsAarch64FrameProbesCalculatedPages(t *testing.T) {
	oldOS := targetOS
	t.Cleanup(func() { targetOS = oldOS })
	targetOS = renvoOSWindows

	var asm renvoAsm
	at := renvoAarch64AsmFrameStart(&asm)
	renvoAarch64AsmPatchFrame(&asm, at, 8193)

	if got := readStackFrameInstruction(asm.code, at); got != 0xd2800049 {
		t.Fatalf("page-count instruction = %#x, want two pages", got)
	}
	if got := readStackFrameInstruction(asm.code, at+24); got != 0xd10043ff {
		t.Fatalf("tail instruction = %#x, want sub sp, sp, #16", got)
	}
}

func TestArmFrameUsesCalculatedStorage(t *testing.T) {
	var asm renvoAsm
	at := renvoArmAsmFrameStart(&asm)
	renvoArmAsmPatchFrame(&asm, at, 24)

	if got := readStackFrameInstruction(asm.code, at); got != 0xe3009018 {
		t.Fatalf("frame instruction = %#x, want movw r9, #24", got)
	}
}
