package check

import (
	"testing"

	"renvo.dev/internal/syntax"
)

func TestGotoScopeDependsOnSourceLanguage(t *testing.T) {
	for _, tc := range []struct {
		body    string
		goError int
		cError  int
	}{
		{"goto L; { L: return }", CheckErrScope, CheckOK},
		{"goto L; var x int; L: _ = x", CheckErrScope, CheckOK},
		{"goto missing", CheckErrScope, CheckErrScope},
	} {
		file := syntax.ParseFile([]byte("package main\nfunc main() { " + tc.body + " }"))
		if !file.Ok || len(file.Funcs) != 1 {
			t.Fatal("failed to parse branch fixture")
		}
		body := syntax.ParseFuncBodyStatements(file, file.Funcs[0])
		if !body.Ok {
			t.Fatal("failed to parse branch body")
		}
		for _, cSource := range []bool{false, true} {
			want := tc.goError
			if cSource {
				want = tc.cError
			}
			got, tok := invalidDefiniteStatement(file, body, cSource)
			if got != want {
				t.Fatalf("%s (C=%v): error=%d token=%d, want %d", tc.body, cSource, got, tok, want)
			}
		}
	}
}
