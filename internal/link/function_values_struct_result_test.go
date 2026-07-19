package link

import (
	"bytes"
	"testing"

	"renvo.dev/internal/load"
)

func TestFunctionValueThunkUsesNamedZeroStructResult(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

type result struct { ok bool }
type handler func() result
type control struct { callback handler }
type form struct{}

func (f *form) invoke() result { return result{ok: true} }

func main() {
	var f form
	var c control
	c.callback = f.invoke
	if c.callback().ok { print("PASS\n") }
}
`)},
	})
	linked := LinkBuildCore(result)
	if !linked.Ok {
		t.Fatalf("LinkBuildCore failed: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	if !bytes.Contains(linked.Program.Text, []byte("var __renvo_zero result\nreturn __renvo_zero")) {
		t.Fatalf("struct-result callback thunk has no named zero result:\n%s", linked.Program.Text)
	}
}
