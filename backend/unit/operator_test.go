package unit

import "testing"

func TestScanSourceKeepsMultiByteOperatorsWhole(t *testing.T) {
	source := []byte("package p\nfunc f() { a >>= 1; a <<= 1; a &^= 1; a &= 1; a |= 1; a ^= 1 }\n")
	tokens, _ := scanSource(source)
	want := map[string]bool{
		">>=": false,
		"<<=": false,
		"&^=": false,
		"&=":  false,
		"|=":  false,
		"^=":  false,
	}
	for _, token := range tokens {
		if _, ok := want[token.text]; ok {
			want[token.text] = true
		}
	}
	for operator, found := range want {
		if !found {
			t.Errorf("operator %q was split by the RenvoUnit scanner", operator)
		}
	}
}

func TestScanSourceKeepsImaginaryLiteralsWhole(t *testing.T) {
	tokens, _ := scanSource([]byte("package p\nvar value = 2 + 1i\n"))
	for _, token := range tokens {
		if token.text == "1i" {
			return
		}
	}
	t.Fatal("imaginary literal 1i was split by the RenvoUnit scanner")
}
