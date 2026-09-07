package syntax

import "testing"

func TestRuneLiteralValue(t *testing.T) {
	for text, want := range map[string]int{
		"'a'": 97, "'界'": 30028, "'😀'": 128512, "'\\xff'": 255, "'\\377'": 255, "'\\u754c'": 30028, "'\\U0001f600'": 128512, "'\\n'": 10, "'\\''": 39, "'\\\\'": 92, "'\"'": 34, "'\\000'": 0,
	} {
		src := []byte(text)
		tok := MakeToken(TokenChar, 0, len(src), 1)
		if value, ok := RuneLiteralValue(src, tok); !ok || value != want {
			t.Errorf("%s: %d %v", text, value, ok)
		}
		var scanner Scanner
		scanner.Scan(src)
		if !scanner.Ok {
			t.Errorf("scanner rejected %s", text)
		}
	}
	for _, text := range []string{"''", "'ab'", "'界a'", "'\\q'", "'\\400'", "'\\uD800'", "'\\U00110000'", "'\\\"'", "'\\12'", "'\\x0'", "'\n'", "'\xff'"} {
		src := []byte(text)
		if _, ok := RuneLiteralValue(src, MakeToken(TokenChar, 0, len(src), 1)); ok {
			t.Errorf("decoded invalid %s", text)
		}
		var scanner Scanner
		scanner.Scan(src)
		if scanner.Ok {
			t.Errorf("scanner accepted invalid %s", text)
		}
	}
}
