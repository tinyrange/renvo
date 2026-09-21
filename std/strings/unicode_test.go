package strings

import (
	"testing"
	"unicode"
)

func TestUnicodeStringOperations(t *testing.T) {
	if Replace("é𐐀", "", "-", -1) != "-é-𐐀-" || Replace("é𐐀", "", "-", 2) != "-é-𐐀" {
		t.Fatal("empty replacement")
	}
	limited := SplitN("é𐐀z", "", 2)
	if len(limited) != 2 || limited[0] != "é" || limited[1] != "𐐀z" {
		t.Fatal("limited rune split")
	}
	if Compare("é", "z") != 1 || IndexByte("é!", '!') != 2 {
		t.Fatal("byte compare/search")
	}
	if ToLower("ÉΩ𐐀") != "éω𐐨" || ToUpper("éω𐐨") != "ÉΩ𐐀" || ToTitle("ǳ") != "ǲ" {
		t.Fatal("Unicode casing")
	}
	if !EqualFold("KΣ𐐀", "Kς𐐨") || EqualFold("ß", "ss") {
		t.Fatal("simple fold")
	}
	if TrimSpace("\u00a0\u2003hello\u3000") != "hello" {
		t.Fatal("Unicode trim")
	}
	parts := Fields("\u00a0one\u2003two\u3000")
	if len(parts) != 2 || parts[0] != "one" || parts[1] != "two" {
		t.Fatal("Unicode fields")
	}
	if IndexFunc("é３!", unicode.IsDigit) != 2 || LastIndexFunc("é３!٤", unicode.IsDigit) != 6 {
		t.Fatal("rune byte offsets")
	}
	if Trim("éΩwordΩé", "éΩ") != "word" || TrimRightFunc("word٣٤", unicode.IsDigit) != "word" {
		t.Fatal("rune trimming")
	}
	if Count("\xff\x80", "") != 3 {
		t.Fatal("invalid UTF8 rune count")
	}
	parts = Split("é𐐀\xff", "")
	if len(parts) != 3 || parts[0] != "é" || parts[1] != "𐐀" || parts[2] != "\xff" {
		t.Fatal("rune splitting")
	}
}
