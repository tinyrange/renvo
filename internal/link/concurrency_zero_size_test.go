package link

import (
	"bytes"
	"testing"

	"renvo.dev/internal/load"
)

func TestLinkLowersZeroSizeChannelElement(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main
func renvo_runtime_ChanCreate(size uintptr, capacity int) uintptr { return 1 }
func main() { _ = make(chan struct{}) }
`)},
	})
	linked := LinkBuildCore(result)
	if !linked.Ok {
		t.Fatalf("link failed: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	if !bytes.Contains(linked.Program.Text, []byte("renvo_runtime_ChanCreate(Sizeof(*new(byte)), 0)")) {
		t.Fatalf("zero-size channel was not assigned runtime storage: %s", linked.Program.Text)
	}
}
