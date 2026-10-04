package check

import (
	"go/types"
	"renvo.dev/internal/load"
	"testing"
)

func TestGenericConcreteAliasesPreserveLexicalScope(t *testing.T) {
	for _, tc := range []struct{ name, source string }{
		{"parameter", `func Id[T any](int T)T{return int}
func main(){_=Id(42)}`},
		{"grouped_parameters", `func Pair[T any](int,string T)(T,T){return int,string}
func main(){_,_=Pair(42,43)}`},
		{"result", `func Id[T any](v T)(int T){return v}
func main(){_=Id(42)}`},
		{"receiver", `type Box[T any]struct{Value T}
func(int Box[T]) Get()T{return int.Value}
func main(){_=Box[int]{42}.Get()}`},
		{"named_type", `type Value int
func Id[T any](Value T)T{return Value}
func main(){_=Id(Value(42))}`},
		{"compound", `func Id[T any](int T)T{return int}
func main(){_=Id(map[string][]int{"key":{42}})}`},
		{"parameter_type_name", `func Id[int any](v int)int{return v}
func main(){_=Id(42)}`},
		{"closure_captured_parameter", `func Make[T any](int T)func(T)T{return func(v T)T{return v}}
func main(){_=Make(42)(43)}`},
		{"closure_captured_local", `func Make[T any](v T)func()T{int:=v;return func()T{return int}}
func main(){_=Make(42)()}`},
		{"closure_local_type", `func Make[T any](v T)func()T{type int string;_=int("");return func()T{return v}}
func main(){_=Make(42)()}`},
		{"closure_parameter", `func Make[T any](v T)func(T)T{return func(int T)T{return int}}
func main(){_=Make(42)(43)}`},
		{"closure_result", `func Make[T any](v T)func()T{return func()(int T){return v}}
func main(){_=Make(42)()}`},
		{"closure_captured_receiver", `type Box[T any]struct{Value T}
func(int Box[T]) Make()func()T{return func()T{return int.Value}}
func main(){_=Box[int]{42}.Make()()}`},
		{"closure_captured_result", `func Make[T any](v T)(int func()T){return func()T{return v}}
func main(){_=Make(42)()}`},
		{"local_alias", `func Id[T any](int T)T{type U=T;var v U=int;return v}
func main(){_=Id(42)}`},
		{"closure_copied_alias", `func Make[T any](int T)func()T{type U=T;return func()U{return int}}
func main(){_=Make(42)()}`},
		{"closure_result_container", `func Make[T any](v T)func()[]T{int:=v;return func()[]T{return []T{int}}}
func main(){_=Make(42)()[0]}`},
		{"map_body", `func Map[T any](int T)map[string]T{return map[string]T{"key":int}}
func main(){_=Map(42)}`},
		{"generic_alias_body", `type Alias[T any]=map[string]T
func Map[T any](int T)Alias[T]{return Alias[T]{"key":int}}
func main(){_=Map(42)}`},
		{"function_conversion", `func Convert[T ~func(int)int](int T)T{return T(int)}
func main(){_=Convert(func(v int)int{return v})(42)}`},
		{"type_parameter_method_cast", `type V int
func(v V)Get()int{return int(v)}
func Read[T interface{Get()int}](int T) func()int{return int.Get}
func main(){_=Read(V(42))()}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			graph := genericTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + tc.source + "\n")}})
			original := genericConcreteImporter{t: t, graph: graph, packages: map[string]*types.Package{}}
			original.Import(graph.Root)
			prepared := PrepareGenerics(graph)
			if !prepared.Ok {
				t.Fatal(prepared.Message)
			}
			concrete := genericConcreteImporter{t: t, graph: prepared.Graph, packages: map[string]*types.Package{}}
			concrete.Import(graph.Root)
			if checked := CheckGraphCore(prepared.Graph); !checked.Ok {
				t.Fatalf("concrete check error=%d pkg=%d token=%d", checked.Error, checked.ErrorPackage, checked.ErrorToken)
			}
		})
	}
}
