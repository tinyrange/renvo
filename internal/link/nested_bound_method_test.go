package link

import (
	"bytes"
	"renvo.dev/internal/load"
	"testing"
)

func TestNestedFieldBoundMethodValue(t *testing.T) {
	built := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main
type Inner struct{N int}
func(i *Inner) Read(x int)int{return i.N+x}
type Outer struct{ Child *Inner }
func invoke(f func(int)int)int{return f(3)}
func value(o *Outer)int{ f:=o.Child.Read; return invoke(o.Child.Read)+f(4) }
func main(){println(value(&Outer{Child:&Inner{N:7}}))}
`)}})
	linked := LinkBuildCore(built)
	if !linked.Ok {
		t.Fatal(linked.Error)
	}
	if bytes.Contains(linked.Program.Text, []byte("invoke(o.Child.Read)")) || bytes.Contains(linked.Program.Text, []byte("f:=o.Child.Read")) {
		t.Fatalf("unlowered field method value: %s", linked.Program.Text)
	}
}
