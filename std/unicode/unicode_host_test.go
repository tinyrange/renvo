//go:build !renvo

package unicode

import (
	"testing"
	standard "unicode"
)

func TestAllCodePointsAgainstGo(t *testing.T) {
	if Version != standard.Version {
		t.Fatal("Unicode version mismatch", Version, standard.Version)
	}
	for r := rune(-1); r <= MaxRune+1; r++ {
		if IsLetter(r) != standard.IsLetter(r) || IsDigit(r) != standard.IsDigit(r) || IsSpace(r) != standard.IsSpace(r) || IsUpper(r) != standard.IsUpper(r) || IsLower(r) != standard.IsLower(r) || IsTitle(r) != standard.IsTitle(r) {
			t.Fatalf("classification U+%X", r)
		}
		if ToUpper(r) != standard.ToUpper(r) || ToLower(r) != standard.ToLower(r) || ToTitle(r) != standard.ToTitle(r) || SimpleFold(r) != standard.SimpleFold(r) {
			t.Fatalf("case mapping U+%X", r)
		}
	}
}
