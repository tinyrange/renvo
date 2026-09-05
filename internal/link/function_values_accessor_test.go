package link

import (
	"bytes"
	"testing"

	"renvo.dev/internal/load"
)

func TestFunctionValuePointerAccessorAssignmentIsLowered(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

type holder struct { value func(int) int }

func (p *holder) callback() *func(int) int {
	return &p.value
}

func increment(value int) int { return value + 1 }

func main() {
	var value holder
	(*value.callback()) = nil
	(*value.callback()) = increment
	print("invoke")
	if (*value.callback()) != nil && (*value.callback())(2) == 3 { print("PASS\n") }
}
`)},
	})
	linked := LinkBuildCore(result)
	if !linked.Ok {
		t.Fatalf("LinkBuildCore failed: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	if !bytes.Contains(linked.Program.Text, []byte("(*value.callback()) = __renvo_function_0{}")) ||
		!bytes.Contains(linked.Program.Text, []byte("(*value.callback()) = __renvo_function_0{kind: 1}")) ||
		!bytes.Contains(linked.Program.Text, []byte("(*value.callback()) .kind!= 0")) ||
		!bytes.Contains(linked.Program.Text, []byte("__renvo_call_0((*value.callback()), 2)")) {
		t.Fatalf("function pointer accessor was not lowered:\n%s", linked.Program.Text)
	}
}
