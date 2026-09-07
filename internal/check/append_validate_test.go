package check

import (
	"testing"

	"renvo.dev/internal/load"
)

func TestAppendOperandValidation(t *testing.T) {
	for _, source := range []string{
		`func main(){_=append(1,2)}`,
		`func main(){_=append("x",'a')}`,
		`func main(){_=append([2]int{},1)}`,
		`func f(v int){_=append(v,2)}`,
		`type M map[int]int;func f(v M){_=append(v,2)}`,
		`func main(){_=append([]int{},1 ...)}`,
		`func main(){_=append([]int{},map[int]int{}...)}`,
		`func main(){_=append([]int{},true...)}`,
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		if result := CheckGraphCore(graph); result.Ok || result.Error != CheckErrBuiltinOperand {
			t.Fatalf("%s: ok=%v error=%d", source, result.Ok, result.Error)
		}
	}
}

func TestAppendControls(t *testing.T) {
	for _, source := range []string{
		`func main(){_=append([]int{})}`,
		`func main(){_=append([]int{},1,2)}`,
		`func main(){_=append([]int{},[]int{1}...)}`,
		`func main(){_=append([]int{},nil...)}`,
		`func main(){_=append([]byte{},"abc"...)}`,
		`type S []int;func f(v S){_=append(v,1)}`,
		`func f(v int){{v:=[]int{};_=append(v,1)}}`,
		`func f(v int){_=func(v []int){_=append(v,1)}}`,
		`func main(){append:=func(a,b int)int{return a+b};_=append(1,2)}`,
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		if result := CheckGraphCore(graph); !result.Ok {
			t.Fatalf("%s: error=%d", source, result.Error)
		}
	}
}

func TestAppendArity(t *testing.T) {
	for _, expression := range []string{"append()", "append([]int{},1,[]int{}...)", "append([]int{}...)"} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nfunc main(){_=" + expression + "}")}})
		if result := CheckGraphCore(graph); result.Ok || result.Error != CheckErrBuiltinArity {
			t.Fatalf("%s: ok=%v error=%d", expression, result.Ok, result.Error)
		}
	}
}
