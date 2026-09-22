package check

import (
	"testing"

	"renvo.dev/internal/load"
)

func TestExplicitInterfaceMethodUniqueness(t *testing.T) {
	for _, source := range []string{
		"type I interface { M(int); M(string) }",
		"type I interface { M(int); M(int) }",
		"func main() { type I interface { M(); M() } }",
		"var x interface { M(); M() }",
		"type I interface { M(interface{ N(); N() }) }",
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		result := CheckGraphCore(graph)
		if result.Ok || result.Error != CheckErrDuplicate {
			t.Fatalf("%s: ok=%v error=%d", source, result.Ok, result.Error)
		}
	}
}

func TestExplicitInterfaceMethodsPreserveEmbedding(t *testing.T) {
	for _, source := range []string{
		"type A interface { M() }; type B interface { M() }; type I interface { A; B }",
		"type A interface { M() }; type I interface { A; M() }",
		"type I interface { M(); N() }",
		"type I interface { M(interface{ M() }) }",
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		result := CheckGraphCore(graph)
		if !result.Ok {
			t.Fatalf("%s: error=%d token=%d", source, result.Error, result.ErrorToken)
		}
	}
}
