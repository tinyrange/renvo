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
		"package main\nvar π, 世界, α٢, 𐐀 = 1, 2, 3, 4",
	} {
		if file := ParseFile([]byte(source)); !file.Ok {
			t.Fatalf("rejected valid source: %+v", file)
		}
	}
}

func TestAuditGotoGrammar(t *testing.T) {
	for _, source := range []string{
		"goto if ) + chan range interface const ] type nil var func switch interface",
		"goto",
		"goto first second",
	} {
		file := ParseFile([]byte("package main\nfunc main() { " + source + " }"))
		if !file.Ok || len(file.Funcs) != 1 {
			t.Fatal("invalid fixture")
		}
		if ParseFuncBodyStatements(file, file.Funcs[0]).Ok {
			t.Fatalf("accepted %q", source)
		}
	}
	file := ParseFile([]byte("package main\nfunc main() { goto done; done: return }"))
	if !ParseFuncBodyStatements(file, file.Funcs[0]).Ok {
		t.Fatal("rejected valid goto")
	}
}

func TestAuditAssignmentNewline(t *testing.T) {
	file := ParseFile([]byte("package main\nfunc main() { x :=\n1\nx =\n2\nx +=\n3\n_ = x }"))
	body := ParseFuncBodyStatements(file, file.Funcs[0])
	if !body.Ok || len(body.Stmts) != 5 {
		t.Fatalf("assignment split at legal newline: %+v", body)
	}
}
