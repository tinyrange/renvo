package check

import (
	"testing"

	"renvo.dev/internal/load"
)

func TestCopyDeleteOperands(t *testing.T) {
	for _, source := range []string{
		`func main(){_=copy(1,2)}`,
		`func main(){delete([]int{},0)}`,
		`func f(v int){_=copy(v,[]int{})}`,
		`func f(v int){_=copy([]int{},v)}`,
		`func main(){_=copy("a","b")}`,
		`func main(){_=copy([2]int{},[2]int{})}`,
		`func main(){delete(make(chan int),0)}`,
		`type S []int;func f(v S){delete(v,0)}`,
		`type M map[int]int;func f(v M){_=copy(v,v)}`,
		`func main(){v:=[]int{};delete(v,0)}`,
		`var v=[]int{};func main(){delete(v,0)}`,
		`func main(){delete(map[int]int{},"x")}`,
		`func main(){_=copy([]int{},[]string{})}`,
		`func main(){_=copy([]rune{},"abc")}`,
		`type B byte;func f(v []B){_=copy(v,"abc")}`,
		`type N int;func f(a []int,b []N){_=copy(a,b)}`,
		`type A int;type B int;func f(a []A,b []B){_=copy(a,b)}`,
		`func main(){a:=make([]int,2);b:=[]bool{};_=copy(a,b)}`,
		`var a=[]int{};var b=[]string{};func main(){_=copy(a,b)}`,
		`func f(a []*int,b []*string){_=copy(a,b)}`,
		`func f(a [][]int,b [][]string){_=copy(a,b)}`,
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		if result := CheckGraphCore(graph); result.Ok || result.Error != CheckErrBuiltinOperand {
			t.Fatalf("%s: ok=%v error=%d", source, result.Ok, result.Error)
		}
	}
}

func TestCopyDeleteControls(t *testing.T) {
	for _, source := range []string{
		`func main(){_=copy([]int{},[]int{})}`,
		`func main(){_=copy([]byte{},"abc")}`,
		`type S []int;func f(a,b S){_=copy(a,b)}`,
		`type M map[int]int;func f(v M){delete(v,0)}`,
		`func f(v int){{v:=[]int{};_=copy(v,v)}}`,
		`func f(v int){_=func(v []int){_=copy(v,v)}}`,
		`func main(){copy:=func(a,b int)int{return a+b};_=copy(1,2)}`,
		`func delete(a,b int){};func main(){delete(1,2)}`,
		`func main(){_=copy(map[int][]int{1:[]int{2}}[1],[]int{})}`,
		`func main(){delete(struct{m map[int]int}{m:map[int]int{}}.m,0)}`,
		`type B = byte;type Bytes []B;func f(a Bytes){_=copy(a,"abc")}`,
		`type A []int;type B []int;func f(a A,b B){_=copy(a,b)}`,
		`type N int;type Alias = N;func f(a []N,b []Alias){_=copy(a,b)}`,
		`func f(a []byte,b []uint8){_=copy(a,b)}`,
		`func f(a []rune,b []int32){_=copy(a,b)}`,
		`type N int;func main(){type N = string;a:=[]N{};_=copy(a,[]string{})}`,
		`func f(a []int){{a:=[]string{};_=copy(a,[]string{})};_=copy(a,[]int{})}`,
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		if result := CheckGraphCore(graph); !result.Ok {
			t.Fatalf("%s: error=%d", source, result.Error)
		}
	}
}

func TestCopyDeleteArity(t *testing.T) {
	for _, expression := range []string{"copy()", "copy(1)", "copy(1,2,3)", "copy([]int{},[]int{}...)", "delete()", "delete(1)", "delete(1,2,3)"} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nfunc main(){" + expression + "}")}})
		if result := CheckGraphCore(graph); result.Ok || result.Error != CheckErrBuiltinArity {
			t.Fatalf("%s: ok=%v error=%d", expression, result.Ok, result.Error)
		}
	}
}
