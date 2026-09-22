package link

import (
	"bytes"
	"testing"

	"renvo.dev/internal/load"
)

func TestClosureLocalShadowing(t *testing.T) {
	built := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main
type Callback func() int
func main() {
 err := 10
 callback := Callback(func() int {
  total := err
  if err := err + 1; err != 11 { panic("init") } else { total += err }
  total += err
  return total
 })
 println(callback())
}
`)},
	})
	linked := LinkBuildCore(built)
	if !linked.Ok {
		t.Fatal("link failed")
	}
	for _, want := range []string{"if err := (*env.err) + 1; err != 11", "else { total += err }", "total += (*env.err)"} {
		if !bytes.Contains(linked.Program.Text, []byte(want)) {
			t.Fatalf("missing %q: %s", want, linked.Program.Text)
		}
	}
}
