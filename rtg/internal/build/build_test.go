package build

import (
	"bytes"
	"testing"

	"j5.nz/rtg/rtg/internal/load"
	"j5.nz/rtg/rtg/internal/unit"
	"j5.nz/rtg/rtgunit"
)

func TestBuildUnitsDependencyOrderAndRoot(t *testing.T) {
	graph := buildTestGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import "example.com/case/pkg/lib"

func appMain() int { return lib.Value() }
`)},
		{Path: "/repo/case/pkg/lib/lib.go", Src: []byte(`package lib

func Value() int { return 42 }
`)},
	})
	result := BuildUnits(graph)
	if !result.Ok {
		t.Fatalf("BuildUnits failed: err=%d pkg=%d file=%d tok=%d", result.Error, result.ErrorPackage, result.ErrorFile, result.ErrorToken)
	}
	if len(result.Units) != 2 {
		t.Fatalf("unit count = %d, want 2", len(result.Units))
	}
	if result.Units[0].ImportPath != "example.com/case/pkg/lib" {
		t.Fatalf("unit 0 = %q, want lib", result.Units[0].ImportPath)
	}
	if result.Units[1].ImportPath != "example.com/case/cmd/app" {
		t.Fatalf("unit 1 = %q, want app", result.Units[1].ImportPath)
	}
	if result.Root != 1 {
		t.Fatalf("root = %d, want 1", result.Root)
	}
	root := RootUnit(result)
	if root.ImportPath != "example.com/case/cmd/app" {
		t.Fatalf("RootUnit = %#v", root)
	}
	for i := 0; i < len(result.Units); i++ {
		decoded, err := rtgunit.Unmarshal(marshalPackageUnit(t, result.Units[i]))
		if err != nil {
			t.Fatalf("unit %d did not decode: %v", i, err)
		}
		if decoded.Package != result.Units[i].Name {
			t.Fatalf("unit %d package = %q, want %q", i, decoded.Package, result.Units[i].Name)
		}
		if !bytes.Equal(decoded.Text, result.Units[i].Program.Text) {
			t.Fatalf("unit %d text mismatch", i)
		}
	}
}

func TestBuildUnitsSinglePackage(t *testing.T) {
	graph := buildTestGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

const answer = 42

func appMain() int { return answer }
`)},
	})
	result := BuildUnits(graph)
	if !result.Ok {
		t.Fatalf("BuildUnits failed: err=%d pkg=%d file=%d tok=%d", result.Error, result.ErrorPackage, result.ErrorFile, result.ErrorToken)
	}
	if len(result.Units) != 1 || result.Root != 0 {
		t.Fatalf("units/root = %d/%d", len(result.Units), result.Root)
	}
	decoded, err := rtgunit.Unmarshal(marshalPackageUnit(t, RootUnit(result)))
	if err != nil {
		t.Fatalf("root unit decode failed: %v", err)
	}
	if len(decoded.Decls) != 1 || len(decoded.Funcs) != 1 {
		t.Fatalf("decoded unit decls/funcs = %d/%d", len(decoded.Decls), len(decoded.Funcs))
	}
}

func TestBuildUnitsCheckError(t *testing.T) {
	graph := buildTestGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/a.go", Src: []byte("package main\nvar value int\n")},
		{Path: "/repo/case/cmd/app/b.go", Src: []byte("package main\nfunc value() {}\n")},
	})
	result := BuildUnits(graph)
	if result.Ok {
		t.Fatal("BuildUnits accepted duplicate symbol graph")
	}
	if result.Error != BuildErrCheck {
		t.Fatalf("error = %d, want BuildErrCheck", result.Error)
	}
	if result.ErrorPackage != 0 || result.ErrorFile != 1 {
		t.Fatalf("error location = pkg %d file %d", result.ErrorPackage, result.ErrorFile)
	}
}

func TestRootUnitInvalidResult(t *testing.T) {
	if got := RootUnit(Result{}); got.ImportPath != "" || got.Program.Package != "" {
		t.Fatalf("RootUnit invalid result = %#v", got)
	}
}

func marshalPackageUnit(t *testing.T, pkg PackageUnit) []byte {
	t.Helper()
	data, ok := unit.MarshalCore(unit.CoreProgramFrom(pkg.Program))
	if !ok {
		t.Fatal("MarshalCore failed")
	}
	return data
}

func buildTestGraph(t *testing.T, files []load.SourceFile) load.Graph {
	t.Helper()
	mod := load.Module{Root: "/repo/case", Path: "example.com/case", Ok: true}
	graph := load.LoadGraph(mod, "/std", "/repo/case", "./cmd/app", files)
	if !graph.Ok {
		t.Fatalf("LoadGraph failed: err=%d pkg=%d graph=%#v", graph.Error, graph.ErrorPackage, graph)
	}
	return graph
}
