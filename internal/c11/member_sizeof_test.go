package c11

import "testing"

func TestSizeofMemberValidation(t *testing.T) {
	for _, expr := range []string{"s.missing", "s.missing.value", "(&s)->missing", "s.good.missing"} {
		source := []byte("struct inner { long value; }; struct outer { struct inner good; }; int probe(void) { struct outer s; return sizeof(" + expr + "); }")
		r := TranslateObject("main", source, nil)
		if r.Ok {
			t.Errorf("accepted invalid sizeof(%s): %s", expr, r.Source)
		}
	}
	r := TranslateObject("main", []byte("struct inner { long value; }; struct outer { struct inner good; }; int probe(void) { struct outer s; return sizeof(s.good.value) + sizeof((&s)->good); }"), nil)
	if !r.Ok {
		t.Fatalf("valid members rejected: %+v", r)
	}
}
