package link

import (
	"bytes"
	"testing"

	"renvo.dev/internal/load"
)

func TestRecursiveAnonymousClosureLowering(t *testing.T) {
	built := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main
func main() {
 total := 0
 var visit func(int)
 visit = func(n int) {
  total += n
  if n > 0 { visit(n-1) }
 }
 visit(4)
 println(total)
}

`)},
	})
	linked := LinkBuildCore(built)
	if !linked.Ok {
		t.Fatal("link failed")
	}
	for _, want := range []string{"var visit __renvo_function_0", "visit *__renvo_function_0", "__renvo_call_0((*env.visit), n-1)"} {
		if !bytes.Contains(linked.Program.Text, []byte(want)) {
			t.Fatalf("missing %q: %s", want, linked.Program.Text)
		}
	}
}
