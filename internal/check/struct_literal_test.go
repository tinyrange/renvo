package check

import (
	"renvo.dev/internal/load"
	"testing"
)

func TestStructLiteralFieldLists(t *testing.T) {
	for _, source := range []string{
		"type S struct { X int }; func main() { _ = S{X:1, X:2} }",
		"type S struct { X,Y int }; func main() { _ = S{1, Y:2} }",
		"type S struct { X,Y int }; func main() { _ = S{X:1, 2} }",
		"type S struct { X int }; func main() { _ = S{Y:1} }",
		"type S struct { X int }; func main() { _ = S{1,2} }",
		"type S struct { X,Y int }; func main() { _ = S{1} }",
		"type S struct { _ int }; func main() { _ = S{_:1} }",
		"type S struct { X int }; var x = S{X:1, X:2}; func main() {}",
		"type S struct { X int }; type A = S; func main() { _ = A{X:1, X:2} }",
		"type S struct { X int }; func f() S { return S{X:1, X:2} }; func main() {}",
		"func main() { _ = struct { X int }{X:1, X:2} }",
	} {
		t.Run(source, func(t *testing.T) {
			graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
			result := CheckGraphCore(graph)
			if result.Ok || result.Error != CheckErrStructLiteral {
				t.Fatalf("ok=%v error=%d, want struct literal diagnostic", result.Ok, result.Error)
			}
		})
	}
}

func TestValidStructLiteralFieldLists(t *testing.T) {
	for _, source := range []string{
		"type S struct { X,Y int }; func main() { _ = S{}; _ = S{1,2}; _ = S{Y:2} }",
		"type S struct { _ int; X int }; func main() { _ = S{1,2}; _ = S{X:2} }",
		"type S struct { X int }; func main() { type S map[int]int; k:=1; _ = S{k:1,k:2} }",
		"func main() { k:=1; _ = map[int]int{k:1,k:2}; _ = [4]int{2:1,2} }",
		"type S struct { X int }; func main() { _ = []S{{X:1},{X:2}} }",
		"type S struct { X int }; type A = S; func main() { _ = A{X:1} }",
		"type S struct { X int }; var f = func() { type S map[int]int; k:=1; _ = S{k:1,k:2} }; func main() {}",
	} {
		t.Run(source, func(t *testing.T) {
			graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
			if result := CheckGraphCore(graph); !result.Ok {
				t.Fatalf("error=%d token=%d", result.Error, result.ErrorToken)
			}
		})
	}
}
