package main

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"fmt"
	"testing"
)

func TestWindowsAmd64RelaxesBranchesBeforeSectionLayout(t *testing.T) {
	// The second case shrinks text from just above a page to exactly one page,
	// so both import and data references must use the new data-section RVA.
	for _, prefixSize := range []int{0, 4080} {
		t.Run(fmt.Sprint(prefixSize), func(t *testing.T) {
			context := renvoNewCompileContext(renvoTargetWindowsAmd64, true, false, false)
			meta := renvoMeta{c: context}
			gen := renvoLinearGen{c: context, meta: &meta}
			asm := &gen.asm
			asm.c = context
			asm.codeOffset = renvoWinSectionRVA
			asm.code = bytes.Repeat([]byte{0x90}, prefixSize)
			asm.data = []byte{1, 2, 3, 4}
			forward := renvoAsmNewLabel(asm)
			renvoAsmEmit8(asm, 0xe9)
			branchAt := len(asm.code)
			renvoAsmEmit32(asm, 0)
			renvoAsmAddReloc(asm, branchAt, forward)
			renvoAsmEmitText(asm, "\x48\x8d\x05")
			dataAt := len(asm.code)
			renvoAsmEmit32(asm, 0)
			renvoAsmAddAbsReloc(asm, dataAt, 3, 0)
			renvoAsmEmitText(asm, "\xff\x15")
			importAt := len(asm.code)
			renvoAsmEmit32(asm, 0)
			renvoAsmAddAbsReloc(asm, importAt, renvoWinImportGetStdHandle, renvoAbsWinImportReloc)
			renvoAsmMarkLabel(asm, forward)
			renvoAsmRet(asm)

			result := renvoFinishScalarProgramAmd64(&gen)
			if !result.ok {
				t.Fatal("could not construct PE image")
			}
			file, err := pe.NewFile(bytes.NewReader(result.data))
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			text, data := file.Section(".text"), file.Section(".data")
			if text == nil || data == nil {
				t.Fatal("missing PE text or data section")
			}
			code, err := text.Data()
			if err != nil {
				t.Fatal(err)
			}
			if text.VirtualSize != uint32(prefixSize+16) || code[prefixSize] != 0xeb || code[prefixSize+1] != 13 {
				t.Fatalf("text size=%d branch=% x, want size=%d branch=eb 0d", text.VirtualSize, code[prefixSize:prefixSize+2], prefixSize+16)
			}
			wantDataRVA := uint32(renvoAlignValue(renvoWinSectionRVA+prefixSize+16, renvoWinSectionAlign))
			if data.VirtualAddress != wantDataRVA {
				t.Fatalf("data RVA=%#x, want %#x", data.VirtualAddress, wantDataRVA)
			}
			dataTarget := int64(text.VirtualAddress) + int64(prefixSize+9) + int64(int32(binary.LittleEndian.Uint32(code[prefixSize+5:])))
			if dataTarget != int64(data.VirtualAddress)+3 {
				t.Fatalf("data reference=%#x, want %#x", dataTarget, data.VirtualAddress+3)
			}
			header := file.OptionalHeader.(*pe.OptionalHeader64)
			iat := header.DataDirectory[12]
			importTarget := int64(text.VirtualAddress) + int64(prefixSize+15) + int64(int32(binary.LittleEndian.Uint32(code[prefixSize+11:])))
			if importTarget < int64(iat.VirtualAddress) || importTarget >= int64(iat.VirtualAddress+iat.Size) {
				t.Fatalf("import reference=%#x outside IAT [%#x,%#x)", importTarget, iat.VirtualAddress, iat.VirtualAddress+iat.Size)
			}
		})
	}
}
