package check

import (
	"testing"

	"renvo.dev/internal/load"
)

func TestRangeInvalidOperands(t *testing.T) {
	for _, source := range []string{
		`func main(){for range true{}}`,
		`func main(){for range nil{}}`,
		`func main(){for range 2.0{}}`,
		`func main(){for range 2i{}}`,
		`func f(v bool){for range v{}}`,
		`type B bool;func f(v B){for range v{}}`,
		`func main(){v:=false;for range v{}}`,
		`func main(){for range struct{}{}{}}`,
		`type S struct{x int};func main(){for range (S{x:1}){}}`,
		`type S struct{};type A=S;func f(v A){for range v{}}`,
		`func main(){v:=struct{}{};for range v{}}`,
		`var v=struct{}{};func main(){for range v{}}`,
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		if result := CheckGraphCore(graph); result.Ok || result.Error != CheckErrOperand {
			t.Fatalf("%s: ok=%v error=%d", source, result.Ok, result.Error)
		}
	}
}

func TestRangeOperandControls(t *testing.T) {
	for _, source := range []string{
		`func main(){for range 2{}}`,
		`func main(){for range "abc"{}}`,
		`func main(){for range []int{1,2}{}}`,
		`func main(){for range [2]int{}{}}`,
		`func main(){for range &[2]int{}{}}`,
		`func main(){for range map[int]int{}{}}`,
		`func f(v chan int){for range v{}}`,
		`func f(v bool){{v:=2;for range v{}}}`,
		`func f(v bool){_=func(v []int){for range v{}}}`,
		`type S struct{};func main(){type S []int;for range S{}{}}`,
		`func main(){for range struct{x []int}{x:[]int{1}}.x{}}`,
		`func main(){for range map[int][]int{1:[]int{2}}[1]{}}`,
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		if result := CheckGraphCore(graph); !result.Ok {
			t.Fatalf("%s: error=%d", source, result.Error)
		}
	}
}
