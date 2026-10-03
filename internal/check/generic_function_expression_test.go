package check

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"renvo.dev/internal/load"
)

func TestGenericFunctionExpressions(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		valid        bool
	}{
		{"anonymous_conversion", `type F func(int)int;func Good(f F){g:=(func(int)int)(f);_=g(42)}`, true},
		{"conversion_nil_comparison", `type F func(int)int;func Good(f F){g:=(func(int)int)(f);_=g!=nil}`, true},
		{"nil_conversion", `func Good(){f:=(func(int)int)(nil);_=f==nil}`, true},
		{"nested_parentheses", `type F func(int)int;func Good(f F){g:=((func(int)int))(f);_=g(42)}`, true},
		{"mismatched_conversion", `type F func(string)int;func Bad(f F){_=(func(int)int)(f)}`, false},
		{"function_from_integer", `func Bad(){_=(func(int)int)(42)}`, false},
		{"pointer_result_literal", `func Good(){f:=func(v int)*int{return &v};_=*f(42)}`, true},
		{"slice_pointer_result_literal", `func Good(){f:=func(v []*int)*int{return v[0]};_=f}`, true},
		{"pointer_result_immediate_call", `func Good(){v:=(func(v int)*int{return &v})(42);_=*v}`, true},
		{"pointer_result_body_checked", `func Bad(){_=func(v int)*int{return "wrong"}}`, false},
		{"function_result_literal", `func Good(){f:=func()func()*int{return func()*int{v:=42;return &v}};_=f()()}`, true},
		{"generic_pointer_closure", `func Make[T any](v T)func()*T{return func()*T{return &v}};func Good(){_=*Make(42)()}`, true},
		{"generic_anonymous_conversion", `type F[T any]func(T)T;func Convert[T any](f F[T])func(T)T{return (func(T)T)(f)};func Good(){_=Convert(F[int](func(v int)int{return v}))}`, true},
		{"anonymous_struct_parameter", `func Good(){f:=func(v struct{Value int})int{return v.Value};_=f(struct{Value int}{42})}`, true},
		{"anonymous_struct_result", `func Good(){f:=func(v int)struct{Value int}{return struct{Value int}{v}};_=f(42)}`, true},
		{"anonymous_interface_parameter", `func Good(){f:=func(v interface{Value()int})int{return v.Value()};_=f}`, true},
		{"nested_anonymous_result", `func Good(){f:=func()func()struct{Value int}{return func()struct{Value int}{return struct{Value int}{42}}};_=f()()}`, true},
		{"parenthesized_map", `func Good(){var m map[string]int;_=(map[string]int)(m)}`, true},
		{"parenthesized_map_invalid", `func Bad(){_=(map[string]int)(42)}`, false},
		{"parenthesized_struct", `func Good(){_=(struct{Value int})(struct{Value int}{42})}`, true},
		{"parenthesized_interface", `func Good(v any){_=(interface{})(v)}`, true},
		{"parenthesized_constraint_interface", `func Bad(){_=(interface{~int})(42)}`, false},
		{"parenthesized_channel", `func Good(v chan int){_=(chan<- int)(v)}`, true},
		{"parenthesized_receive_channel", `func Good(v chan int){_=(<-chan int)(v)}`, true},
		{"parenthesized_channel_invalid", `func Bad(){_=(chan int)(42)}`, false},
		{"anonymous_result_body_checked", `func Bad(){_=func()struct{Value int}{return 42}}`, false},
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
					t.Fatalf("concrete error=%d token=%d", checked.Error, checked.ErrorToken)
				}
			}
			if valid != tc.valid {
				t.Fatalf("valid=%v accepted=%v: %s", tc.valid, valid, prepared.Message)
			}
		})
	}
}
