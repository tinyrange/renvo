package check

import (
	"renvo.dev/internal/syntax"
	"testing"
)

func TestParseConstIntDoesNotWrap(t *testing.T) {
	for _, text := range []string{"18446744073709551616", "0xffffffffffffffff", "0x10000000000000000", "02000000000000000000000"} {
		file := syntax.ParseFile([]byte("package main\nconst n = " + text))
		if !file.Ok {
			t.Fatal("invalid fixture", text)
		}
		for i, token := range file.Tokens {
			if token.KindLine&255 == syntax.TokenNumber {
				if value, ok := parseConstInt(file, i); ok {
					t.Fatalf("%s evaluated as truncated %d", text, value)
				}
			}
		}
	}
}
