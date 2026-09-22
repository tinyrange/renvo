package check

import (
	"renvo.dev/internal/load"
	"testing"
)

func TestRecursiveValueTypes(t *testing.T) {
	for _, source := range []string{
		"type T struct { X T }",
		"type T [0]T",
		"type T struct { _ T }",
		"type A B; type B struct { Value A }",
		"type A = B; type B [2]A",
		"type A struct { Field [2]struct { Nested A } }",
		"type A B; type B A",
	} {
		t.Run(source, func(t *testing.T) {
			graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source + "\nfunc main(){}")}})
			result := CheckGraphCore(graph)
			if result.Ok || result.Error != CheckErrRecursiveType {
				t.Fatalf("ok=%v error=%d, want recursive type diagnostic", result.Ok, result.Error)
			}
		})
	}
}

func TestRecursiveIndirectTypes(t *testing.T) {
	for _, source := range []string{
		"type T struct { Next *T }",
		"type T []T",
		"type T map[string]T",
		"type T chan T",
		"type T func() T",
		"type T *T",
		"type A struct { B B }; type B struct { A *A }",
		"type A = B; type B struct { Next *A }",
		"type A struct { B B; C B }; type B [2]int",
	} {
		t.Run(source, func(t *testing.T) {
			graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source + "\nfunc main(){}")}})
			if result := CheckGraphCore(graph); !result.Ok {
				t.Fatalf("valid indirect type rejected: error=%d token=%d", result.Error, result.ErrorToken)
			}
		})
	}
}
