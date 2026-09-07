package link

import (
	"bytes"
	"renvo.dev/internal/load"
	"testing"
)

func TestInterfaceMethodExpressionLowering(t *testing.T) {
	built := buildFromFiles(t, []load.SourceFile{{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")}, {Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\ntype I interface { M(int) int }; type T int; func(t T) M(x int) int {return int(t)+x}; func main(){f:=I.M;println(f(T(2),3))}")}})
	linked := LinkBuildCore(built)
	if !linked.Ok {
		t.Fatal("link failed")
	}
	if bytes.Contains(linked.Program.Text, []byte("I.M")) {
		t.Fatalf("method expression was not lowered: %s", linked.Program.Text)
	}
}

func TestInterfaceNamedFieldMethodCall(t *testing.T) {
	built := buildFromFiles(t, []load.SourceFile{{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")}, {Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\ntype Value interface { Set(int) }; type Item struct { Value Value }; type T struct { n int }; func(t *T) Set(x int) {t.n=x}; func main(){t:=&T{}; item:=Item{Value:t}; item.Value.Set(3); if t.n!=3 {panic(\"setter\")}}")}})
	linked := LinkBuildCore(built)
	if !linked.Ok {
		t.Fatal("link failed")
	}
	if bytes.Contains(linked.Program.Text, []byte("item.__renvo_interface_expression_")) {
		t.Fatalf("field call mistaken for type method expression: %s", linked.Program.Text)
	}
}
