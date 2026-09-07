package syntax

import "testing"

func TestNumberLiteralGrammar(t *testing.T) {
	for _, text := range []string{"0", "0123", "0_123", "0b_101", "0O_77", "0x_Bad_Face", "1_234", "1.", ".5", "1.2_3", "1e+2", "0_9.0", "09e1", "09i", "0_9i", "0x1i", "0b10i", "0x1.8p+1", "0x.8p1", "0x_1p-2", "1.2e3i", "0x1p2i"} {
		var scanner Scanner
		scanner.Scan([]byte(text))
		if !validNumberLiteral([]byte(text), 0, len(text)) || !scanner.Ok {
			t.Errorf("rejected %s", text)
		}
	}
	for _, text := range []string{"0x", "0b", "0o", "0b2", "0o8", "09", "0_9", "1__2", "1_", "0x_", "0x__1", "1._0", "1_.0", "1e", "1e+", "1e_1", "1e+_1", "1e1_", "0x1.2", "0x.p1", "0x_.8p1", "0x1p", "0x1p+", "0x1p_1", "0x1g", "0b1.0"} {
		if validNumberLiteral([]byte(text), 0, len(text)) {
			t.Errorf("accepted %s", text)
		}
		var scanner Scanner
		scanner.Scan([]byte(text))
		if scanner.Ok {
			t.Errorf("scanner accepted %s", text)
		}
	}
}
