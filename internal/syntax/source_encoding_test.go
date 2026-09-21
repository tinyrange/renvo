package syntax

import (
	"strings"
	"testing"
)

func TestAuditSourceEncodingAndLiteralLimit(t *testing.T) {
	for _, source := range []string{
		"package main\n// \xff\n",
		"package main\nvar s = `\xed\xa0\x80`",
		"package main\nvar s = \"\xc0\x80\"",
		"package main\x00",
		"package main\nvar 😀 = 1",
		"package main\nvar ٢x = 1",
		"package main\nvar x\u0301 = 1",
		"package main\nvar x = " + strings.Repeat("9", 20000),
	} {
		if file := ParseFile([]byte(source)); file.Ok {
			t.Fatal("accepted invalid source")
		}
	}
	for _, source := range []string{
		"package main\n// 日本語\nvar s = `é`",
		"\xef\xbb\xbfpackage main\nvar s = \"\\xff\"",
		"package main\nvar x = .5 + +.01",
	} {
		if file := ParseFile([]byte(source)); !file.Ok {
			t.Fatalf("rejected valid source: %+v", file)
		}
	}
}
