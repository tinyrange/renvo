package syntax

import (
	"strings"
	"testing"
)

func TestLinkedParsingPreservesWideLines(t *testing.T) {
	source := "package p\n" + strings.Repeat("\n", TokenLineLimit) + "var X = 1\nvar Y = 2\nfunc f() {\n x := 1\n x++\n}\n"
	ordinary := ParseFile([]byte(source))
	if ordinary.Ok || ordinary.Error != ParseErrScan {
		t.Fatal("source line admission changed")
	}
	linked := ParseLinkedFile([]byte(source))
	if !linked.Ok || len(linked.Decls) != 2 || len(linked.Funcs) != 1 {
		t.Fatalf("linked parse: %+v", linked.Error)
	}
	if line := TokenLineAt(&linked, linked.Decls[0].NameTok); line != TokenLineLimit+2 {
		t.Fatalf("lost physical line: %d", line)
	}
	if line := TokenLineAt(&linked, linked.Decls[1].NameTok); line != TokenLineLimit+3 {
		t.Fatalf("lost declaration boundary: %d", line)
	}
}

func TestLinkedParsingRetainsEncodingAndEscapeValidation(t *testing.T) {
	for _, source := range []string{
		"package p\nvar X = \"\\q\"\n",
		"package p\n// bad \xff\n",
		"package p\nvar X = \"unterminated\n",
	} {
		if ParseLinkedFile([]byte(source)).Ok {
			t.Fatal("invalid linked source accepted")
		}
	}
}
