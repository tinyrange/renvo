package check

import (
	"renvo.dev/internal/load"
	"testing"
)

func TestMapElementFieldAssignment(t *testing.T) {
	for _, source := range []string{
		`func main(){m:=map[int]S{1:{}};m[1].X=2}`,
		`func main(){m:=map[int]S{1:{}};m[1].X+=2}`,
		`func main(){m:=map[int]S{1:{}};m[1].X++}`,
		`func main(){m:=map[int]S{1:{}};(m[1]).X=2}`,
		`func main(){m:=map[int]S{1:{}};_,m[1].X=1,2}`,
		`func f(m map[int]S){m[1].X=2}`,
		`var m map[int]S; func main(){m[1].X=2}`,
		`type M map[int]S; type N = M; func main(){m:=make(N);m[1].X=2}`,
		`func main(){m:=map[int]struct{V S}{};m[1].V.X=2}`,
		`func main(){m:=map[int]struct{P *S}{};m[1].P=nil}`,
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\ntype S struct{X int}\n" + source)}})
		if result := CheckGraphCore(graph); result.Ok || result.Error != CheckErrAssignTarget {
			t.Fatalf("%s: ok=%v error=%d token=%d", source, result.Ok, result.Error, result.ErrorToken)
		}
	}
}

func TestMapElementFieldAssignmentControls(t *testing.T) {
	for _, source := range []string{
		`func main(){m:=map[int]*S{1:&S{}};m[1].X=2}`,
		`func main(){m:=map[int]*S{1:{}};m[1].X=2}`,
		`func main(){m:=[]*S{{}};m[0].X=2}`,
		`func main(){m:=map[int]struct{P *S}{};m[1].P.X=2}`,
		`func main(){m:=map[int]struct{*S}{};m[1].X=2}`,
		`func main(){m:=map[int]S{1:{}};m[1]=S{X:2};_=m[1].X}`,
		`func main(){m:=map[int]S{1:{}};s:=m[1];s.X=2}`,
		`func main(){m:=[]S{{}};m[0].X=2}`,
		`func f(m map[int]S){{m:=map[int]*S{1:&S{}};m[1].X=2};_=m[1].X}`,
		`type M map[int]S;func main(){type M map[int]*S;m:=M{1:&S{}};m[1].X=2}`,
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\ntype S struct{X int}\n" + source)}})
		if result := CheckGraphCore(graph); !result.Ok {
			t.Fatalf("%s: error=%d token=%d", source, result.Error, result.ErrorToken)
		}
	}
}
