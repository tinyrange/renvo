package link

import (
	"bytes"
	"testing"

	"renvo.dev/internal/load"
)

func TestInferredGlobalCallbackRepresentation(t *testing.T) {
	built := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main
type Holder struct { Callback func() }
var calls int
var Callback = func() { calls++ }
var holder = &Holder{}
func replace() { calls += 10 }
func main() {
 holder.Callback = Callback
 holder.Callback()
 Callback = replace
 Callback()
 Callback = nil
 if Callback != nil { panic("nil") }
}
`)},
	})
	linked := LinkBuildCore(built)
	if !linked.Ok {
		t.Fatal("link failed")
	}
	for _, want := range []string{"var Callback = __renvo_function_0{", "__renvo_call_0(Callback)", "Callback = __renvo_function_0{", "Callback.kind != 0"} {
		if !bytes.Contains(linked.Program.Text, []byte(want)) {
			t.Fatalf("missing %q in %s", want, linked.Program.Text)
		}
	}
}
