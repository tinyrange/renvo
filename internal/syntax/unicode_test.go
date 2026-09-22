package syntax

import (
	"testing"
	"unicode"
)

func TestUnicodeIdentifierCategories(t *testing.T) {
	for r := rune(0); r <= unicode.MaxRune; r++ {
		if unicodeRangeContains(unicodeLetterRanges, int(r)) != unicode.IsLetter(r) ||
			unicodeRangeContains(unicodeDigitRanges, int(r)) != unicode.IsDigit(r) ||
			unicodeRangeContains(unicodeUpperRanges, int(r)) != unicode.IsUpper(r) {
			t.Fatalf("Unicode classification differs for U+%04X", r)
		}
	}
	for _, name := range []string{"Ω", "É", "𐐀"} {
		if !IdentifierExported([]byte(name), 0) {
			t.Fatalf("not exported: %s", name)
		}
	}
	for _, name := range []string{"π", "世界", "ǅ", "_"} {
		if IdentifierExported([]byte(name), 0) {
			t.Fatalf("incorrectly exported: %s", name)
		}
	}
}
