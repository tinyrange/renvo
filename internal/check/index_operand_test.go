package check

import (
	"testing"

	"renvo.dev/internal/load"
)

func TestIndexOperandValidation(t *testing.T) {
	for _, source := range []string{
		`func main(){_=1[0]}`,
		`func main(){v:=1;_=v[0]}`,
		`type N int;func f(v N){_=v[0]}`,
		`func f(v bool){_=v[0]}`,
		`func f(v chan int){_=v[0]}`,
		`type S struct{};func f(v S){_=v[0]}`,
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		if result := CheckGraphCore(graph); result.Ok || result.Error != CheckErrOperand {
			t.Fatalf("%s: ok=%v error=%d", source, result.Ok, result.Error)
		}
	}
}

func TestIndexOperandControls(t *testing.T) {
	for _, source := range []string{
		`func main(){_="abc"[0]}`,
		`func f(v []int){_=v[0]}`,
		`func f(v [2]int){_=v[0]}`,
		`func f(v *[2]int){_=v[0]}`,
		`func f(v map[int]int){_=v[0]}`,
		`func f(v int){{v:="abc";_=v[0]}}`,
		`func f(v int){_=func(v []int){_=v[0]}}`,
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		if result := CheckGraphCore(graph); !result.Ok {
			t.Fatalf("%s: error=%d", source, result.Error)
		}
	}
}
