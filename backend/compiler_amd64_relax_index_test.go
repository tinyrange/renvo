package main

import (
	"bytes"
	"testing"
)

func TestAmd64BranchRelaxationMapsUnorderedLabelsAcrossSpans(t *testing.T) {
	starts := []int{1, 248, 256, 500, 512, 758, 768, 1000}
	positions := []int{0, 7, 255, 256, 262, 511, 512, 518, 767, 768, 774, 1023, 1024}
	asm := renvoAsm{code: bytes.Repeat([]byte{0x90}, 1024)}
	labels := make([]int, len(positions))
	// Forward destinations can be allocated in any order before emission.
	for i := len(positions) - 1; i >= 0; i-- {
		labels[i] = renvoAsmNewLabel(&asm)
		asm.labelPos[labels[i]] = int32(positions[i])
	}
	var expected []byte
	read := 0
	for i, start := range starts {
		size, offset, opcode := 5, 1, byte(0xeb)
		if i%2 != 0 {
			size, offset, opcode = 6, 2, 0x75
			copy(asm.code[start:], []byte{0x0f, 0x85, 0, 0, 0, 0})
		} else {
			copy(asm.code[start:], []byte{0xe9, 0, 0, 0, 0})
		}
		target := renvoAsmNewLabel(&asm)
		asm.labelPos[target] = int32(start + size + 1)
		renvoAsmAddReloc(&asm, start+offset, target)
		expected = append(expected, bytes.Repeat([]byte{0x90}, start-read)...)
		expected = append(expected, opcode, 1)
		read = start + size
	}
	expected = append(expected, bytes.Repeat([]byte{0x90}, len(asm.code)-read)...)
	renvoAmd64RelaxBranches(&asm)
	if !bytes.Equal(asm.code, expected) || len(asm.relocs) != 0 {
		t.Fatalf("branch code or displacements differ: code=%x relocs=%v", asm.code, asm.relocs)
	}
	for i, position := range positions {
		want := position
		for j, start := range starts {
			if start < position {
				want -= 3 + j%2
			}
		}
		if got := int(asm.labelPos[labels[i]]); got != want {
			t.Fatalf("label at original position %d moved to %d, want %d", position, got, want)
		}
	}
}
