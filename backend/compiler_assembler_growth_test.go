package main

import "testing"

func TestAssemblerEmissionGrowsPastCapacity(t *testing.T) {
	var asm renvoAsm
	asm.code = make([]byte, 1, 1)
	asm.code[0] = 0x7f

	for i := 0; i < 4096; i++ {
		renvoAsmEmit8(&asm, i)
	}

	if len(asm.code) != 4097 {
		t.Fatalf("code length = %d, want 4097", len(asm.code))
	}
	if asm.code[0] != 0x7f {
		t.Fatalf("existing prefix changed to %#x", asm.code[0])
	}
	for i := 0; i < 4096; i++ {
		if asm.code[i+1] != byte(i) {
			t.Fatalf("emitted byte %d = %#x, want %#x", i, asm.code[i+1], byte(i))
		}
	}
}
