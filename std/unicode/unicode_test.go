package unicode

import "testing"

func TestClassificationAndCase(t *testing.T) {
	letters := []rune{'A', 'é', 'Ω', 'Ж', '世', 'م'}
	for _, r := range letters {
		if !IsLetter(r) {
			t.Fatalf("not letter %U", r)
		}
	}
	digits := []rune{'7', '٧', '७', '９'}
	for _, r := range digits {
		if !IsDigit(r) || !IsNumber(r) {
			t.Fatalf("not digit %U", r)
		}
	}
	spaces := []rune{32, 9, 10, 0x00a0, 0x2003, 0x2028}
	for _, r := range spaces {
		if !IsSpace(r) {
			t.Fatalf("not space %U", r)
		}
	}
	if !IsUpper('Ω') || !IsLower('ω') || ToLower('Ω') != 'ω' || ToUpper('ж') != 'Ж' || ToUpper('世') != '世' {
		t.Fatal("case mapping")
	}
}
