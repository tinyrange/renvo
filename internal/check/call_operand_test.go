package check

import (
	"testing"

	"renvo.dev/internal/load"
)

func TestMultiValueCallOperands(t *testing.T) {
	for _, expression := range []string{"1 + pair()", "pair() + 1", "-pair()", "!pair()", "(pair()) + 1", "pair()[0]", "pair()()"} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nfunc pair() (int, int) { return 1, 2 }\nfunc main() { _ = " + expression + " }\n")}})
		result := CheckGraphCore(graph)
		if result.Ok || result.Error != CheckErrOperand {
			t.Fatalf("%s: ok=%v error=%d", expression, result.Ok, result.Error)
		}
	}
	graph := checkTestGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nimport \"example.com/case/lib\"\nfunc main() { _ = 1 + lib.Pair() }\n")},
		{Path: "/repo/case/lib/lib.go", Src: []byte("package lib\nfunc Pair() (a, b int) { return 1, 2 }\n")},
	})
	if result := CheckGraphCore(graph); result.Ok || result.Error != CheckErrOperand {
		t.Fatalf("imported call: ok=%v error=%d", result.Ok, result.Error)
	}
}

func TestSingleValueCallOperandControls(t *testing.T) {
	for _, source := range []string{
		"func f() int { return 1 }; func main() { _ = 1 + f() }",
		"func f() (x int) { return 1 }; func main() { _ = (f()) + 1 }",
		"func f() (x int,) { return 1 }; func main() { _ = f() + 1 }",
		"func pair() (int,int) { return 1,2 }; func main() { a,b := pair(); _,_ = a,b }",
		"func pair() (int,int) { return 1,2 }; func f(a,b int) {}; func main() { f(pair()) }",
		"func pair() (int,int) { return 1,2 }; func f(a,b int) int { return a+b }; func main() { _ = f(pair()) + 1 }",
		"func pair() (int,int) { return 1,2 }; func f() (int,int) { return pair() }",
		"func pair() (int,int) { return 1,2 }; func main() { pair := func() int { return 3 }; _ = 1 + pair() }",
		"func f() {}; func main() { x := 1; p := &x; f()\n*p = 2 }",
		"func pair() (int,int) { return 1,2 }; func main() { x := 1; p := &x; pair()\n*p = 2 }",
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source + "\n")}})
		if result := CheckGraphCore(graph); !result.Ok {
			t.Fatalf("%s: error=%d token=%d", source, result.Error, result.ErrorToken)
		}
	}
}
