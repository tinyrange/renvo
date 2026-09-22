package syntax

import "testing"

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
