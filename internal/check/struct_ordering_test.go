package check

import (
	"testing"

	"renvo.dev/internal/load"
)

func TestStructOrderingRejected(t *testing.T) {
	for _, source := range []string{
		`func main(){_=struct{X int}{1}<struct{X int}{2}}`,
		`type S struct{x int};func f(a,b S){_=a<=b}`,
		`type S struct{};type A=S;func f(a,b A){_=a>b}`,
		`func main(){a:=struct{}{};b:=a;_=a>=b}`,
		`var a=struct{}{};func main(){_=a<a}`,
		`func main(){a:=struct{}{};_=1<a}`,
		`type S struct{};func get()S{return S{}};func main(){_=get()<get()}`,
		`type S struct{};type T S;func f(a S){_=T(a)<T(a)}`,
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		if result := CheckGraphCore(graph); result.Ok || result.Error != CheckErrOperand {
			t.Fatalf("%s: ok=%v error=%d", source, result.Ok, result.Error)
		}
	}
}

func TestStructOrderingControls(t *testing.T) {
	for _, source := range []string{
		`type S struct{x int};func f(a,b S){_=a==b;_=a!=b;_=a.x<b.x}`,
		`type S struct{};func f(a S){{a:=1;_=a<2}}`,
		`type S struct{};func f(a S){_=func(a int)bool{return a<2}}`,
		`func main(){_=struct{x int}{x:1}.x<2}`,
		`func main(){_=2>struct{x int}{x:1}.x}`,
		`type S struct{};func main(){type S int;var a S;_=a<2}`,
		`type S struct{};func main(){S:=func(n int)int{return n};_=S(1)<2}`,
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		if result := CheckGraphCore(graph); !result.Ok {
			t.Fatalf("%s: error=%d", source, result.Error)
		}
	}
}
