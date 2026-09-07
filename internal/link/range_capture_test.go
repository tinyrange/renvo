package link

import (
	"bytes"
	"testing"

	"renvo.dev/internal/load"
)

func TestRangeVariableClosureCapture(t *testing.T) {
	built := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main
type Node struct { N int }
type Callback func() int
func invoke(cb Callback) int { return cb() }
func sum(nodes []*Node) int {
 total := 0
 for _, node := range nodes {
  total += invoke(func() int { return node.N })
 }
 return total
}
func main() { println(sum(nil)) }
`)},
	})
	linked := LinkBuildCore(built)
	if !linked.Ok {
		t.Fatal("link failed")
	}
	for _, want := range []string{"node **Node", "node: &node", "(*env.node).N"} {
		if !bytes.Contains(linked.Program.Text, []byte(want)) {
			t.Fatalf("missing capture %q: %s", want, linked.Program.Text)
		}
	}
}
