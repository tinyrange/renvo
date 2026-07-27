package main

import "testing"

func generatedAArch64Word(t *testing.T, emit func(*renvoAsm)) int {
	t.Helper()
	var assembly renvoAsm
	emit(&assembly)
	if len(assembly.code) != 4 {
		t.Fatalf("instruction size = %d, want 4", len(assembly.code))
	}
	return renvoGet32At(assembly.code, 0)
}

func TestGeneratedAArch64InstructionBindings(t *testing.T) {
	if got := generatedAArch64Word(t, func(out *renvoAsm) {
		renvoAarch64AsmRet(out)
	}); got != 0xd65f03c0 {
		t.Fatalf("ret = %#08x", got)
	}
	if got := generatedAArch64Word(t, func(out *renvoAsm) {
		renvoAarch64AsmAddRegImm(out, 0, 1, 7)
	}); got != 0x91001c20 {
		t.Fatalf("add immediate = %#08x", got)
	}
	if got := generatedAArch64Word(t, func(out *renvoAsm) {
		renvoAarch64AsmMovRegReg(out, 3, 4)
	}); got != 0xaa0403e3 {
		t.Fatalf("move register = %#08x", got)
	}
	if got := generatedAArch64Word(t, func(out *renvoAsm) {
		renvoAarch64AsmCmpRegReg(out, 1, 2)
	}); got != 0xeb02003f {
		t.Fatalf("compare register = %#08x", got)
	}
}

func TestGeneratedAArch64BranchRecordsAssemblerRelocation(t *testing.T) {
	var assembly renvoAsm
	label := renvoAsmNewLabel(&assembly)
	renvoAarch64AsmJmpLabel(&assembly, label)
	if len(assembly.code) != 4 || renvoGet32At(assembly.code, 0) != 0x14000000 {
		t.Fatalf("branch bytes = %x", assembly.code)
	}
	if len(assembly.relocs) != 2 || assembly.relocs[0] != 0 ||
		assembly.relocs[1] != int32(label) {
		t.Fatalf("branch relocations = %#v", assembly.relocs)
	}
}
