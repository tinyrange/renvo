package rtgunit

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
			t.Errorf("operator %q was split by the RTGU scanner", operator)
		}
	}
}
