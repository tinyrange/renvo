package link

import (
	"bytes"
	"renvo.dev/internal/load"
	"testing"
)

func TestMapLiteralRangeLowersHeaderAndConstruction(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main
func main() {
  total := 0
  for key, value := range map[string]string{"one": "x", "two": "yy"} {
    total += len(key) + len(value)
  }
  if total != 9 { panic("range") }
  print("PASS\n")
}
`)},
	})
	linked := LinkBuildCore(result)
	if !linked.Ok {
		t.Fatalf("link failed: %d", linked.Error)
	}
	if !bytes.Contains(linked.Program.Text, []byte("__renvo_map_range_mapping_")) ||
		bytes.Contains(linked.Program.Text, []byte("range __renvo_map_literal_")) {
		t.Fatalf("map literal range was not lowered:\n%s", linked.Program.Text)
	}
}
