package utf8

import "testing"

func TestDecodeRuneBytes(t *testing.T) {
	for _, source := range []string{"", "a", "é", "€", "😀", "😀tail", "\xff", "\xc0\x80", "\xed\xa0\x80", "\xf4\x90\x80\x80", "\xe2\x82"} {
		want, width := DecodeRuneInString(source)
		got, size := DecodeRune([]byte(source))
		if got != want || size != width {
			t.Fatal("byte decoder mismatch", source)
		}
		if Valid([]byte(source)) != ValidString(source) {
			t.Fatal("byte validity", source)
		}
	}
}

func TestLastRuneAndInvalidEncoding(t *testing.T) {
	for _, source := range []string{"", "a", "é", "😀", "x😀", "x\xff", "é\x80", "\xe2\x82"} {
		var want rune = RuneError
		width := 0
		for _, r := range source {
			want = r
		}
		if len(source) > 0 {
			for i := range source {
				width = len(source) - i
			}
		}
		got, size := DecodeLastRuneInString(source)
		if got != want || size != width {
			t.Fatal("last rune", source)
		}
	}
	for _, r := range []rune{-1, 0xd800, 0x110000} {
		var buf [4]byte
		n := EncodeRune(buf[:], r)
		if n != 3 || string(buf[:n]) != "�" {
			t.Fatal("invalid rune encoding")
		}
	}
}
