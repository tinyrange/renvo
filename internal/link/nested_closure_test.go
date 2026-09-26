package link

import (
	"bytes"
	"os"
	"renvo.dev/internal/load"
	"testing"
)

func TestNestedClosureCaptureLowering(t *testing.T) {
	source, err := os.ReadFile("../../frontend_tests/regressions/nested_closure_capture/cmd/app/main.go")
	if err != nil {
		t.Fatal(err)
	}
	built := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: source},
	})
	linked := LinkBuildCore(built)
	if !linked.Ok {
		t.Fatal("link failed")
	}
	for _, want := range []string{"base *int", "index *int", "local *int"} {
		if !bytes.Contains(linked.Program.Text, []byte(want)) {
			t.Fatalf("missing %q: %s", want, linked.Program.Text)
		}
	}
	if bytes.Contains(linked.Program.Text, []byte("return func(")) {
		t.Fatalf("unlowered literal: %s", linked.Program.Text)
	}
}
