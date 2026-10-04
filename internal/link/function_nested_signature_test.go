package link

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func TestNestedFunctionSignatureIdentityMatchesGo(t *testing.T) {
	for _, test := range []struct{ left, right string }{
		{"func(value int)int", "func(int)int"},
		{"func(value int)(result int)", "func(int)int"},
		{"func(a,b int)int", "func(int,int)int"},
		{"func(a,b,c int)int", "func(int,int,int)int"},
		{"func(F,b,c int)int", "func(int,int,int)int"},
		{"func()(F,b,c int)", "func()(int,int,int)"},
		{"func(func(value int)int)int", "func(func(int)int)int"},
		{"func()func(value int)int", "func()(func(int)int)"},
		{"[]func(value int)int", "[]func(int)int"},
		{"[2]func(value int)int", "[2]func(int)int"},
		{"*func(value int)int", "*func(int)int"},
		{"map[string]func(value int)int", "map[string]func(int)int"},
		{"chan func(value int)int", "chan func(int)int"},
		{"func(chan int)int", "func(chanint)int"},
		{"func(...int)int", "func([]int)int"},
		{"func(int)int", "func(int)string"},
		{"func(F)int", "func(G)int"},
		{"*func(int)int", "func(int)int"},
	} {
		t.Run(test.left+"/"+test.right, func(t *testing.T) {
			set := token.NewFileSet()
			source := "package p;type chanint int;type F func(int)int;type G func(int)int;type Left=" + test.left + ";type Right=" + test.right
			file, err := parser.ParseFile(set, "case.go", source, 0)
			if err != nil {
				t.Fatal(err)
			}
			config := types.Config{}
			pkg, err := config.Check("example.com/identity", set, []*ast.File{file}, nil)
			if err != nil {
				t.Fatal(err)
			}
			want := types.Identical(pkg.Scope().Lookup("Left").Type(), pkg.Scope().Lookup("Right").Type())
			if got := functionValueSameNestedType(test.left, test.right); got != want {
				t.Fatalf("identity=%v, Go=%v", got, want)
			}
		})
	}
}
