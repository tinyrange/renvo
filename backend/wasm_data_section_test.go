package main

import (
	"bytes"
	"testing"
)

func TestWasmDirectDataSectionEncoding(t *testing.T) {
	for _, n := range []int{0, 1, 127, 128, 16383, 16384, 1048576} {
		data := make([]byte, n)
		for i := range data {
			data[i] = byte(i * 37)
		}
		for _, base := range []int{0, 127, 128, 65536, 2147483647} {
			var want, got renvoWasmBuffer
			renvoWasmAppendEncoded(&want, "prefix")
			renvoWasmAppendEncoded(&got, "prefix")
			renvoWasmAppendSection(&want, 11, renvoWasm32DataSectionFull(base, data))
			renvoWasm32AppendDataSectionDirect(&got, base, data)
			if !bytes.Equal(want.data[:want.length], got.data[:got.length]) {
				t.Fatalf("data=%d base=%d: encoded bytes differ", n, base)
			}
		}
	}
}
