package check

import (
	"testing"

	"renvo.dev/internal/load"
)

func TestGenericControlHeaderClosures(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		valid        bool
	}{
		{"condition", `func Id[T any](v T)T{return v};func main(){for func()bool{for _,v:=range []int{1,2}{_=Id(v)};return true}(){break}}`, true},
		{"non_boolean_condition", `func Id[T any](v T)T{return v};func Loop[T any](){for func()int{for _,v:=range []int{1,2}{_=Id(v)};return 1}(){break}};func main(){Loop[int]()}`, false},
		{"range_operand", `func Sum[T ~int](values []T)T{var sum T;for _,v:=range func()[]T{for range values{};return values}(){sum+=v};return sum};func main(){_=Sum([]int{1,2})}`, true},
		{"invalid_inner_range", `func Id[T any](v T)T{return v};func main(){for func()bool{for range true{};return Id(true)}(){break}}`, false},
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
