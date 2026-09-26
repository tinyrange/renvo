package check

import (
	"testing"

	"renvo.dev/internal/load"
)

func TestMissingStructSelector(t *testing.T) {
	for _, source := range []string{
		`type S struct{};func main(){_=S{}.X}`,
		`func main(){_=struct{Y int}{1}.X}`,
		`type S struct{};func f(s S){_=s.X}`,
		`type S struct{};func f(s *S){_=s.X}`,
		`type S struct{};type A=S;func main(){s:=A{};_=s.X}`,
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		if result := CheckGraphCore(graph); result.Ok || result.Error != CheckErrUndefined {
			t.Fatalf("%s: ok=%v error=%d", source, result.Ok, result.Error)
		}
	}
}

func TestStructSelectorControls(t *testing.T) {
	for _, source := range []string{
		`type S struct{X int};func main(){_=S{1}.X}`,
		`type S struct{};func(S) X()int{return 1};func main(){_=S{}.X()}`,
		`type S struct{};func(*S) X()int{return 1};func f(s *S){_=s.X()}`,
		`type S struct{X int};type T struct{S};func main(){_=T{}.X}`,
		`type S struct{};func f(s S){{s:=struct{X int}{1};_=s.X}}`,
		`type S struct{};func f(s S){_=func(s struct{X int}){_=s.X}}`,
		`type S struct{};func main(){type S struct{X int};_=S{1}.X}`,
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		if result := CheckGraphCore(graph); !result.Ok {
			t.Fatalf("%s: error=%d", source, result.Error)
		}
	}
}
