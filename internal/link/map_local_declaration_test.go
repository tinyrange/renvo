package link

import (
	"testing"

	"renvo.dev/internal/load"
)

func TestMapBuiltinImmediatelyAfterNamedDeclaration(t *testing.T) {
	built := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\ntype M map[int]int\nfunc main(){\nvar empty M\ndelete(empty,1)\n}\n")},
	})
	program := &built.Units[built.Root].Program
	for i := range program.Tokens {
		if functionValueTokenEquals(program, i, "delete") {
			if typ := functionValueEnclosingLocalType(program, i, "empty"); typ != "M" {
				t.Fatalf("map type=%q, want M", typ)
			}
			return
		}
	}
	t.Fatal("missing delete call")
}
