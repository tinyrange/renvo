package check

import (
	"renvo.dev/internal/load"
	"testing"
)

func TestGenericPreparationMethodSelections(t *testing.T) {
	for _, test := range []struct {
		name, source string
		valid        bool
	}{
		{"promoted_expression", `type Hidden interface{boxed()int}; type Mixed struct{Padding int;Hidden}; func F[T interface{boxed()int}]()func(T)int{return T.boxed};func main(){_=F[Mixed]()}`, true},
		{"interface_value", `type Hidden interface{boxed()int};func F[T interface{boxed()int}](v T)func()int{return v.boxed};func main(){var v Hidden;_=F(v)}`, true},
		{"if_initializer_inference", `func F[T any](v T)T{return v};func main(){var v any;if x,ok:=v.(interface{boxed()int});ok{_=F(x)}}`, true},
		{"deferred_literal", `func F[T any](v T)T{defer func(){_=recover()}();return func(x T)T{return x}(v)};func main(){_=F(42)}`, true},
		{"expression_needs_receiver", `func F[T interface{boxed()int}]()func()int{return T.boxed};func main(){}`, false},
		{"value_has_bound_receiver", `func F[T interface{boxed()int}](v T)func(T)int{return v.boxed};func main(){}`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			graph := genericTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main;" + test.source)}})
			got := PrepareGenerics(graph)
			if got.Ok != test.valid {
				t.Fatalf("ok=%v want=%v: %s", got.Ok, test.valid, got.Message)
			}
			if got.Ok {
				if checked := CheckGraph(got.Graph); !checked.Ok {
					t.Fatalf("concrete check: %d at %d", checked.Error, checked.ErrorToken)
				}
			}
		})
	}
}
