package c11

import "testing"

func TestSizeofTypeNameVersusExpression(t *testing.T) {
	for _, tc := range []struct {
		expr  string
		valid bool
	}{{"sizeof(word)", true}, {"sizeof((word))", false}, {"sizeof(((word)))", false}, {"sizeof word", false}, {"sizeof((x))", true}, {"sizeof((word)1)", true}} {
		r := TranslateObject("main", []byte("typedef unsigned long word; int f(void) {word x=0; return "+tc.expr+";}"), nil)
		if r.Ok != tc.valid {
			t.Errorf("%s: ok=%v error=%d source=%s", tc.expr, r.Ok, r.Error, r.Source)
		}
	}
}
