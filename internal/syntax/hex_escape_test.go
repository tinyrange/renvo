package syntax

import (
	"fmt"
	"testing"
)

func TestHexByteEscapeBoundaries(t *testing.T) {
	for value := 0; value < 256; value++ {
		for _, format := range []string{`\x%02x`, `\x%02X`} {
			escape := fmt.Sprintf(format, value)
			next, got, unicode, ok := stringEscapeValue([]byte(escape), 0, len(escape))
			if !ok || unicode || next != 4 || got != value {
				t.Fatalf("%s: %d %d %v %v", escape, next, got, unicode, ok)
			}
			quoted := []byte(`"` + escape + `"`)
			decoded, ok := StringLiteralValue(quoted, Token{KindLine: TokenString, End: int32(len(quoted))})
			if !ok || len(decoded) != 1 || int(decoded[0]) != value {
				t.Fatalf("decode %s: %q %v", quoted, decoded, ok)
			}
			for end := 0; end < 4; end++ {
				if _, _, _, ok := stringEscapeValue([]byte(escape), 0, end); ok {
					t.Fatalf("accepted truncated %s at %d", escape, end)
				}
			}
		}
	}
	for _, escape := range []string{`\xg0`, `\x0g`, `\x-1`, `\x 1`, `\x1"`} {
		if _, _, _, ok := stringEscapeValue([]byte(escape), 0, len(escape)); ok {
			t.Fatalf("accepted %s", escape)
		}
	}
}
