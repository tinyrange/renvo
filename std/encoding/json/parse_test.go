package json

import "testing"

func TestParseJSONKeepsNumbersAndMembers(t *testing.T) {
	source := []byte(" {\"n\":18446744073709551615,\"n\":-0.50e+02,\"s\":\"\\ud834\\udd1e\",\"a\":[null,true,false]} trailing")
	value, at, err := parseJSON(source)
	if err != nil || value.kind != 'o' || len(value.keys) != 4 || value.keys[0] != "n" || value.keys[1] != "n" {
		t.Fatal("object parse")
	}
	if value.items[0].text != "18446744073709551615" || value.items[1].text != "-0.50e+02" {
		t.Fatal("number spelling lost")
	}
	if value.items[2].text != "𝄞" || string(source[at:]) != " trailing" {
		t.Fatal("string or cursor")
	}
	if len(value.items[3].items) != 3 || value.items[3].items[0].kind != '0' {
		t.Fatal("array")
	}
}

func TestJSONSyntaxAndUnicode(t *testing.T) {
	for _, source := range []string{`null`, ` true `, `-0`, `1.2e-3`, `{}`, `[]`, `{"a":[1,"x"]}`, `"\ud800"`, `"\udc00"`, `"\ud800\u0061"`, "\"\xff\""} {
		if !Valid([]byte(source)) {
			t.Fatal("rejected valid JSON", source)
		}
	}
	for _, source := range []string{"", " ", "01", "-01", "-", "+1", "1.", "1e", "1e+", "NaN", "[1,]", "{\"a\":}", "{\"a\":1,}", "{a:1}", "true false", "nullx", "\"\n\"", "\"\\x00\"", "\"\\u123\"", "\"\\ud800\\uZZZZ\"", "/*x*/1"} {
		if Valid([]byte(source)) {
			t.Fatal("accepted invalid JSON", source)
		}
	}
	for _, test := range []struct{ source, want string }{
		{`"\ud800\u0061"`, "�a"}, {`"\udc00"`, "�"}, {`"\ud834\udd1e"`, "𝄞"}, {"\"\xff\"", "�"},
	} {
		value, _, err := parseJSON([]byte(test.source))
		if err != nil || value.text != test.want {
			t.Fatal("Unicode decoding", test.source)
		}
	}
}

func TestJSONSyntaxErrorOffset(t *testing.T) {
	_, _, err := parseJSON([]byte("[1,]"))
	syntax, ok := err.(*SyntaxError)
	if !ok || syntax.Offset != 4 {
		t.Fatal("syntax error offset")
	}
	_, _, err = parseJSON([]byte("[1"))
	syntax, ok = err.(*SyntaxError)
	if !ok || syntax.Offset != 2 {
		t.Fatal("EOF offset")
	}
}
