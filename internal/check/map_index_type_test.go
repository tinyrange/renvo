package check

import (
	"renvo.dev/internal/load"
	"testing"
)

func TestMapIndexPrimitiveTypes(t *testing.T) {
	for _, source := range []string{
		`func main(){ m:=map[int]int{}; _=m["x"] }`,
		`func main(){ m:=make(map[int]int); _=m["x"] }`,
		`func f(m map[string]int){ _=m[1] }`,
		`func main(){ var m map[int]int; m["x"]=1 }`,
		`func main(){ var (m map[int]int); _=m["x"] }`,
		`type Key int; type M map[Key]int; func main(){m:=M{}; _=m["x"]}`,
		`var m=map[int]int{}; func main(){_=m["x"]}`,
		`func main(){ m:=map[int]int{}; n:=m; _=n["x"] }`,
		`func main(){ m:=map[int]int{}; {m:=map[string]int{}; _=m["x"]}; _=m["x"] }`,
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		if result := CheckGraphCore(graph); result.Ok || result.Error != CheckErrType {
			t.Fatalf("%s: %+v", source, result)
		}
	}
}

func TestMapIndexPrimitiveControls(t *testing.T) {
	for _, source := range []string{
		`func main(){m:=map[int]int{}; _=m[1]}`,
		`func main(){m:=map[any]int{}; _=m["x"]}`,
		`func f(m map[int]int){ {m:=map[string]int{}; _=m["x"]}; _=m[1] }`,
		`var m map[int]int; func main(){m:=map[string]int{}; _=m["x"]}`,
		`func main(){m:=map[int]int{}; {m:=map[string]int{}; _=m["x"]}; _=m[1]}`,
		`func main(){m:=map[string]int{}; if m:=map[int]int{}; m[1]==0 { _=m[1] }; _=m["x"]}`,
		`func main(){m:=map[int]int{}; {m:=[]int{1}; _=m[0]}; _=m[1]}`,
		`func main(){type int string; m:=map[int]int{}; _=m["x"]}`,
		`func main(){m:=map[int]int{}; _=m[1.0]}`,
		`func main(){m:=map[int]int{}; _=m["x"[0]]}`,
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		if result := CheckGraphCore(graph); !result.Ok {
			t.Fatalf("%s: %+v", source, result)
		}
	}
}
