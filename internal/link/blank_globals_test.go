package link

import (
	"bytes"
	"renvo.dev/internal/load"
	"testing"
)

func TestBlankGlobalsRetainSingleInitializerEvaluation(t *testing.T) {
	built := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main
var count int
func touch(n int) int {count=count*10+n;return n}
var _=touch(1)
var _=touch(2)
var _,kept=touch(3),touch(4)
var _=touch
func main(){if kept!=4 {panic("kept")}}
`)},
	})
	linked := LinkBuildCore(built)
	if !linked.Ok {
		t.Fatalf("blank global link failed: %d", linked.Error)
	}
	if bytes.Count(linked.Program.Text, []byte("touch(3)")) != 1 || bytes.Count(linked.Program.Text, []byte("touch(4)")) != 1 {
		t.Fatalf("initializer evaluation duplicated:\n%s", linked.Program.Text)
	}
}
