package check

import (
	"renvo.dev/internal/load"
	"testing"
)

func TestGenericImmediateClosureArrayParameters(t *testing.T) {
	for _, source := range []string{
		`func F[T any](){defer func(value [0]T){_=value}([0]T{})};func main(){F[int]()}`,
		`func F[T any](v T){func(value [1]T){_=value}([1]T{v})};func main(){F(42)}`,
		`func F[T any](){go func(value [0]T){_=value}([0]T{})};func main(){F[int]()}`,
	} {
		graph := genericTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main;" + source)}})
		if result := PrepareGenerics(graph); !result.Ok {
			t.Fatalf("%s: %s", source, result.Message)
		}
	}
}
