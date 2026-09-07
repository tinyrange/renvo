package link

import (
	"renvo.dev/internal/load"
	"testing"
)

func TestLocalTypeLookupUsesActiveDeclaration(t *testing.T) {
	built := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main
func probe(x any) {}
func main() {
 m := map[int]int{}
 probe(m)
 { m := map[string]int{}; probe(m) }
 probe(m)
 if m := map[string]int{}; true { probe(m) } else { probe(m) }
 probe(m)
 for m := map[string]int{}; false; { probe(m) }
 probe(m)
}
`)},
	})
	program := &built.Units[built.Root].Program
	want := []string{"map[int]int", "map[string]int", "map[int]int", "map[string]int", "map[string]int", "map[int]int", "map[string]int", "map[int]int"}
	count := 0
	for i := 0; i+3 < len(program.Tokens); i++ {
		if !functionValueTokenEquals(program, i, "probe") || !functionValueTokenEquals(program, i+2, "m") {
			continue
		}
		got := compactMapLowerType(functionValueEnclosingLocalType(program, i, "m"))
		if count >= len(want) || got != want[count] {
			t.Fatalf("probe %d: got %q", count, got)
		}
		count++
	}
	if count != len(want) {
		t.Fatalf("found %d probes, want %d", count, len(want))
	}
}
