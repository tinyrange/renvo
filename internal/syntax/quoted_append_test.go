package syntax

import "testing"

func TestAppendQuotedStringLiteralCanonicalBytes(t *testing.T) {
	for _, test := range []struct{ input, want string }{
		{`""`, `""`},
		{`"plain"`, `"plain"`},
		{`"\x61\141\u0061"`, `"aaa"`},
		{`"\a\b\f\n\r\t\v\\\""`, `"\x07\x08\x0c\x0a\x0d\x09\x0b\\\""`},
		{`"\u007f\u0080\u07ff\u0800\uffff"`, `"\x7f\xc2\x80\xdf\xbf\xe0\xa0\x80\xef\xbf\xbf"`},
		{`"\U00010000\U0010ffff"`, `"\xf0\x90\x80\x80\xf4\x8f\xbf\xbf"`},
		{`"\x00\xFFé"`, `"\x00\xff\xc3\xa9"`},
	} {
		src := []byte(test.input)
		got, ok := AppendQuotedStringLiteral([]byte("prefix:"), src, MakeToken(TokenString, 0, len(src), 1))
		if !ok || string(got) != "prefix:"+test.want {
			t.Fatalf("canonicalize %s: %q, %v; want %s", test.input, got, ok, test.want)
		}
	}
	for _, input := range []string{`"\x0"`, `"\xg0"`, `"\uD800"`, `"\U00110000"`, `"\400"`, `"\q"`, "\"unterminated"} {
		src := []byte(input)
		if _, ok := AppendQuotedStringLiteral(nil, src, MakeToken(TokenString, 0, len(src), 1)); ok {
			t.Fatalf("accepted %s", input)
		}
	}
}
