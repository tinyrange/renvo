package check

import (
	"testing"

	"renvo.dev/internal/load"
)

func TestCallArityRejectsDefiniteCounts(t *testing.T) {
	for _, source := range []string{
		`func f(x ...int) {}; func main() { f(1, []int{2}...) }`,
		`func f(x,y int) {}; func main(){f(1)}`,
		`func f(x int,y int) {}; func main(){f(1)}`,
		`func f(x int,y int) {}; func main(){v:=1;f(v)}`,
		`func f(x int,y ...int) {}; func main(){f(1,2,[]int{3}...)}`,
		`func f(x int,y ...int) {}; func main(){f([]int{3}...)}`,
		`func f(x []int) {}; func main(){f([]int{3}...)}`,
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		if result := CheckGraphCore(graph); result.Ok || result.Error != CheckErrCallArity {
			t.Fatalf("%s: ok=%v error=%d", source, result.Ok, result.Error)
		}
	}
}

func TestCallArityAllowsTupleAndVariadicArguments(t *testing.T) {
	for _, source := range []string{
		`func f(x,y int) {}; func pair()(int,int){return 1,2};func main(){f(pair())}`,
		`func f(x,y int) {}; func pair()(int,int){return 1,2};func main(){f((pair()))}`,
		`func f(x ...int) {}; func main(){f();f(1);f(1,2);f([]int{1,2}...)}`,
		`func f(x int,y ...int) {}; func main(){f(1);f(1,2);f(1,2,3);f(1,[]int{2,3}...)}`,
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		if result := CheckGraphCore(graph); !result.Ok {
			t.Fatalf("%s: error=%d token=%d", source, result.Error, result.ErrorToken)
		}
	}
}
