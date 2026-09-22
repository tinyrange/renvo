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
		`func main(){_=append([]int{},[]string{}...)}`,
		`func main(){_=append([]rune{},"abc"...)}`,
		`type B byte;func f(v []B){_=append(v,"abc"...)}`,
		`type N int;func f(a []N,b []int){_=append(a,b...)}`,
		`func f(a []*int,b []*string){_=append(a,b...)}`,
		`func f(a [][]int,b [][]string){_=append(a,b...)}`,
		`func main(){_=append([]int{},"x")}`,
		`func main(){_=append([]string{},1)}`,
		`func main(){_=append([]bool{},0)}`,
		`func main(){_=append([]float64{},true)}`,
		`func main(){_=append([]int{},1.5)}`,
		`func main(){_=append([]int{},nil)}`,
		`func main(){_=append([]int{},1,"x")}`,
		`type N int;func f(a []N,b int){_=append(a,b)}`,
		`type N int;func f(a []int,b N){_=append(a,b)}`,
		`func f(a []int,b int32){_=append(a,b)}`,
		`func f(a []float64,b float32){_=append(a,b)}`,
		`type N string;func f(a []N){b:="x";_=append(a,b)}`,
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
		`type N int;type A []N;type B []N;func f(a A,b B){_=append(a,b...)}`,
		`type B = byte;type Bytes []B;func f(v Bytes){_=append(v,"abc"...)}`,
		`func f(a []rune,b []int32){_=append(a,b...)}`,
		`func f(a []byte,b []uint8){_=append(a,b...)}`,
		`func f(a []int){{a:=[]string{};_=append(a,[]string{}...)};_=append(a,nil...)}`,
		`func f(a []int){_=func(a []string){_=append(a,[]string{}...)}}`,
		`type N int;type A = N;func f(a []N,b A){_=append(a,b,1,2.0)}`,
		`func f(a []byte,b uint8){_=append(a,b,'a')}`,
		`func f(a []rune,b int32){_=append(a,b,1)}`,
		`func f(a []string){b:="x";_=append(a,b)}`,
		`func f(a []bool){b:=true;_=append(a,b)}`,
		`func f(a []interface{}){_=append(a,1,"x",nil)}`,
		`func f(a []*int){_=append(a,nil)}`,
		`func f(a []int){{a:=[]string{};_=append(a,"x")}}`,
		`type N float64;func f(a []N){_=append(a,1,1.5)}`,
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
