package parse

import "testing"

func TestFileSourceParsesImportsAndDecls(t *testing.T) {
	file, err := FileSource("main.go", []byte(`package main

import (
	"example.com/app/pkg/a"
	b "example.com/app/pkg/b"
)

const answer = 42
type box struct { value int }
func appMain() int { return answer }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if file.PackageName != "main" {
		t.Fatalf("package = %q", file.PackageName)
	}
	if len(file.Imports) != 2 {
		t.Fatalf("imports = %#v, want 2", file.Imports)
	}
	if file.Imports[1].Alias != "b" {
		t.Fatalf("second import alias = %q, want b", file.Imports[1].Alias)
	}
	if len(file.Decls) != 3 {
		t.Fatalf("decls = %#v, want 3", file.Decls)
	}
	if file.Decls[2].Kind != "func" || file.Decls[2].Name != "appMain" {
		t.Fatalf("func decl = %#v", file.Decls[2])
	}
	if got := DeclText(file, file.Decls[2]); got != "func appMain() int { return answer }" {
		t.Fatalf("decl text = %q", got)
	}
}

func TestFileSourceRejectsUnexpectedTopLevelToken(t *testing.T) {
	_, err := FileSource("bad.go", []byte("package main\nx := 1\n"))
	if err == nil {
		t.Fatalf("FileSource succeeded for non-declaration top-level statement")
	}
}

func TestFileSourceMarksMethodDecls(t *testing.T) {
	file, err := FileSource("method.go", []byte(`package main

type box struct { value int }
func (b box) Value() int { return b.value }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if len(file.Decls) != 2 {
		t.Fatalf("decls = %#v, want 2", file.Decls)
	}
	method := file.Decls[1]
	if method.Name != "Value" || !method.Receiver {
		t.Fatalf("method decl = %#v, want receiver Value", method)
	}
}
