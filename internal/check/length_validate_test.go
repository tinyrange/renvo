package check

import (
	"testing"

	"renvo.dev/internal/load"
)

func TestLengthCapacityArity(t *testing.T) {
	for _, expression := range []string{"len()", "len(1, 2)", "len([]int{}...)", "cap()", "cap(1, 2)", "cap([]int{}...)"} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nfunc main(){_=" + expression + "}")}})
		if result := CheckGraphCore(graph); result.Ok || result.Error != CheckErrBuiltinArity {
			t.Fatalf("%s: ok=%v error=%d", expression, result.Ok, result.Error)
		}
	}
}

func TestLengthCapacityScalarOperands(t *testing.T) {
	for _, source := range []string{
		`func main(){_=len(1)}`,
		`func main(){_=len(true)}`,
		`func main(){_=len(nil)}`,
		`func main(){v:=1;_=len(v)}`,
		`type N int;func f(v N){_=len(v)}`,
		`var v=1;func main(){_=len(v)}`,
		`func f(v bool){_=cap(v)}`,
		`type S string;func f(v S){_=cap(v)}`,
		`func main(){v:=2i;_=cap(v)}`,
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		if result := CheckGraphCore(graph); result.Ok || result.Error != CheckErrBuiltinOperand {
			t.Fatalf("%s: ok=%v error=%d", source, result.Ok, result.Error)
		}
	}
}

func TestLengthCapacityScalarControls(t *testing.T) {
	for _, source := range []string{
		`func main(){_=len("abc")}`,
		`type S string;func f(v S){_=len(v)}`,
		`func f(v int){{v:="abc";_=len(v)}}`,
		`func f(v int){_=func(v string){_=len(v)}}`,
		`func main(){len:=func(v int)int{return v};_=len(1)}`,
		`func main(){var v []int;_=len(v);_=cap(v)}`,
		`func main(){var v [2]int;_=len(v);_=cap(v)}`,
		`func main(){var v *[2]int;_=len(v);_=cap(v)}`,
		`func main(){var v map[int]int;_=len(v)}`,
		`func main(){var v chan int;_=len(v);_=cap(v)}`,
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		if result := CheckGraphCore(graph); !result.Ok {
			t.Fatalf("%s: error=%d", source, result.Error)
		}
	}
}
