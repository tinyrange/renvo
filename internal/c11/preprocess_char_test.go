package c11

import "testing"

func TestParsePreprocessorCharacterEscapes(t *testing.T) {
	for _, tc := range []struct {
		text  string
		value int64
		ok    bool
	}{
		{`L'\0'`, 0, true}, {`u'\101'`, 65, true}, {`'\7'`, 7, true},
		{`U'\x41'`, 65, true}, {`'\xAF'`, 175, true}, {`'\x'`, 0, false},
		{`'\8'`, 0, false}, {`'\1234'`, 83, false}, {`'\xG'`, 0, false},
	} {
		value, ok := ppParseChar([]byte(tc.text))
		if value != tc.value || ok != tc.ok {
			t.Fatalf("%s: (%d,%v), want (%d,%v)", tc.text, value, ok, tc.value, tc.ok)
		}
	}
}
