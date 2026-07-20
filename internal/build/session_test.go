package build

import (
	"testing"

	"renvo.dev/internal/load"
)

func TestProgramSessionYieldsBetweenPackages(t *testing.T) {
	graph := buildTestGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n\nimport \"example.com/case/pkg/lib\"\n\nfunc appMain() int { return lib.Value() }\n")},
		{Path: "/repo/case/pkg/lib/lib.go", Src: []byte("package lib\n\nfunc Value() int { return 42 }\n")},
	})
	session := BeginProgramsSession(graph, false, false)
	steps := 0
	previousUnits := 0
	for {
		done := session.Step()
		steps++
		units := len(session.Result().Units)
		if units-previousUnits > 1 {
			t.Fatalf("step %d built %d packages; want at most one", steps, units-previousUnits)
		}
		previousUnits = units
		if done {
			break
		}
	}
	result := session.Result()
	if !result.Ok || len(result.Units) != 2 || result.Root != 1 {
		t.Fatalf("session result = %#v", result)
	}
	if steps < 4 {
		t.Fatalf("session completed in %d steps; expected header, package, package, and final yields", steps)
	}
}
