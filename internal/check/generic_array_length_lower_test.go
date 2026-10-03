package check

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"renvo.dev/internal/load"
	"strings"
	"testing"
)

func TestGenericArrayLengthsPreserveGoChecking(t *testing.T) {
	for _, tc := range []struct{ name, source, want string }{
		{"global", `const N=2;type A=struct{Value [N]int};func main(){_=Id(A{})}`, `[0x2]int`},
		{"typed_constant", `const N uint=2;type A=[N]int;func main(){_=Id(A{})}`, `[0x2]int`},
		{"floating_constant", `const N=2.0;type A=[N]int;func main(){_=Id(A{})}`, `[0x2]int`},
		{"local_iota", `func main(){const(A=iota;N);type Local=[N]int;_=Id(Local{})}`, `[0x1]int`},
		{"local_shadow", `const N=2;type A=[N]int;func main(){const N=3;type B=[N]int;_,_=Id(A{}),Id(B{})}`, `[0x3]int`},
		{"variable_use", `const N=2;func main(){var a [N]int;type Local=[len(a)]int;_=Id(Local{})}`, `len(a)`},
		{"generic_body", `const N=2;func Array[T any](v T)[N]T{type Local=[N]T;return Local{v,v}};func main(){_=Array(42)}`, `type Local=[0x2]T`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte("package main;func Id[T any](v T)T{return v};" + tc.source)
			genericArrayLengthCheckGo(t, source)
			graph := genericTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: source}})
			prepared := PrepareGenerics(graph)
			if !prepared.Ok {
				t.Fatalf("prepare: %s", prepared.Message)
			}
			result := CheckGraphCore(prepared.Graph)
			if !result.Ok {
				t.Fatalf("check: %+v", result)
			}
			text := prepared.Graph.Packages[0].Files[0].Src
			if !strings.Contains(string(text), tc.want) {
				t.Fatalf("missing %s: %s", tc.want, text)
			}
			genericArrayLengthCheckGo(t, text)
		})
	}
}

func TestGenericArrayLengthTypeParameterHidesConstant(t *testing.T) {
	source := []byte("package main;const N=2;type Bad[N any] [N]int;func main(){}")
	set := token.NewFileSet()
	file, err := parser.ParseFile(set, "bad.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	config := types.Config{}
	if _, err = config.Check("example.com/case", set, []*ast.File{file}, nil); err == nil {
		t.Fatal("Go accepted type parameter as a length")
	}
	graph := genericTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: source}})
	if result := PrepareGenerics(graph); result.Ok {
		t.Fatal("Renvo accepted type parameter as a length")
	}
}

func genericArrayLengthCheckGo(t *testing.T, source []byte) {
	t.Helper()
	set := token.NewFileSet()
	file, err := parser.ParseFile(set, "case.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	config := types.Config{}
	if _, err = config.Check("example.com/case", set, []*ast.File{file}, nil); err != nil {
		t.Fatalf("Go rejected source: %v\n%s", err, source)
	}
}
