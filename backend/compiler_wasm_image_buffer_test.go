package main

import (
	"bytes"
	"testing"
)

func TestWasmImageBufferPreservesImage(t *testing.T) {
	for _, dataSize := range []int{0, 173, 8193} {
		makeAsm := func() *renvoAsm {
			a := &renvoAsm{c: renvoNewCompileContext(renvoTargetWasiWasm32, false, false, false)}
			renvoWasm32EmitRegImm(a, renvoWasm32OpMovRegImm, renvoWasm32RegRax, 0)
			renvoWasm32AsmExit(a)
			a.data = make([]byte, dataSize)
			for i := range a.data {
				a.data[i] = byte(i*17 + 3)
			}
			a.bssSize = 97
			return a
		}
		want := renvoWasm32Image(makeAsm())
		buffer := rtgWasm32Wasm32PackageRenvoWasm32ImageBuffer(makeAsm())
		if !bytes.Equal(buffer.data[:buffer.length], want) {
			t.Fatalf("image differs for data size %d", dataSize)
		}
	}
}
