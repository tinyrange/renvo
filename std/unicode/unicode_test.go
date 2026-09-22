package unicode

import "testing"

func TestClassification(t *testing.T) {
	for _, r := range []rune{'A', 'é', 'Ω', '中', '𐐀'} {
		if !IsLetter(r) {
			t.Fatal("letter", r)
		}
	}
	for _, r := range []rune{'0', '٣', '９', '𝟠'} {
		if !IsDigit(r) {
			t.Fatal("digit", r)
		}
	}
	for _, r := range []rune{' ', '\t', '\n', '\u0085', '\u00a0', '\u2003', '\u3000'} {
		if !IsSpace(r) {
			t.Fatal("space", r)
		}
	}
	for _, r := range []rune{-1, 0xd800, 0x110000} {
		if IsLetter(r) || IsDigit(r) || IsSpace(r) || IsUpper(r) || IsLower(r) || IsTitle(r) {
			t.Fatal("invalid category")
		}
	}
	if IsSpace('\u200b') || IsDigit('²') || IsLetter('😀') {
		t.Fatal("wrong category")
	}
}
func TestCasingAndFolding(t *testing.T) {
	if ToLower('É') != 'é' || ToUpper('ω') != 'Ω' || ToLower('𐐀') != '𐐨' || ToTitle('ǳ') != 'ǲ' {
		t.Fatal("case conversion")
	}
	if !IsUpper('Ǳ') || !IsTitle('ǲ') || !IsLower('ǳ') {
		t.Fatal("case category")
	}
	if SimpleFold('K') != 'k' || SimpleFold('k') != 'K' || SimpleFold('K') != 'K' {
		t.Fatal("Kelvin fold")
	}
	if SimpleFold('Σ') != 'ς' || SimpleFold('ς') != 'σ' || SimpleFold('σ') != 'Σ' {
		t.Fatal("sigma fold")
	}
	if To(-1, 'A') != ReplacementChar || ToLower(-1) != -1 || SimpleFold(0x110000) != 0x110000 {
		t.Fatal("invalid casing")
	}
}
