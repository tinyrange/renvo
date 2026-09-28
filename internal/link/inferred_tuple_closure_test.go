package link

import (
	"bytes"
	"os"
	"renvo.dev/internal/load"
	"testing"
)

func TestInferredTupleClosureLowering(t *testing.T) {
	source, err := os.ReadFile("../../frontend_tests/regressions/inferred_tuple_closure/cmd/app/main.go")
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
	for _, want := range []string{"base *int", "calls *int", "convert := __renvo_function_"} {
		if !bytes.Contains(linked.Program.Text, []byte(want)) {
			t.Fatalf("missing %q: %s", want, linked.Program.Text)
		}
	}
	if bytes.Contains(linked.Program.Text, []byte("convert(i)")) {
		t.Fatalf("unlowered callback call: %s", linked.Program.Text)
	}
}
