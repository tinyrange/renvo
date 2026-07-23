package main

import (
	"encoding/binary"
	"testing"
)

func TestWindowsPE64ReservesGrowableStack(t *testing.T) {
	for _, target := range []int{renvoTargetWindowsAmd64, renvoTargetWindowsArm64} {
		renvoSetTarget(target)
		image := renvoAppendPEHeader64(nil, 0x200, 0x200, 0x2000, 0x200, 0x200, 0, 0, 0, 0)
		peOffset := int(binary.LittleEndian.Uint32(image[0x3c:]))
		optionalHeader := peOffset + 24
		stackReserve := binary.LittleEndian.Uint64(image[optionalHeader+72:])
		stackCommit := binary.LittleEndian.Uint64(image[optionalHeader+80:])
		if stackReserve != 8<<20 {
			t.Fatalf("target %d stack reserve = %d, want %d", target, stackReserve, 8<<20)
		}
		if stackCommit >= stackReserve {
			t.Fatalf("target %d stack commit = %d, want less than reserve %d", target, stackCommit, stackReserve)
		}
	}
}

func TestWindowsAmd64ImportCallAlignsDynamicExpressionStack(t *testing.T) {
	renvoSetTarget(renvoTargetWindowsAmd64)
	var asm renvoAsm
	renvoAsmInit(&asm)
	renvoWinAmd64CallImport(&asm, renvoWinImportCloseHandle)
	// push r12; save rsp; align rsp; reserve the 32-byte shadow area.
	want := []byte{
		0x41, 0x54,
		0x49, 0x89, 0xe4,
		0x48, 0x83, 0xe4, 0xf0,
		0x48, 0x83, 0xec, 0x20,
	}
	if len(asm.code) < len(want) {
		t.Fatalf("import call length = %d, want at least %d", len(asm.code), len(want))
	}
	for i := 0; i < len(want); i++ {
		if asm.code[i] != want[i] {
			t.Fatalf("import call prefix = % x, want % x", asm.code[:len(want)], want)
		}
	}
}
