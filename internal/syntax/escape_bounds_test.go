package syntax

import "testing"

func TestEscapeAccumulatorBounds(t *testing.T) {
	for _, escape := range []string{`\U80000000`, `\Uffffffff`, `\U00110000`, `\U0000d800`, `\400`, `\777`} {
		if _, _, _, ok := stringEscapeValue([]byte(escape), 0, len(escape)); ok {
			t.Errorf("accepted %s", escape)
		}
		for _, quote := range []string{"\"", "'"} {
			var scanner Scanner
			scanner.Scan([]byte(quote + escape + quote))
			if scanner.Ok {
				t.Errorf("scanner accepted %s%s%s", quote, escape, quote)
			}
		}
	}
	for text, want := range map[string]int{`\U0010ffff`: 0x10ffff, `\U00010000`: 0x10000, `\uD7ff`: 0xd7ff, `\ue000`: 0xe000, `\377`: 255, `\xff`: 255, `\000`: 0} {
		next, value, _, ok := stringEscapeValue([]byte(text), 0, len(text))
		if !ok || next != len(text) || value != want {
			t.Errorf("%s: %d %d %v", text, next, value, ok)
		}
	}
}
