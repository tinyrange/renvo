package check

import (
	"renvo.dev/internal/load"
	"strings"
	"testing"
)

func TestGenericPackageValuesShadowPredeclaredNames(t *testing.T) {
	for _, name := range []string{"true", "false", "nil", "iota"} {
		for _, declaration := range []string{"const " + name + " = 7", "var " + name + " = 7", "func " + name + "()int{return 7}"} {
			expression := name
			if declaration[:4] == "func" {
				expression += "()"
			}
			graph := genericTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main;" + declaration + ";func F[T ~int]()T{return T(" + expression + ")};func main(){_=F[int]()}")}})
			if got := PrepareGenerics(graph); !got.Ok {
				t.Errorf("%s: %s", declaration, got.Message)
			}
		}
	}
}

func TestGenericFoldedConstantsPreserveShadowedTypes(t *testing.T) {
	graph := genericTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main
func F[T any](){int:=int(42);if int!=42{panic(1)}}
func main(){F[string]()}
`)}})
	prepared := PrepareGenerics(graph)
	if !prepared.Ok {
		t.Fatal(prepared.Message)
	}
	text := string(prepared.Graph.Packages[checkRootPackage(t, graph)].Files[0].Src)
	if strings.Contains(text, "int != int(") || !strings.Contains(text, "type RenvoGenericType_") {
		t.Fatalf("folded constant refers to a shadowed type:\n%s", text)
	}
}

func TestGenericPredeclaredTypeShadowing(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		valid        bool
	}{
		{"variable_type", `var int=0;func F[T any](v int){};func main(){}`, false},
		{"constant_type", `const int=0;func F[T any](v int){};func main(){}`, false},
		{"variable_any_constraint", `var any=0;func F[T any](){};func main(){}`, false},
		{"variable_comparable_constraint", `var comparable=0;func F[T comparable](){};func main(){}`, false},
		{"local_type_argument", `func F[T any](){};func main(){int:=0;F[int]();_=int}`, false},
		{"local_container_argument", `func F[T any](){};func main(){int:=0;F[[]int]();_=int}`, false},
		{"closure_parameter_type", `func F[T any](){};func main(){_=func(int string){F[[]int]()}}`, false},
		{"global_closure_parameter_type", `func F[T any](){};var f=func(int string){F[[]int]()};func main(){}`, false},
		{"local_type_parameter", `func F[T any](v T){T:=0;var x T;_=x;_=T;_=v};func main(){}`, false},
		{"parameter_comparable_constraint", `func F[comparable any,T comparable](){};func main(){}`, false},
		{"function_variable_call", `var int=func(v string)string{return v+"!"};func F[T ~string](v T)T{return T(int(string(v)))};func main(){_=F[string]("a")}`, true},
		{"builtin_variable_call", `var len=func(v string)string{return v+"!"};func F[T any](v string)string{return len(v)};func main(){_=F[int]("a")}`, true},
		{"parameter_shadows_package", `var any=0;func F[any interface{}](v any)any{return v};func main(){_=F[int](1)}`, true},
		{"type_shadows_value", `func F[T any](v T){int:=0;_=int;{type int=string;var s int="a";_=s};_=v};func main(){_=F[int]}`, true},
		{"initializer_visibility", `func F[T any](){};func main(){int:=int(1);F[[]string]();_=int}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			graph := genericTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main;" + tc.source)}})
			got := PrepareGenerics(graph)
			if got.Ok != tc.valid {
				t.Fatalf("ok=%v want=%v: %s", got.Ok, tc.valid, got.Message)
			}
		})
	}
}

func TestGenericPredeclaredTypeNamesShadowedByPackageValues(t *testing.T) {
	for _, name := range []string{"bool", "string", "int", "uint", "uintptr", "byte", "rune", "int8", "int16", "int32", "int64", "uint8", "uint16", "uint32", "uint64", "float32", "float64", "complex64", "complex128", "any", "error", "comparable"} {
		t.Run(name, func(t *testing.T) {
			graph := genericTestGraph(t, []load.SourceFile{
				{Path: "/repo/case/cmd/app/value.go", Src: []byte("package main;var " + name + "=0")},
				{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main;func F[T interface{}](v " + name + "){};func main(){}")},
			})
			if got := PrepareGenerics(graph); got.Ok {
				t.Fatal("package value used as a type")
			}
		})
	}
}
