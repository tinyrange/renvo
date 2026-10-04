package main

import (
	"bytes"
	"crypto/sha256"
	"testing"
)

func TestDarwinSHA256BlockAndPageBoundaries(t *testing.T) {
	for _, size := range []int{55, 56, 63, 64, 65, 119, 120, 127, 128, 129, 16383, 16384, 16385} {
		data := make([]byte, size)
		for i := range data {
			data[i] = byte(i*31 + 17)
		}
		want := sha256.Sum256(data)
		if got := renvoDarwinSHA256(data); !bytes.Equal(got, want[:]) {
			t.Fatalf("SHA-256 length %d = %x, want %x", size, got, want)
		}
	}
}
