package link

import (
	"bytes"
	"testing"

	"renvo.dev/internal/load"
)

func TestIncrementalDefaultHandlerMatchesWholeProgram(t *testing.T) {
	built := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module renvo.dev/x/runtime\n")},
		{Path: "/repo/case/runtime.go", Src: []byte(`package runtime

type Handler struct { Value int }
var activeHandler *Handler
func requireHandler() *Handler { panic("missing handler") }
func Value() int { return requireHandler().Value }
`)},
		{Path: "/repo/case/serial/serial.go", Src: []byte(`package serial
import "renvo.dev/x/runtime"
func New() *runtime.Handler { return &runtime.Handler{Value: 7} }
`)},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main
import "renvo.dev/x/runtime"
import _ "renvo.dev/x/runtime/serial"
func main() { print(runtime.Value()) }
`)},
	})
	want := LinkBuildCore(built)
	if !want.Ok || bytes.Contains(want.Program.Text, []byte("missing handler")) {
		t.Fatal("whole-program linker did not install the default handler")
	}
	for _, transient := range []bool{false, false, true} {
		session := BeginPackageSession(built, transient)
		for !session.Step() {
		}
		got := session.Result()
		if !got.Ok || !bytes.Equal(got.Data, want.Data) {
			t.Fatalf("incremental default handler differs (transient=%v): ok=%v", transient, got.Ok)
		}
	}
}
