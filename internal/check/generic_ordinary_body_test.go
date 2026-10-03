package check

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"renvo.dev/internal/load"
)

func TestGenericPreparationChecksOrdinaryBodies(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		valid        bool
	}{
		{"string_integer", `func Bad(){var a string;_=a+1}`, false},
		{"integer_string", `func Bad(){var a int;_=a+"x"}`, false},
		{"boolean_addition", `func Bad(){var a bool;_=a+true}`, false},
		{"named_number_identity", `type A int;type B int;func Bad(a A,b B){_=a+b}`, false},
		{"string_conversion", `func Bad(){_=int("wrong")}`, false},
		{"missing_return", `func Bad()int{}`, false},
		{"call_argument", `func Take(v int){};func Bad(){Take("wrong")}`, false},
		{"unknown_name", `func Bad(){_=unknown}`, false},
		{"string_addition", `func Good(a string)string{return a+"x"}`, true},
		{"integer_addition", `func Good(a int)int{return a+1}`, true},
		{"integer_exact_float", `func Good(a int)int{return a+1.0}`, true},
		{"named_number", `type A int;func Good(a A)A{return a+1}`, true},
		{"generic_named_parameter", `type Box[T any]struct{Value T};func Good(a Box[int])int{return a.Value+1}`, true},
		{"generic_alias_parameter", `type Values[T any]=[]T;func Good(a Values[int])int{return a[0]+1}`, true},
		{"method", `type Value int;func(v Value)Good()int{return int(v)+1}`, true},
		{"closure", `func Good(a string)func(string)string{return func(a string)string{return a+"x"}}`, true},
		{"grouped_inferred_callbacks", `func First()int{return 17};func Second()int{return 19};func Good(){var(f=(First);g=(Second));_,_=Id(f),Id(g)}`, true},
		{"grouped_typed_shadow", `func First()int{return 17};func Good(){var(First func()int=(First));_=Id(First)}`, true},
		{"grouped_inferred_shadow", `func First()int{return 17};func Good(){var(First=(First));_=Id(First)}`, true},
		{"grouped_multiple_names", `func First()int{return 17};func Second()int{return 19};func Good(){var(f,g=(First),(Second));_,_=Id(f),Id(g)}`, true},
		{"grouped_tuple", `func Pair()(int,string){return 17,"yes"};func Good(){var(a,b=Pair());_,_=Id(a),Id(b)}`, true},
		{"grouped_tuple_wrong_type", `func Pair()(int,string){return 17,"yes"};func Good(){var(a,b int=Pair());_,_=Id(a),Id(b)}`, false},
		{"grouped_nested_semicolons", `func Good(){var(f=func()int{var(n=17;m=19);return Id(n)+m};g=23);_,_=Id(f),Id(g)}`, true},
		{"grouped_multiline", "func Good(){var(\nf=func()int{\n n:=17\n return n\n}\n g=\n23\n);_,_=Id(f),Id(g)}", true},
		{"grouped_wrong_callback", `func First()string{return "wrong"};func Good(){var(f func()int=First);_=Id(f)}`, false},
		{"grouped_wrong_second_spec", `func Good(){var(n=17;m int="wrong");_,_=Id(n),Id(m)}`, false},
		{"type_switch_local", `func Good(v any)int{switch v.(type){case int:value:=v.(int);return Id(value);default:return 0}}`, true},
		{"literal_string_index", `func Good(n int)byte{return "abc"[n]}`, true},
		{"constant_string_index", `const table="abc";func Good(n int)int{return int(table[n])}`, true},
		{"indexed_string_append", `func Good(n int)[]byte{return append([]byte{},"abc"[n])}`, true},
		{"indexed_string_bounds", `func Bad(){_="abc"[3]}`, false},
		{"indexed_string_not_constant", `func Bad(){const n="abc"[0];_=n}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte("package main;func Id[T any](v T)T{return v};" + tc.source + ";func main(){}\n")
			set := token.NewFileSet()
			file, err := parser.ParseFile(set, "main.go", source, 0)
			if err != nil {
				t.Fatal(err)
			}
			config := types.Config{}
			_, err = config.Check("example.com/case", set, []*ast.File{file}, nil)
			if (err == nil) != tc.valid {
				t.Fatalf("Go validity differs from test: %v", err)
			}
			graph := genericTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: source}})
			prepared := PrepareGenerics(graph)
			valid := prepared.Ok
			if prepared.Ok {
				checked := CheckGraphCore(prepared.Graph)
				valid = checked.Ok
				if !checked.Ok && tc.valid {
					t.Fatalf("concrete check error=%d token=%d", checked.Error, checked.ErrorToken)
				}
			}
			if valid != tc.valid {
				t.Fatalf("valid=%v accepted=%v: %s", tc.valid, valid, prepared.Message)
			}
		})
	}
}
