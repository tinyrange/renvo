package main

import (
	"strings"
	"unicode"
	"unicode/utf16"
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
	units := utf16.Encode([]rune{'A', '𐐀', '\ufffd'})
	decoded := utf16.Decode(units)
	if len(units) != 4 || units[1] != 0xd801 || units[2] != 0xdc00 || len(decoded) != 3 || decoded[1] != '𐐀' {
		panic("UTF16 round trip")
	}
	invalid := utf16.Decode([]uint16{0xd800, 'A', 0xdc00})
	if len(invalid) != 3 || invalid[0] != '\ufffd' || invalid[1] != 'A' || invalid[2] != '\ufffd' {
		panic("UTF16 replacement")
	}
	if strings.ToLower("ASCII Already lower") != "ascii already lower" || strings.ToUpper("ascii") != "ASCII" {
		panic("ASCII casing")
	}
	if strings.TrimSpace(" \t\u2003text\u00a0\r\n") != "text" {
		panic("mixed whitespace")
	}
	if strings.Map(func(r rune) rune { return r }, "sameé") != "sameé" || strings.Map(func(r rune) rune { return r }, "a\xffb") != "a\ufffdb" {
		panic("identity map")
	}
	if strings.Map(func(r rune) rune {
		if r == 'x' {
			return -1
		}
		return r
	}, "beforexxé") != "beforeé" {
		panic("map deletion after prefix")
	}
	print("PASS\n")
}
