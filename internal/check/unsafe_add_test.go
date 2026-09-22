package check

import (
	"renvo.dev/internal/syntax"
	"testing"
)

func TestUnsafeAddFractionalDecimal(t *testing.T) {
	for _, tc := range []struct {
		text       string
		fractional bool
	}{
		{"1.5", true}, {"1e-1", true}, {"1.00000000000000000001", true},
		{"-.01", true}, {"100e-3", true}, {"1e-9999999999999999999", true},
		{"1.0", false}, {"100e-2", false}, {"0e-99999999", false},
		{"2", false}, {"20e-1", false}, {"1e999999999999999999", false},
	} {
		file := syntax.ParseFile([]byte("package main\nvar x = " + tc.text))
		start := -1
		end := len(file.Tokens)
		for i := range file.Tokens {
			if tokenTextIs(&file, i, "=") {
				start = i + 1
			}
			if file.Tokens[i].KindLine&255 == syntax.TokenEOF {
				end = i
				break
			}
		}
		start, end = trimExprSpan(file, start, end)
		if got := unsafeAddFractionalDecimal(file, start, end); got != tc.fractional {
			t.Errorf("%s: fractional=%v, want %v", tc.text, got, tc.fractional)
		}
	}
}
