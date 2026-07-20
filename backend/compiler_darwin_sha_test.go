package main

import (
	"encoding/hex"
	"testing"
)

func TestDarwinSHA256KnownVectors(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
		{"abc", "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"},
		{"The quick brown fox jumps over the lazy dog", "d7a8fbb307d7809469ca9abcb0082e4f8d5651e46d3cdb762d02d0bf37c9e592"},
	}
	for _, test := range tests {
		got := hex.EncodeToString(renvoDarwinSHA256([]byte(test.input)))
		if got != test.want {
			t.Fatalf("SHA-256(%q) = %s, want %s", test.input, got, test.want)
		}
	}
}
