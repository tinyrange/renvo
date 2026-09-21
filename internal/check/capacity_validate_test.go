package check

import (
	"testing"

	"renvo.dev/internal/load"
)

func TestCapacityLiteralValidation(t *testing.T) {
	for _, expression := range []string{"map[int]int{}", "(map[int]int{1: 2})", "struct{}{}", "struct{ x int }{x: 1}", "1", "1.2", "1i", "'a'", "\"abc\"", "(\"abc\")"} {
		t.Run(expression, func(t *testing.T) {
			graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nfunc main() { _ = cap(" + expression + ") }\n")}})
			result := CheckGraphCore(graph)
			if result.Ok || result.Error != CheckErrBuiltinOperand {
				t.Fatalf("ok=%v error=%d, want builtin operand diagnostic", result.Ok, result.Error)
			}
		})
	}
}

func TestCapacityPreservesValidExpressionsAndShadowing(t *testing.T) {
	for _, source := range []string{
		"func main() { _ = cap([2]int{}) }",
		"func main() { _ = cap(&[2]int{}) }",
		"func main() { _ = cap([]int{1, 2}) }",
		"func main() { _ = cap(make(chan int, 2)) }",
		"func main() { _ = cap(map[int][]int{1: []int{2}}[1]) }",
		"func main() { _ = cap(struct{ x []int }{x: []int{1}}.x) }",
		"func main() { cap := func(x string) int { return 3 }; _ = cap(\"abc\") }",
		"func cap(x map[int]int) int { return 1 }; func main() { _ = cap(map[int]int{}) }",
	} {
		t.Run(source, func(t *testing.T) {
			graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source + "\n")}})
			result := CheckGraphCore(graph)
			if !result.Ok {
				t.Fatalf("error=%d token=%d", result.Error, result.ErrorToken)
			}
		})
	}
}
