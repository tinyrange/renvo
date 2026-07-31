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

func TestAmd64StrippedImageGrowsPastCodeCapacity(t *testing.T) {
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, true, false, false)
	var asm renvoAsm
	asm.c = context
	asm.codeOffset = renvoAmd64ELFCodeOffset
	asm.code = make([]byte, 1, 1)
	asm.code[0] = 0xc3
	asm.data = make([]byte, 512)
	asm.data[len(asm.data)-1] = 0x5a

	image := renvoAsmImageAmd64(&asm)
	wantSize := renvoAmd64ELFCodeOffset + 1 + len(asm.data)
	if len(image) != wantSize {
		t.Fatalf("image length = %d, want %d", len(image), wantSize)
	}
	if image[renvoAmd64ELFCodeOffset] != 0xc3 {
		t.Fatalf("code byte = %#x, want RET", image[renvoAmd64ELFCodeOffset])
	}
	if image[len(image)-1] != 0x5a {
		t.Fatalf("last data byte = %#x, want 0x5a", image[len(image)-1])
	}
}
