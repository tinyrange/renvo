package check

import (
	"testing"

	"j5.nz/rtg/rtg/internal/load"
)

func TestCheckGraphSymbolsAndImports(t *testing.T) {
	graph := testGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import (
	"example.com/case/pkg/lib"
	helper "example.com/case/pkg/helper"
	_ "runtime"
)

const answer = 42
var left, right int
type item struct { value int }

func run() {}
func (i item) Score() int { return i.value }
`)},
		{Path: "/repo/case/pkg/lib/lib.go", Src: []byte(`package core

func Value() int { return 1 }
`)},
		{Path: "/repo/case/pkg/helper/helper.go", Src: []byte(`package helper

func Value() int { return 2 }
`)},
		{Path: "/std/runtime/runtime.go", Src: []byte(`package runtime

func KeepAlive(v int) {}
`)},
	})
	prog := CheckGraph(graph)
	if !prog.Ok {
		t.Fatalf("CheckGraph failed: err=%d pkg=%d file=%d tok=%d", prog.Error, prog.ErrorPackage, prog.ErrorFile, prog.ErrorToken)
	}
	root := prog.Packages[len(prog.Packages)-1]
	assertSymbol(t, root, "answer", SymbolConst)
	assertSymbol(t, root, "left", SymbolVar)
	assertSymbol(t, root, "right", SymbolVar)
	assertSymbol(t, root, "item", SymbolType)
	assertSymbol(t, root, "run", SymbolFunc)
	assertSymbol(t, root, "item.Score", SymbolMethod)

	core := LookupImport(root, 0, "core")
	if core < 0 || root.Imports[core].ImportPath != "example.com/case/pkg/lib" {
		t.Fatalf("default package import = %#v", root.Imports)
	}
	helper := LookupImport(root, 0, "helper")
	if helper < 0 || root.Imports[helper].ImportPath != "example.com/case/pkg/helper" {
		t.Fatalf("aliased package import = %#v", root.Imports)
	}
	foundBlank := false
	for i := 0; i < len(root.Imports); i++ {
		if root.Imports[i].Blank && root.Imports[i].ImportPath == "runtime" {
			foundBlank = true
		}
	}
	if !foundBlank {
		t.Fatalf("blank runtime import not found: %#v", root.Imports)
	}
}

func TestCheckGraphDuplicateSymbols(t *testing.T) {
	graph := testGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/a.go", Src: []byte(`package main

var value int
`)},
		{Path: "/repo/case/cmd/app/b.go", Src: []byte(`package main

func value() {}
`)},
	})
	prog := CheckGraph(graph)
	if prog.Ok || prog.Error != CheckErrDuplicate || prog.ErrorPackage != 0 || prog.ErrorFile != 1 {
		t.Fatalf("duplicate symbol check = %#v", prog)
	}
}

func TestCheckGraphMethodDuplicates(t *testing.T) {
	okGraph := testGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

type one struct{}
type two struct{}

func (v one) Value() int { return 1 }
func (v two) Value() int { return 2 }
`)},
	})
	okProg := CheckGraph(okGraph)
	if !okProg.Ok {
		t.Fatalf("methods on different receivers failed: %#v", okProg)
	}

	dupGraph := testGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

type item struct{}

func (v item) Value() int { return 1 }
func (v *item) Value() int { return 2 }
`)},
	})
	dupProg := CheckGraph(dupGraph)
	if dupProg.Ok || dupProg.Error != CheckErrDuplicate {
		t.Fatalf("duplicate method check = %#v", dupProg)
	}
}

func TestCheckGraphDuplicateImportNames(t *testing.T) {
	graph := testGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import (
	"example.com/case/pkg/left"
	"example.com/case/pkg/right"
)
`)},
		{Path: "/repo/case/pkg/left/left.go", Src: []byte(`package same
`)},
		{Path: "/repo/case/pkg/right/right.go", Src: []byte(`package same
`)},
	})
	prog := CheckGraph(graph)
	if prog.Ok || prog.Error != CheckErrDuplicate {
		t.Fatalf("duplicate import check = %#v", prog)
	}
}

func TestCheckGraphImportForms(t *testing.T) {
	graph := testGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import (
	. "example.com/case/pkg/dot"
	_ "example.com/case/pkg/blank"
)
`)},
		{Path: "/repo/case/pkg/dot/dot.go", Src: []byte(`package dot
`)},
		{Path: "/repo/case/pkg/blank/blank.go", Src: []byte(`package blank
`)},
	})
	prog := CheckGraph(graph)
	if !prog.Ok {
		t.Fatalf("CheckGraph failed: %#v", prog)
	}
	root := prog.Packages[len(prog.Packages)-1]
	if len(root.Imports) != 2 {
		t.Fatalf("import count = %d, want 2", len(root.Imports))
	}
	foundDot := false
	foundBlank := false
	for i := 0; i < len(root.Imports); i++ {
		foundDot = foundDot || root.Imports[i].Dot
		foundBlank = foundBlank || root.Imports[i].Blank
	}
	if !foundDot || !foundBlank {
		t.Fatalf("dot/blank imports = %#v", root.Imports)
	}
}

func testGraph(t *testing.T, files []load.SourceFile) load.Graph {
	t.Helper()
	mod := load.Module{Root: "/repo/case", Path: "example.com/case", Ok: true}
	graph := load.LoadGraph(mod, "/std", "/repo/case", "./cmd/app", files)
	if !graph.Ok {
		t.Fatalf("LoadGraph failed: err=%d pkg=%d graph=%#v", graph.Error, graph.ErrorPackage, graph)
	}
	return graph
}

func assertSymbol(t *testing.T, info PackageInfo, name string, kind int) {
	t.Helper()
	index := LookupPackageSymbol(info, name)
	if index < 0 {
		t.Fatalf("symbol %q not found in %#v", name, info.Symbols)
	}
	if info.Symbols[index].Kind != kind {
		t.Fatalf("symbol %q kind = %d, want %d", name, info.Symbols[index].Kind, kind)
	}
}
