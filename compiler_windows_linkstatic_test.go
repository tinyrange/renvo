package main

import "testing"

func useWindowsAmd64LinkStaticTestTarget(t *testing.T) {
	t.Helper()
	oldArch := rtgTargetArch
	t.Cleanup(func() { rtgTargetArch = oldArch })
	rtgTargetArch = rtgArchAmd64
}

func TestWindowsAmd64LinkStaticCallAlignsEvenStackArguments(t *testing.T) {
	useWindowsAmd64LinkStaticTestTarget(t)
	var asm rtgAsm
	rtgAsmInit(&asm)
	rtgWinAmd64CallStaticImport(&asm, 0, 12)
	wantPrefix := []byte{
		0x59, 0x5a, 0x41, 0x58, 0x41, 0x59, // rcx, rdx, r8, r9
		0x48, 0x83, 0xec, 40, // shadow space plus alignment slot
		0x48, 0x8b, 0x44, 0x24, 40, // load argument 5
		0x48, 0x89, 0x44, 0x24, 32, // store it at the ABI offset
	}
	if len(asm.code) < len(wantPrefix) {
		t.Fatalf("linkstatic call length = %d, want at least %d", len(asm.code), len(wantPrefix))
	}
	for i := 0; i < len(wantPrefix); i++ {
		if asm.code[i] != wantPrefix[i] {
			t.Fatalf("linkstatic call prefix = % x, want % x", asm.code[:len(wantPrefix)], wantPrefix)
		}
	}
	// Eight stack arguments are discarded along with the 40-byte reservation.
	wantCleanup := []byte{0x48, 0x83, 0xc4, 104}
	foundCleanup := false
	for i := 0; i+len(wantCleanup) <= len(asm.code); i++ {
		matched := true
		for j := 0; j < len(wantCleanup); j++ {
			if asm.code[i+j] != wantCleanup[j] {
				matched = false
			}
		}
		if matched {
			foundCleanup = true
		}
	}
	if !foundCleanup {
		t.Fatalf("linkstatic call missing 104-byte cleanup: % x", asm.code)
	}
}

func TestWindowsAmd64LinkStaticCallKeepsOddStackArgumentsAtABIOffset(t *testing.T) {
	useWindowsAmd64LinkStaticTestTarget(t)
	var asm rtgAsm
	rtgAsmInit(&asm)
	rtgWinAmd64CallStaticImport(&asm, 0, 5)
	want := []byte{
		0x59, 0x5a, 0x41, 0x58, 0x41, 0x59,
		0x48, 0x83, 0xec, 32,
		0xff, 0x15,
	}
	if len(asm.code) < len(want) {
		t.Fatalf("linkstatic call length = %d, want at least %d", len(asm.code), len(want))
	}
	for i := 0; i < len(want); i++ {
		if asm.code[i] != want[i] {
			t.Fatalf("linkstatic call prefix = % x, want % x", asm.code[:len(want)], want)
		}
	}
}
