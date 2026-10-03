package check

import (
	"renvo.dev/internal/load"
	"strings"
	"testing"
)

func TestConstraintInterfacesInValueExpressions(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		valid        bool
	}{
		{"conversion", `type C interface{~int};func unused(){_=C(1)};func main(){}`, false},
		{"parenthesized_conversion", `type C interface{~int};func unused(){_=(C)(1)};func main(){}`, false},
		{"anonymous_conversion", `func unused(){_=interface{~int}(1)};func main(){}`, false},
		{"alias_conversion", `type C interface{~int};type A=C;func unused(){_=A(1)};func main(){}`, false},
		{"local_alias_conversion", `type C interface{~int};func unused(){type A=C;_=A(1)};func main(){}`, false},
		{"pointer_conversion", `type C interface{~int};func unused(){_=(*C)(nil)};func main(){}`, false},
		{"slice_conversion", `type C interface{~int};func unused(){_=[]C(nil)};func main(){}`, false},
		{"map_conversion", `type C interface{~int};func unused(){_=map[int]C(nil)};func main(){}`, false},
		{"channel_conversion", `type C interface{~int};func unused(){_=(chan C)(nil)};func main(){}`, false},
		{"assertion", `type C interface{~int};func unused(v any){_=v.(C)};func main(){}`, false},
		{"method_expression", `type C interface{~int;M()};func unused(){_=C.M};func main(){}`, false},
		{"new", `type C interface{~int};func unused(){_=new(C)};func main(){}`, false},
		{"make", `type C interface{~int};func unused(){_=make([]C,1)};func main(){}`, false},
		{"literal", `type C interface{~int};func unused(){_=[]C{}};func main(){}`, false},
		{"initializer", `type C interface{~int};var v=C(1);func main(){}`, false},
		{"closure", `type C interface{~int};func unused(){_=func(){_=C(1)}};func main(){}`, false},
		{"generic_unused", `type C interface{~int};func unused[T ~int](v T){_=C(v)};func main(){}`, false},
		{"generic_alias", `type C[T any]=interface{~int};func unused(){_=C[int](1)};func main(){}`, false},
		{"generic_local", `type C interface{~int};func unused[T any](){var c C;_=c};func main(){}`, false},
		{"comparable", `func unused(){_=comparable(1)};func main(){}`, false},
		{"comparable_variable", `var c comparable;func main(){}`, false},
		{"shadow_local", `type C interface{~int};func unused(){C:=func(v int)int{return v};_=C(1)};func main(){}`, true},
		{"shadow_multiple", `type C interface{~int};func unused(){C,x:=func(v int)int{return v},1;_=C(x)};func main(){}`, true},
		{"shadow_header", `type C interface{~int};func unused(){if C:=func(v int)int{return v};C(1)==1{}};func main(){}`, true},
		{"shadow_closure", `type C interface{~int};func unused(){_=func(C func(int)int){_=C(1)}};func main(){}`, true},
		{"shadow_alias", `type C interface{~int};func unused(){type C=int;_=C(1)};func main(){}`, true},
		{"shadow_comparable", `var comparable=func(v int)int{return v};func unused(){_=comparable(1)};func main(){}`, true},
		{"shadow_builtin", `type C interface{~int};func unused(){new:=func(v int)int{return v};_=new(1)};func main(){}`, true},
		{"basic_conversion", `type C interface{M()};type V int;func(V)M(){};func unused(){_=C(V(1))};func main(){}`, true},
		{"constraint_declaration", `type C interface{~int};func unused(){type D interface{C};_=0};func main(){}`, true},
		{"method_name", `type C interface{~int};func unused(){type D interface{C()int};var d D;_=d};func main(){}`, true},
		{"parameter_conversion", `func unused[T ~int](v int)T{return T(v)};func main(){}`, true},
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

func TestGenericConstraintValueInterfaceConversion(t *testing.T) {
	source := `package main
type C interface{~int}
type Value[T any]=interface{Value()T}
type Box[T any] struct{Item T}
func(b Box[T])Value()T{return b.Item}
func Read[T any](v Value[T])T{return v.Value()}
func Convert[T C](v int)T{return T(v)}
func main(){
 if Read(Value[int](Box[int]{42}))!=42||Convert[int](41)!=41{panic(1)}
 C,v:=func(v int)int{return v+1},40
 if C(v)!=41{panic(2)}
 {type C=int;if C(42)!=42{panic(3)}}
}`
	graph := genericTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte(source)}})
	prepared := PrepareGenerics(graph)
	if !prepared.Ok {
		t.Fatal(prepared.Message)
	}
	checked := CheckGraph(prepared.Graph)
	if !checked.Ok {
		file := &prepared.Graph.Packages[checked.ErrorPackage].Files[checked.ErrorFile]
		t.Fatalf("check %d at %s token %d, byte %d\n%s", checked.Error, tokenString(&file.File, checked.ErrorToken), checked.ErrorToken, file.File.Tokens[checked.ErrorToken].Start, string(file.Src))
	}
	if strings.Contains(string(prepared.Graph.Packages[checkRootPackage(t, graph)].Files[0].Src), "Value[int]") {
		t.Fatal("generic alias reached concrete graph")
	}
}

func TestGenericLocalAliasPreservesMemberNames(t *testing.T) {
	source := `package main
type Constraint interface{~int}
func Id[T any](v T)T{return v}
func main(){
 C:=func(v int)int{return v+1};_=C(1)
 {type C=int;var s struct{C int};s.C=42;_=s.C;_=C(42)
 var iface interface{C()int};_=iface
 var callback func(C int)int;_=callback
 goto C;C:}
}`
	graph := genericTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte(source)}})
	prepared := PrepareGenerics(graph)
	if !prepared.Ok {
		t.Fatal(prepared.Message)
	}
	text := string(prepared.Graph.Packages[checkRootPackage(t, graph)].Files[0].Src)
	for _, member := range []string{"struct{C int}", "interface{C()int}", "func(C int)int", "goto C"} {
		if !strings.Contains(text, member) {
			t.Fatalf("local alias rewrite changed a member or label name: %s\n%s", member, text)
		}
	}
}
