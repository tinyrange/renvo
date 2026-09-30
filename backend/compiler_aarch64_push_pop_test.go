package main

import "testing"

func TestAArch64AdjacentRegisterPushPop(t *testing.T) {
	for source := 0; source < 32; source++ {
		for destination := 0; destination < 32; destination++ {
			var a renvoAsm
			renvoAarch64AsmPushReg(&a, source)
			renvoAarch64AsmPopReg(&a, destination)
			if source == destination {
				if len(a.code) != 0 {
					t.Fatalf("same register %d: %x", source, a.code)
				}
			} else if len(a.code) != 4 || renvoGet32At(a.code, 0) != 0xaa0003e0|(source<<16)|destination {
				t.Fatalf("register %d to %d: %x", source, destination, a.code)
			}
			if a.lastPrimaryLoad != 0 {
				t.Fatalf("register %d to %d retained marker %d", source, destination, a.lastPrimaryLoad)
			}
			// A subsequent pop must still read its own stack slot.
			before := len(a.code)
			renvoAarch64AsmPopReg(&a, destination)
			if len(a.code) != before+4 || renvoGet32At(a.code, before) != 0xf84107e0|destination {
				t.Fatalf("register %d to %d lost following pop: %x", source, destination, a.code)
			}
		}
	}
}

func TestAArch64AdjacentPushPopBarriers(t *testing.T) {
	for _, tc := range []struct {
		name    string
		barrier func(*renvoAsm)
	}{
		{"label", func(a *renvoAsm) { renvoAsmMarkLabel(a, renvoAsmNewLabel(a)) }},
		{"relative relocation", func(a *renvoAsm) { renvoAsmAddReloc(a, 0, renvoAsmNewLabel(a)) }},
		{"absolute relocation", func(a *renvoAsm) { renvoAsmAddAbsReloc(a, 0, 0, 0) }},
		{"instruction", func(a *renvoAsm) { renvoAarch64AsmEmit(a, 0xd503201f) }},
		{"replaced register", func(a *renvoAsm) { a.code[0] = 0xe1 }},
		{"replaced encoding", func(a *renvoAsm) { a.code[1] = 0x07 }},
		{"unrecorded bytes", func(a *renvoAsm) { a.lastPrimaryLoad = 0 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var a renvoAsm
			renvoAarch64AsmPushReg(&a, 0)
			tc.barrier(&a)
			before := len(a.code)
			renvoAarch64AsmPopReg(&a, 2)
			if len(a.code) != before+4 || renvoGet32At(a.code, before) != 0xf84107e2 {
				t.Fatalf("pop folded across %s: %x", tc.name, a.code)
			}
		})
	}
}

func TestAArch64NestedPushPopKeepsOuterSlot(t *testing.T) {
	var a renvoAsm
	renvoAarch64AsmPushReg(&a, 1)
	renvoAarch64AsmPushReg(&a, 2)
	renvoAarch64AsmPopReg(&a, 2)
	renvoAarch64AsmPopReg(&a, 0)
	if len(a.code) != 8 || renvoGet32At(a.code, 0) != 0xf81f0fe1 || renvoGet32At(a.code, 4) != 0xf84107e0 {
		t.Fatalf("outer stack slot lost: %x", a.code)
	}
}

func TestAArch64PushPopCommonOperations(t *testing.T) {
	for _, register := range []int{0, 1, 2, 3} {
		a := renvoAsm{c: &renvoCompileContext{renvoTargetArch: renvoArchAarch64}}
		renvoAsmPushPrimary(&a)
		if register == 0 {
			renvoAsmPopPrimary(&a)
		} else if register == 1 {
			renvoAsmPopSecondary(&a)
		} else if register == 2 {
			renvoAsmPopTertiary(&a)
		} else {
			renvoAsmPopCallWord0(&a)
		}
		if register == 0 {
			if len(a.code) != 0 {
				t.Fatalf("common same-register push/pop: %x", a.code)
			}
		} else if len(a.code) != 4 || renvoGet32At(a.code, 0) != 0xaa0003e0|register {
			t.Fatalf("common register 0 to %d: %x", register, a.code)
		}
	}
}

func TestAArch64PushPopAllowsZeroLengthEmission(t *testing.T) {
	var a renvoAsm
	renvoAarch64AsmPushReg(&a, 9)
	renvoAarch64AsmMovRegReg(&a, 0, 0)
	renvoAarch64AsmAddRegImm(&a, 1, 1, 0)
	renvoAarch64AsmAlign(&a)
	renvoAarch64AsmPopReg(&a, 3)
	if len(a.code) != 4 || renvoGet32At(a.code, 0) != 0xaa0903e3 {
		t.Fatalf("empty emission blocked valid adjacent transfer: %x", a.code)
	}
}

func TestAArch64UnmarkedPushEncodingIsNotFolded(t *testing.T) {
	var a renvoAsm
	renvoAarch64AsmEmit(&a, 0xf81f0fe9)
	renvoAarch64AsmPopReg(&a, 3)
	if len(a.code) != 8 || renvoGet32At(a.code, 0) != 0xf81f0fe9 || renvoGet32At(a.code, 4) != 0xf84107e3 {
		t.Fatalf("unmarked instruction bytes folded: %x", a.code)
	}
}
