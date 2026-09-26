package json

import "testing"

//renvo:reflect
type foldRecord struct {
	Name string
	NAME string
	K    string
	Σ    string
	𐐀    string
}

func TestDecoderFieldFolding(t *testing.T) {
	var value foldRecord
	node, _, err := parseJSON([]byte("{\"name\":\"first\",\"NAME\":\"exact\",\"K\":\"kelvin\",\"ς\":\"sigma\",\"𐐨\":\"deseret\"}"))
	if err != nil {
		t.Fatal(err)
	}
	if err := decodeTarget(node, &value, true); err != nil {
		t.Fatal(err)
	}
	if value.Name != "first" || value.NAME != "exact" || value.K != "kelvin" || value.Σ != "sigma" || value.𐐀 != "deseret" {
		t.Fatal("field fold/precedence")
	}
}
