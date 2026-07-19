package syntax

import "testing"

func TestParseFileTopLevel(t *testing.T) {
	src := []byte(`package main

import (
	"fmt"
	lib "example/lib"
	. "example/dot"
	_ "example/blank"
)

const answer = 42
var left, right int
type point struct { x int; y int }

func add(a int, b int) int { return a + b }
func (p point) Len() int { return p.x + p.y }
`)
	file := ParseFile(src)
	if !file.Ok {
		t.Fatalf("ParseFile failed: err=%d tok=%d", file.Error, file.ErrorTok)
	}
	if tokenString(file, file.PackageName) != "main" {
		t.Fatalf("package = %q, want main", tokenString(file, file.PackageName))
	}
	if len(file.Imports) != 4 {
		t.Fatalf("import count = %d, want 4", len(file.Imports))
	}
	if file.Imports[0].NameTok != -1 || tokenString(file, file.Imports[0].PathTok) != `"fmt"` {
		t.Fatalf("first import = %#v", file.Imports[0])
	}
	if tokenString(file, file.Imports[1].NameTok) != "lib" || tokenString(file, file.Imports[1].PathTok) != `"example/lib"` {
		t.Fatalf("second import = %#v", file.Imports[1])
	}
	if tokenString(file, file.Imports[2].NameTok) != "." || tokenString(file, file.Imports[3].NameTok) != "_" {
		t.Fatalf("dot/blank imports = %#v %#v", file.Imports[2], file.Imports[3])
	}
	if len(file.Decls) != 4 {
		t.Fatalf("decl count = %d, want 4: %#v", len(file.Decls), file.Decls)
	}
	wantDecls := []string{"answer", "left", "right", "point"}
	for i := 0; i < len(wantDecls); i++ {
		if tokenString(file, file.Decls[i].NameTok) != wantDecls[i] {
			t.Fatalf("decl %d name = %q, want %q", i, tokenString(file, file.Decls[i].NameTok), wantDecls[i])
		}
	}
	if len(file.Funcs) != 2 {
		t.Fatalf("func count = %d, want 2", len(file.Funcs))
	}
	if tokenString(file, file.Funcs[0].NameTok) != "add" {
		t.Fatalf("first function = %q, want add", tokenString(file, file.Funcs[0].NameTok))
	}
	if tokenString(file, file.Funcs[1].NameTok) != "Len" {
		t.Fatalf("second function = %q, want Len", tokenString(file, file.Funcs[1].NameTok))
	}
	if file.Funcs[1].ReceiverStart < 0 || file.Funcs[1].ReceiverEnd <= file.Funcs[1].ReceiverStart {
		t.Fatalf("method receiver range = %d:%d", file.Funcs[1].ReceiverStart, file.Funcs[1].ReceiverEnd)
	}
}

func TestParseGroupedDeclarations(t *testing.T) {
	src := []byte(`package main

const (
	A = 1
	B
)

var (
	x, y int
	z = []int{
		1,
	}
)

type (
	item struct { value int }
	alias = item
)
`)
	file := ParseFile(src)
	if !file.Ok {
		t.Fatalf("ParseFile failed: err=%d tok=%d", file.Error, file.ErrorTok)
	}
	want := []string{"A", "B", "x", "y", "z", "item", "alias"}
	if len(file.Decls) != len(want) {
		t.Fatalf("decl count = %d, want %d: %#v", len(file.Decls), len(want), file.Decls)
	}
	for i := 0; i < len(want); i++ {
		if tokenString(file, file.Decls[i].NameTok) != want[i] {
			t.Fatalf("decl %d name = %q, want %q", i, tokenString(file, file.Decls[i].NameTok), want[i])
		}
	}
}

func TestParseFunctionResultShapes(t *testing.T) {
	src := []byte(`package main

func one() struct { value int } { return struct { value int }{value: 1} }
func two() interface { Value() int } { return nil }
func three() map[string]struct { value int } { return nil }
`)
	file := ParseFile(src)
	if !file.Ok {
		t.Fatalf("ParseFile failed: err=%d tok=%d", file.Error, file.ErrorTok)
	}
	if len(file.Funcs) != 3 {
		t.Fatalf("func count = %d, want 3", len(file.Funcs))
	}
	for i := 0; i < len(file.Funcs); i++ {
		fn := file.Funcs[i]
		if fn.BodyStart < 0 || fn.BodyEnd <= fn.BodyStart {
			t.Fatalf("function %d body range = %d:%d", i, fn.BodyStart, fn.BodyEnd)
		}
		if tokenString(file, fn.BodyStart) != "{" {
			t.Fatalf("function %d body start = %q, want {", i, tokenString(file, fn.BodyStart))
		}
	}
}

func TestParseErrors(t *testing.T) {
	cases := []struct {
		name string
		src  string
		err  int
	}{
		{name: "missing package", src: "func main() {}\n", err: ParseErrPackage},
		{name: "bad import", src: "package main\nimport bad\n", err: ParseErrImport},
		{name: "bad decl", src: "package main\nvar = 1\n", err: ParseErrDecl},
		{name: "bad func", src: "package main\nfunc f()\n", err: ParseErrFunc},
		{name: "bad scan", src: "package main\nvar s = \"unterminated\n", err: ParseErrScan},
	}
	for _, tc := range cases {
		file := ParseFile([]byte(tc.src))
		if file.Ok {
			t.Fatalf("%s: ParseFile succeeded", tc.name)
		}
		if file.Error != tc.err {
			t.Fatalf("%s: error = %d, want %d", tc.name, file.Error, tc.err)
		}
	}
}

func tokenString(file File, tok int) string {
	if tok < 0 || tok >= len(file.Tokens) {
		return ""
	}
	return string(TokenText(file.Src, file.Tokens[tok]))
}
