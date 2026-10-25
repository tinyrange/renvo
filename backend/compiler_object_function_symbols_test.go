package main

import (
	"bytes"
	"debug/elf"
	"testing"
)

func TestObjectImplementationSymbolUsesAuthoritativeRange(t *testing.T) {
	for _, is386 := range []bool{false, true} {
		name := []byte("__renvo_impl_leaf")
		emitter := renvoAsm{
			code:       []byte{0xc3},
			labelPos:   []int32{0},
			symbolName: name,
			symbols: []renvoAsmSymbol{{
				label: 0, nameEnd: len(name),
			}},
			objectFunctions: []renvoObjectFunctionRange{{
				label: 0, end: 1, nameEnd: len(name),
			}},
		}
		data := renvoAsmImageRelocatableObjectX86(&emitter, is386)
		file, err := elf.NewFile(bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		symbols, err := file.Symbols()
		file.Close()
		if err != nil {
			t.Fatal(err)
		}
		count := 0
		for _, symbol := range symbols {
			if symbol.Name == string(name) {
				count++
				if symbol.Size != 1 || elf.ST_BIND(symbol.Info) != elf.STB_LOCAL || elf.ST_TYPE(symbol.Info) != elf.STT_FUNC {
					t.Fatalf("386=%v: implementation symbol = %#v", is386, symbol)
				}
			}
		}
		if count != 1 {
			t.Fatalf("386=%v: got %d implementation symbols, want one", is386, count)
		}
	}
}
