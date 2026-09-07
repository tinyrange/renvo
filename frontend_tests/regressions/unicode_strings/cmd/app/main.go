package main

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func main() {
	if !unicode.IsLetter('𐐀') || !unicode.IsDigit('𝟠') || !unicode.IsSpace('\u2003') {
		panic("classification")
	}
	if strings.ToLower("ÉΩ𐐀") != "éω𐐨" || unicode.ToTitle('ǳ') != 'ǲ' {
		panic("casing")
	}
	if !strings.EqualFold("KΣ𐐀", "Kς𐐨") {
		panic("folding")
	}
	if strings.TrimSpace("\u00a0text\u3000") != "text" {
		panic("whitespace")
	}
	parts := strings.Fields("\u2003one\u3000two\u00a0")
	if len(parts) != 2 || parts[0] != "one" || parts[1] != "two" {
		panic("fields")
	}
	parts = strings.SplitN("é𐐀z", "", 2)
	if len(parts) != 2 || parts[1] != "𐐀z" {
		panic("split")
	}
	if strings.Replace("é𐐀", "", "-", -1) != "-é-𐐀-" {
		panic("replace")
	}
	r, n := utf8.DecodeLastRuneInString("x𐐀")
	if r != '𐐀' || n != 4 || utf8.Valid([]byte{255}) {
		panic("UTF8")
	}
	print("PASS\n")
}
