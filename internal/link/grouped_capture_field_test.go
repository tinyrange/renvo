package link

import (
	"bytes"
	"renvo.dev/internal/load"
	"testing"
)

func TestCapturedGroupedStructFieldType(t *testing.T) {
	built := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main
type Instruction struct { Width, SourceWidth uint8 }
func value(i Instruction) uint8 { w:=i.Width; s:=i.SourceWidth; f:=func()uint8{return w+s}; return f() }
func main(){ println(value(Instruction{Width:3,SourceWidth:5})) }
`)},
	})
	linked := LinkBuildCore(built)
	if !linked.Ok {
		t.Fatal(linked.Error)
	}
	if bytes.Contains(linked.Program.Text, []byte("*,")) {
		t.Fatalf("grouped field names leaked into capture type: %s", linked.Program.Text)
	}
}
