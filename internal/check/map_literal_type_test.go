package check

import (
	"testing"

	"renvo.dev/internal/load"
)

func TestMapLiteralPrimitiveTypes(t *testing.T) {
	for _, source := range []string{
		"func main() { _ = map[int]int{\"x\": 1} }",
		"func main() { _ = map[int]int{1: \"x\"} }",
		"var x = map[string]int{1: 2}",
		"type Key int; type M map[Key]string; var x = M{1: 2}",
		"type Key int; type M map[Key]string; type N = M; var x = N{\"x\": \"y\"}",
		"func main() { _ = map[int]bool{1: 2} }",
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		result := CheckGraphCore(graph)
		if result.Ok || result.Error != CheckErrType {
			t.Fatalf("%s: ok=%v error=%d", source, result.Ok, result.Error)
		}
	}
}

func TestMapLiteralPrimitiveControls(t *testing.T) {
	for _, source := range []string{
		"func main() { _ = map[int]int{1: 2} }",
		"type Key int; type M map[Key]string; var x = M{1: \"x\"}",
		"func main() { _ = map[any]any{1: \"x\"} }",
		"type M map[int]int; func main() { type M map[string]string; _ = M{\"x\": \"y\"} }",
		"func main() { type int string; _ = map[int]int{\"x\": \"y\"} }",
		"func main() { _ = map[int]byte{1: \"x\"[0]} }",
		"func main() { _ = map[int]int{1.0: 2.0} }",
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		if result := CheckGraphCore(graph); !result.Ok {
			t.Fatalf("%s: error=%d token=%d", source, result.Error, result.ErrorToken)
		}
	}
}
