package check

import (
	"testing"

	"renvo.dev/internal/syntax"
)

func TestBareReturnShadow(t *testing.T) {
	for _, body := range []string{
		"if true { x := 1; return }; return",
		"if true { var x int; return }; return",
		"if true { const x = 1; return }; return",
		"if true { type x int; return }; return",
		"if true { var (x int); return }; return",
		"if x := 1; x > 0 { return }; return",
		"if x := 1; x > 0 {} else { return }; return",
		"for x := 0; x < 1; x++ { return }; return",
		"for x := range []int{1} { return }; return",
		"switch x := 1; x { case 1: return }; return",
		"switch 1 { case 1: x := 2; return }; return",
		"select { case x := <-make(chan int): return }; return",
	} {
		t.Run(body, func(t *testing.T) {
			file := syntax.ParseFile([]byte("package main\nfunc f() (x int) { " + body + " }\n"))
			fn := file.Funcs[0]
			parsed := syntax.ParseFuncBody(file, fn)
			if !file.Ok || !parsed.Ok {
				t.Fatal("parse failed")
			}
			if tok := invalidBareReturnShadow(file, fn, parsed, buildFuncSignature(file, fn)); tok < 0 {
				t.Fatal("shadowed bare return accepted")
			}
		})
	}
}

func TestBareReturnPreservesLexicalScopes(t *testing.T) {
	for _, body := range []string{
		"return",
		"x, y := 1, 2; _ = y; return",
		"if true { x := 1; _ = x }; return",
		"if true { return; x := 1; _ = x }; return",
		"if true { x := 1; _ = x } else { return }; return",
		"if x := 1; x > 0 {} else { _ = x }; return",
		"switch 1 { case 1: x := 2; _ = x; case 2: return }; return",
		"for x := 0; x < 1; x++ {}; return",
		"if true { x := 1; return x }; return",
		"_ = func() int { x := 1; return x }; return",
		"_ = func() (x int) { return }; return",
	} {
		t.Run(body, func(t *testing.T) {
			file := syntax.ParseFile([]byte("package main\nfunc f() (x int) { " + body + " }\n"))
			fn := file.Funcs[0]
			parsed := syntax.ParseFuncBody(file, fn)
			if !file.Ok || !parsed.Ok {
				t.Fatal("parse failed")
			}
			if tok := invalidBareReturnShadow(file, fn, parsed, buildFuncSignature(file, fn)); tok >= 0 {
				t.Fatalf("valid return rejected at token %d", tok)
			}
		})
	}
}
