package check

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"renvo.dev/internal/load"
)

func TestGenericRangeFunctionSignatures(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		valid        bool
	}{
		{"literal_iterator", `func Walk[T ~int](){for v:=range func(yield func(T)bool){yield(42)}{_=v}};func main(){Walk[int]()}`, true},
		{"ordinary_literal_iterator", `func main(){for v:=range func(yield func(int)bool){yield(42)}{_=v}}`, true},
		{"blank_declaration", `func Walk(seq func(func(int)bool)){for _:=range seq{}};func main(){}`, false},
		{"duplicate_declaration", `func Walk(seq func(func(int,int)bool)){for x,x:=range seq{_=x}};func main(){}`, false},
		{"indexed_declaration", `func Walk(seq func(func(int)bool)){var a [1]int;for a[0]:=range seq{}};func main(){}`, false},
		{"named_yield", `type Yield[V any]func(V)bool;type Sequence[V any]func(Yield[V]);func Values[V any](v V)Sequence[V]{return func(yield Yield[V]){yield(v)}};func main(){for v:=range Values(42){_=v}}`, true},
		{"parameter_yield", `func Walk[Y ~func(int)bool](seq func(Y)){for v:=range seq{_=v}};func main(){}`, true},
		{"variadic_yield", `func Walk(seq func(func(...int)bool)){for values:=range seq{_=len(values)}};func main(){}`, true},
		{"named_variadic_yield", `type Yield[V any]func(...V)bool;func Walk[V any](seq func(Yield[V])){for values:=range seq{_=len(values)}};func main(){}`, true},
		{"zero_yield", `func Walk(seq func(func()bool)){for range seq{}};func main(){}`, true},
		{"two_yield", `func Walk[K,V any](seq func(func(K,V)bool)){for k,v:=range seq{_,_=k,v}};func main(){}`, true},
		{"boolean_alias", `type Boolean=bool;func Walk(seq func(func(int)Boolean)){for v:=range seq{_=v}};func main(){}`, true},
		{"named_boolean", `type Boolean bool;func Walk(seq func(func(int)Boolean)){for v:=range seq{_=v}};func main(){}`, false},
		{"yield_without_result", `func Walk(seq func(func(int))){for v:=range seq{_=v}};func main(){}`, false},
		{"yield_with_two_results", `func Walk(seq func(func(int)(bool,bool))){for v:=range seq{_=v}};func main(){}`, false},
		{"three_yield", `func Walk(seq func(func(int,int,int)bool)){for v:=range seq{_=v}};func main(){}`, false},
		{"iterator_result", `func Walk(seq func(func(int)bool)int){for v:=range seq{_=v}};func main(){}`, false},
		{"pointer_yield", `func Walk(seq func(*func(int)bool)){for v:=range seq{_=v}};func main(){}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte("package main;func Identity[T any](v T)T{return v};" + tc.source + "\n")
			set := token.NewFileSet()
			parsed, err := parser.ParseFile(set, "main.go", source, 0)
			if err != nil {
				t.Fatal(err)
			}
			config := types.Config{GoVersion: "go1.25"}
			_, err = config.Check("example.com/case", set, []*ast.File{parsed}, nil)
			if (err == nil) != tc.valid {
				t.Fatalf("Go validity: %v", err)
			}
			graph := genericTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: source}})
			prepared := PrepareGenerics(graph)
			if prepared.Ok != tc.valid {
				t.Fatalf("ok=%v want=%v: %s", prepared.Ok, tc.valid, prepared.Message)
			}
		})
	}
}

func TestGenericRangeLanguageVersions(t *testing.T) {
	for _, tc := range []struct {
		version, body string
		valid         bool
	}{
		{"1.21", `func Walk[T ~int](n T){for v:=range n{_=v}}`, false},
		{"1.22", `func Walk[T ~int](n T){for v:=range n{_=v}}`, true},
		{"1.21", `func Walk(){for v:=range 3{_=v}}`, false},
		{"1.22", `func Walk(){for v:=range 3{_=v}}`, true},
		{"1.22", `func Walk[T any](seq func(func(T)bool)){for v:=range seq{_=v}}`, false},
		{"1.23", `func Walk[T any](seq func(func(T)bool)){for v:=range seq{_=v}}`, true},
		{"1.22", `func Walk(seq func(func(int)bool)){for v:=range seq{_=v}}`, false},
		{"1.23", `func Walk(seq func(func(int)bool)){for v:=range seq{_=v}}`, true},
		{"1.21", `func Walk(){for v:=range "value"{_=v}}`, true},
	} {
		t.Run(tc.version+tc.body, func(t *testing.T) {
			source := []byte("package main;func Identity[T any](v T)T{return v};" + tc.body + ";func main(){}\n")
			set := token.NewFileSet()
			parsed, err := parser.ParseFile(set, "main.go", source, 0)
			if err != nil {
				t.Fatal(err)
			}
			config := types.Config{GoVersion: "go" + tc.version}
			_, err = config.Check("example.com/case", set, []*ast.File{parsed}, nil)
			if (err == nil) != tc.valid {
				t.Fatalf("Go validity: %v", err)
			}
			graph := genericVersionTestGraph(t, tc.version, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: source}})
			if prepared := PrepareGenerics(graph); prepared.Ok != tc.valid {
				t.Fatalf("ok=%v want=%v: %s", prepared.Ok, tc.valid, prepared.Message)
			}
		})
	}
}
