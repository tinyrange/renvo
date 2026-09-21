package check

import (
	"renvo.dev/internal/load"
	"testing"
)

func TestMakeValidation(t *testing.T) {
	for _, tc := range []struct {
		source string
		code   int
	}{
		{`func main(){_=make(int)}`, CheckErrBuiltinOperand},
		{`func main(){_=make([2]int,2)}`, CheckErrBuiltinOperand},
		{`func main(){_=make(*int)}`, CheckErrBuiltinOperand},
		{`type A [2]int;func main(){_=make(A,2)}`, CheckErrBuiltinOperand},
		{`func main(){_=make(true)}`, CheckErrBuiltinOperand},
		{`func main(){_=make([]int{},2)}`, CheckErrBuiltinOperand},
		{`func main(){_=make()}`, CheckErrBuiltinArity},
		{`func main(){_=make([]int)}`, CheckErrBuiltinArity},
		{`func main(){_=make(map[int]int,1,2)}`, CheckErrBuiltinArity},
		{`func main(){_=make(chan int,1,2)}`, CheckErrChannel},
		{`func main(){_=make([]int,1,2,3)}`, CheckErrBuiltinArity},
		{`func main(){_=make([]int,-1)}`, CheckErrBuiltinOperand},
		{`func main(){_=make([]int,3,2)}`, CheckErrBuiltinOperand},
		{`func main(){_=make(map[int]int,"x")}`, CheckErrBuiltinOperand},
		{`func main(){_=make([]int,1.5)}`, CheckErrBuiltinOperand},
		{`func main(){_=make([]int,1<<80)}`, CheckErrBuiltinOperand},
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + tc.source)}})
		if result := CheckGraphCore(graph); result.Ok || result.Error != tc.code {
			t.Fatalf("%s: ok=%v error=%d token=%d", tc.source, result.Ok, result.Error, result.ErrorToken)
		}
	}
}

func TestMakeValidationControls(t *testing.T) {
	for _, source := range []string{
		`func main(){_=make([]int,2,3)}`,
		`type S []int;type A=S;func main(){_=make(A,2)}`,
		`func main(){_=make([]struct{X int},2)}`,
		`func main(){_=make(map[struct{X int}]int,2)}`,
		`func main(){_=make(chan int);_=make(chan int,2)}`,
		`func main(){_=make(map[int]int);_=make(map[int]int,2)}`,
		`func main(){_=make([]int,1.0)}`,
		`func make(x int)int{return x};func main(){_=make(1)}`,
		`func main(){make:=func(x int)int{return x};_=make(1)}`,
		`type A [2]int;func main(){type A []int;_=make(A,2)}`,
		`const N=-1;func main(){const N=2;_=make([]int,N)}`,
		`func f(true int){_=make([]int,true)}`,
		`func main(){_=make([]int,"x"[0])}`,
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		if result := CheckGraphCore(graph); !result.Ok {
			t.Fatalf("%s: error=%d token=%d", source, result.Error, result.ErrorToken)
		}
	}
}
