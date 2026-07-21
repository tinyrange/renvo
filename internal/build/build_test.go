package build

import (
	"bytes"
	"testing"

	wireunit "renvo.dev/backend/unit"
	"renvo.dev/internal/check"
	"renvo.dev/internal/load"
	"renvo.dev/internal/unit"
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
		decoded, err := wireunit.Unmarshal(marshalPackageUnit(t, result.Units[i]))
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
	decoded, err := wireunit.Unmarshal(marshalPackageUnit(t, RootUnit(result)))
	if err != nil {
		t.Fatalf("root unit decode failed: %v", err)
	}
	if len(decoded.Decls) != 1 || len(decoded.Funcs) != 1 {
		t.Fatalf("decoded unit decls/funcs = %d/%d", len(decoded.Decls), len(decoded.Funcs))
	}
}

func TestTransientPackageCacheReusesDependenciesAndRebuildsRoot(t *testing.T) {
	packageProgramCacheUsed = nil
	packageProgramCacheData = nil
	InitializePackageProgramCache()
	packageProgramCacheNext = 0
	packageProgramCacheHits = 0
	packageProgramCacheMisses = 0
	files := []load.SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n\nimport \"example.com/case/pkg/lib\"\n\nfunc appMain() int { return lib.Value() }\n")},
		{Path: "/repo/case/pkg/lib/lib.go", Src: []byte("package lib\n\ntype Box struct { Value int }\n\nfunc Value() int { return 42 }\n")},
	}
	first := BuildProgramsTransientCached(buildTestGraph(t, files))
	if !first.Ok || packageProgramCacheHits != 0 || packageProgramCacheMisses != 1 {
		t.Fatalf("cold cache result = %#v, hits/misses = %d/%d", first, packageProgramCacheHits, packageProgramCacheMisses)
	}
	files[0].Src = []byte("package main\n\nimport \"example.com/case/pkg/lib\"\n\nfunc appMain() int { var box lib.Box; return box.Value + 1 }\n")
	second := BuildProgramsTransientCached(buildTestGraph(t, files))
	if !second.Ok || packageProgramCacheHits != 1 || packageProgramCacheMisses != 1 {
		t.Fatalf("warm cache result = %#v, hits/misses = %d/%d", second, packageProgramCacheHits, packageProgramCacheMisses)
	}
	if !bytes.Contains(second.Units[second.Root].Program.Text, []byte("+ 1")) {
		t.Fatal("cached build did not rebuild changed root package")
	}
	files[1].Src = []byte("package lib\n\ntype Box struct { Value int }\n\nfunc Value() int { return 43 }\n")
	third := BuildProgramsTransientCached(buildTestGraph(t, files))
	if !third.Ok || packageProgramCacheHits != 1 || packageProgramCacheMisses != 2 {
		t.Fatalf("invalidated cache result = %#v, hits/misses = %d/%d", third, packageProgramCacheHits, packageProgramCacheMisses)
	}
	if !bytes.Contains(third.Units[0].Program.Text, []byte("43")) {
		t.Fatal("cached build reused a changed dependency package")
	}
}

func TestTransientPackageCacheInvalidatesImportersWhenDependencyChanges(t *testing.T) {
	packageProgramCacheUsed = nil
	packageProgramCacheData = nil
	InitializePackageProgramCache()
	packageProgramCacheNext = 0
	packageProgramCacheHits = 0
	packageProgramCacheMisses = 0
	files := []load.SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n\nimport \"example.com/case/pkg/mid\"\n\nfunc appMain() int { return mid.Use() }\n")},
		{Path: "/repo/case/pkg/mid/mid.go", Src: []byte("package mid\n\nimport \"example.com/case/pkg/lib\"\n\nfunc Use() int { return lib.Value() }\n")},
		{Path: "/repo/case/pkg/lib/lib.go", Src: []byte("package lib\n\nfunc Value() int { return 42 }\n")},
	}
	first := BuildProgramsTransientCached(buildTestGraph(t, files))
	if !first.Ok || packageProgramCacheHits != 0 || packageProgramCacheMisses != 2 {
		t.Fatalf("cold cache result = %#v, hits/misses = %d/%d", first, packageProgramCacheHits, packageProgramCacheMisses)
	}
	files[2].Src = []byte("package lib\n\nfunc Value() int { return 43 }\n")
	changed := BuildProgramsTransientCached(buildTestGraph(t, files))
	if !changed.Ok {
		t.Fatalf("dependency rebuild failed: %#v", changed)
	}
	if packageProgramCacheHits != 0 || packageProgramCacheMisses != 4 {
		t.Fatalf("dependency change hits/misses = %d/%d, want 0/4", packageProgramCacheHits, packageProgramCacheMisses)
	}
}

func TestPackageSourceHashIsIndependentOfPackageRoot(t *testing.T) {
	source := []byte("package lib\nfunc Value() int { return 42 }\n")
	left := load.Package{
		Ref:   load.PackageRef{Dir: "/first/module/pkg/lib"},
		Files: []load.ParsedFile{{Path: "/first/module/pkg/lib/lib.go", Src: source}},
	}
	right := load.Package{
		Ref:   load.PackageRef{Dir: "/second/module/pkg/lib"},
		Files: []load.ParsedFile{{Path: "/second/module/pkg/lib/lib.go", Src: source}},
	}
	leftA, leftB := packageSourceHash(left)
	rightA, rightB := packageSourceHash(right)
	if leftA != rightA || leftB != rightB {
		t.Fatalf("relocated package hashes = %d/%d and %d/%d", leftA, leftB, rightA, rightB)
	}
	right.Files[0].Path = "/second/module/pkg/lib/other.go"
	rightA, rightB = packageSourceHash(right)
	if leftA == rightA && leftB == rightB {
		t.Fatal("package hash ignored the relative source filename")
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

func TestBuildUnitsRejectsMissingRootMain(t *testing.T) {
	graph := buildTestGraph(t, []load.SourceFile{{
		Path: "/repo/case/cmd/app/main.go",
		Src:  []byte("package main\nfunc helper() {}\n"),
	}})
	result := BuildUnits(graph)
	if result.Ok || result.Error != BuildErrCheck || result.ErrorDetail != check.CheckErrMissingMain {
		t.Fatalf("missing main result = %#v", result)
	}
}

func TestBuildUnitsAllowsLibraryRootWithoutMain(t *testing.T) {
	module := load.Module{Root: "/repo/case", Path: "example.com/case", Ok: true}
	graph := load.LoadGraph(module, "/std", "/repo/case", "./lib", []load.SourceFile{{
		Path: "/repo/case/lib/lib.go",
		Src:  []byte("package lib\nfunc Value() int { return 42 }\n"),
	}})
	if !graph.Ok {
		t.Fatalf("LoadGraph failed: %#v", graph)
	}
	result := BuildUnits(graph)
	if !result.Ok || len(result.Units) != 1 || result.Units[0].Name != "lib" {
		t.Fatalf("library root result = %#v", result)
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
