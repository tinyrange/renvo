package parse

import (
	"strings"
	"testing"
)

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

func TestFileSourceRejectsMalformedImportBlockEntry(t *testing.T) {
	_, err := FileSource("bad.go", []byte(`package main

import (
	fmt
)

func appMain() int { return 0 }
`))
	if err == nil {
		t.Fatalf("FileSource accepted malformed import block entry")
	}
	if !strings.Contains(err.Error(), "4:2: malformed import declaration") {
		t.Fatalf("error = %q", err)
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

func TestFileSourceRecordsGroupedDeclNames(t *testing.T) {
	file, err := FileSource("group.go", []byte(`package pkg

const (
	A = 1
	B, C = 2, 3
)

type (
	Box struct {
		value int
	}
	Alias int
)
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if len(file.Decls) != 2 {
		t.Fatalf("decls = %#v, want 2", file.Decls)
	}
	wantConst := []string{"A", "B", "C"}
	if !sameStrings(file.Decls[0].Names, wantConst) {
		t.Fatalf("const names = %#v, want %#v", file.Decls[0].Names, wantConst)
	}
	if len(file.Decls[0].NameToks) != 3 || file.Decls[0].NameToks[1].Line != 5 || file.Decls[0].NameToks[1].Column != 2 {
		t.Fatalf("const name tokens = %#v", file.Decls[0].NameToks)
	}
	wantType := []string{"Box", "Alias"}
	if !sameStrings(file.Decls[1].Names, wantType) {
		t.Fatalf("type names = %#v, want %#v", file.Decls[1].Names, wantType)
	}
}

func TestFileSourceRecordsSingleDeclNameLists(t *testing.T) {
	file, err := FileSource("names.go", []byte(`package pkg

const A, B = 1, 2
var C, D int
var E = A
type Box struct { value int }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if len(file.Decls) != 4 {
		t.Fatalf("decls = %#v, want 4", file.Decls)
	}
	if !sameStrings(file.Decls[0].Names, []string{"A", "B"}) {
		t.Fatalf("const names = %#v", file.Decls[0].Names)
	}
	if !sameStrings(file.Decls[1].Names, []string{"C", "D"}) {
		t.Fatalf("var names = %#v", file.Decls[1].Names)
	}
	if !sameStrings(file.Decls[2].Names, []string{"E"}) {
		t.Fatalf("single var names = %#v", file.Decls[2].Names)
	}
	if !sameStrings(file.Decls[3].Names, []string{"Box"}) {
		t.Fatalf("type names = %#v", file.Decls[3].Names)
	}
}

func TestFileSourceMarksTopLevelFuncsAfterCompositeLiteralReturn(t *testing.T) {
	file, err := FileSource("funcs.go", []byte(`package pkg

type Package struct{}

func packageByImportPath() (Package, bool) {
	return Package{}, false
}

func nextFunc() int {
	return 1
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if len(file.Decls) != 3 {
		t.Fatalf("decls = %#v, want type plus two funcs", file.Decls)
	}
	nextFuncToken := -1
	for i := 0; i+1 < len(file.Tokens); i++ {
		if file.Tokens[i].Text == "func" && file.Tokens[i+1].Text == "nextFunc" {
			nextFuncToken = i
			break
		}
	}
	if nextFuncToken < 0 {
		t.Fatalf("did not find nextFunc token")
	}
	if !file.IsTopLevelFuncAt(nextFuncToken) {
		t.Fatalf("nextFunc token was not marked as top-level")
	}
	file.Decls = file.Decls[:1]
	if !file.IsTopLevelFuncAt(nextFuncToken) {
		t.Fatalf("column-1 nextFunc token was not recognized as top-level without decl bookkeeping")
	}
}

func sameStrings(got []string, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
