package link

import (
	"bytes"
	"renvo.dev/internal/load"
	"testing"
)

func TestConvertedCallbackLocalCall(t *testing.T) {
	built := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main
type Callback func(int) int
func main() {
 n := 3
 f := Callback(func(x int) int { return x+n })
 if f(4) != 7 { panic("converted closure") }
}
`)},
	})
	linked := LinkBuildCore(built)
	if !linked.Ok || !bytes.Contains(linked.Program.Text, []byte("__renvo_call_0(f, 4)")) {
		t.Fatalf("converted callback call not lowered: %s", linked.Program.Text)
	}
}
