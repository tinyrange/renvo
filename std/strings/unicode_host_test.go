//go:build !renvo

package strings

import (
	"reflect"
	standard "strings"
	"testing"
	"unicode"
)

func TestUnicodeStringsAgainstGo(t *testing.T) {
	for _, s := range []string{"", "ASCII", "\u00a0É\u2003Ω𐐀\u3000", "ǳǲǱ", "KkKΣσς", "\xff\x80a\xc0\xaf", "١a２b"} {
		if ToLower(s) != standard.ToLower(s) || ToUpper(s) != standard.ToUpper(s) || ToTitle(s) != standard.ToTitle(s) || TrimSpace(s) != standard.TrimSpace(s) {
			t.Fatalf("case/trim %q", s)
		}
		if !reflect.DeepEqual(Fields(s), standard.Fields(s)) || !reflect.DeepEqual(Split(s, ""), standard.Split(s, "")) || Count(s, "") != standard.Count(s, "") {
			t.Fatalf("fields/split/count %q", s)
		}
		if IndexFunc(s, unicode.IsLetter) != standard.IndexFunc(s, unicode.IsLetter) || LastIndexFunc(s, unicode.IsLetter) != standard.LastIndexFunc(s, unicode.IsLetter) {
			t.Fatalf("index %q", s)
		}
		for _, other := range []string{s, standard.ToUpper(s), standard.ToLower(s), "kΣ𐐨", ""} {
			if EqualFold(s, other) != standard.EqualFold(s, other) {
				t.Fatalf("fold %q %q", s, other)
			}
		}
		for _, sep := range []string{"", "a", "é", " "} {
			for _, n := range []int{-1, 0, 1, 2, 5} {
				if !reflect.DeepEqual(SplitN(s, sep, n), standard.SplitN(s, sep, n)) {
					t.Fatalf("SplitN %q %q %d", s, sep, n)
				}
				if Replace(s, sep, "X", n) != standard.Replace(s, sep, "X", n) {
					t.Fatalf("Replace %q %q %d", s, sep, n)
				}
			}
		}
	}
}
