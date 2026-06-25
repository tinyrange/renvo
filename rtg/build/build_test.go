package build

import (
	"testing"

	"j5.nz/rtg/rtg/load"
)

func TestUnitsLowersWholePackageGraph(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Imports:    []string{"example.com/app/answer"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import "example.com/app/answer"

func appMain() int { return answer.Value() }
`),
			},
		},
	}
	depPkg := load.Package{
		ImportPath: "example.com/app/answer",
		Name:       "answer",
		Files: []load.File{
			{
				Path: "answer.go",
				Source: []byte(`package answer

func Value() int { return 7 }
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, depPkg}}
	units, err := Units(graph)
	if err != nil {
		t.Fatalf("Units failed: %v", err)
	}
	if len(units) != 2 {
		t.Fatalf("units = %#v, want 2", units)
	}
	if units[0].ImportPath != "example.com/app" || units[1].ImportPath != "example.com/app/answer" {
		t.Fatalf("unit order = %#v", units)
	}
	if len(units[0].References) != 1 || units[0].References[0].ImportPath != "example.com/app/answer" {
		t.Fatalf("main references = %#v", units[0].References)
	}
	if len(units[1].Exports) != 1 || units[1].Exports[0].Name != "Value" {
		t.Fatalf("dep exports = %#v", units[1].Exports)
	}
}
