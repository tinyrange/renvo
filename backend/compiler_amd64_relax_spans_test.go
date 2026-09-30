package main

import (
	"bytes"
	"fmt"
	"testing"
)

func TestAmd64BranchRelaxationCopiesSpansAtAnyImageSize(t *testing.T) {
	for _, prefixSize := range []int{0, 1 << 20} {
		t.Run(fmt.Sprint(prefixSize), func(t *testing.T) {
			asm := renvoAsm{code: bytes.Repeat([]byte{0x90}, prefixSize)}
			start := renvoAsmNewLabel(&asm)
			forward := renvoAsmNewLabel(&asm)
			end := renvoAsmNewLabel(&asm)
			renvoAsmMarkLabel(&asm, start)
			renvoAsmEmit8(&asm, 0x90)
			renvoAmd64AsmJccLabel(&asm, 0x85, forward)
			// Keep a call relocation and a data-address relocation in the span
			// between the two shortened branches. Neither instruction may change.
			renvoAsmEmit8(&asm, 0xe8)
			callAt := len(asm.code)
			renvoAsmEmit32(&asm, 0)
			renvoAsmAddReloc(&asm, callAt, start)
			renvoAsmEmitText(&asm, "\x48\x8d\x05")
			dataAt := len(asm.code)
			renvoAsmEmit32(&asm, 0)
			renvoAsmAddAbsReloc(&asm, dataAt, 17, renvoAbsBssReloc)
			renvoAsmEmitText(&asm, "\x0f\x1f\x40\x00")
			renvoAsmMarkLabel(&asm, forward)
			renvoAsmEmit8(&asm, 0xe9)
			backAt := len(asm.code)
			renvoAsmEmit32(&asm, 0)
			renvoAsmAddReloc(&asm, backAt, start)
			renvoAsmEmitText(&asm, "\x66\x90\x48\x31\xc0\xc3")
			renvoAsmMarkLabel(&asm, end)

			renvoAmd64RelaxBranches(&asm)

			want := []byte("\x90\x75\x10\xe8\x00\x00\x00\x00\x48\x8d\x05\x00\x00\x00\x00\x0f\x1f\x40\x00\xeb\xeb\x66\x90\x48\x31\xc0\xc3")
			if len(asm.code) != prefixSize+len(want) || !bytes.Equal(asm.code[prefixSize:], want) {
				t.Fatalf("relaxed code length=%d suffix=% x, want length=%d suffix=% x", len(asm.code), asm.code[prefixSize:], prefixSize+len(want), want)
			}
			if !bytes.Equal(asm.code[:prefixSize], bytes.Repeat([]byte{0x90}, prefixSize)) {
				t.Fatal("unchanged prefix was corrupted")
			}
			if int(asm.labelPos[start]) != prefixSize || int(asm.labelPos[forward]) != prefixSize+19 || int(asm.labelPos[end]) != prefixSize+len(want) {
				t.Fatalf("relaxed labels = %v", asm.labelPos)
			}
			if len(asm.relocs) != 2 || int(asm.relocs[0]) != prefixSize+4 || int(asm.relocs[1]) != start {
				t.Fatalf("remaining call relocation = %v", asm.relocs)
			}
			if len(asm.absRelocs) != 3 || int(asm.absRelocs[0]) != prefixSize+11 || asm.absRelocs[1] != 17 || asm.absRelocs[2] != renvoAbsBssReloc {
				t.Fatalf("shifted data relocation = %v", asm.absRelocs)
			}
		})
	}
}
