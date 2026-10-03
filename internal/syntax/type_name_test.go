package syntax

import "testing"

func TestGeneratedTypeNamesStayAdjacentAndDecode(t *testing.T) {
	for _, test := range []struct {
		source, typeName, fieldName string
	}{
		{"//renvo:typename \"Box[int]\" \"Box\"\n", "Box[int]", "Box"},
		{"//renvo:typename \"Box[int]\" \"Box\"\n//renvo:reflect\n", "Box[int]", "Box"},
		{"  //renvo:typename \"Box[\\xCE\\x94]\" \"\\xCE\\x94\"\r\n  //renvo:reflect\r\n", "Box[Δ]", "Δ"},
		{"//renvo:typename \"Box[int]\" \"Box\"\n\n", "", ""},
		{"//renvo:typename \"Box[int]\"\n", "", ""},
		{"//renvo:typename \"Box[int]\" \"Box\" extra\n", "", ""},
		{"//renvo:typename \"\\q\" \"Box\"\n", "", ""},
		{"//renvo:typename \"\" \"Box\"\n", "", ""},
		{"", "", ""},
	} {
		source := []byte("package p\n" + test.source + "type Renamed struct{}\n")
		file := ParseFile(source)
		if !file.Ok || len(file.Decls) != 1 {
			t.Fatalf("invalid fixture: %s", source)
		}
		token := file.Decls[0].StartTok
		if token > 0 && file.Tokens[token-1].KindLine&255 == TokenType {
			token--
		}
		typeName, fieldName := GeneratedTypeNames(source, int(file.Tokens[token].Start))
		if typeName != test.typeName || fieldName != test.fieldName {
			t.Errorf("names=(%q,%q) want=(%q,%q): %s", typeName, fieldName, test.typeName, test.fieldName, test.source)
		}
	}
}
