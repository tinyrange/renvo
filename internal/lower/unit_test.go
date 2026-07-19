package lower

import (
	"bytes"
	"testing"

	"renvo.dev/internal/check"
	"renvo.dev/internal/load"
	"renvo.dev/internal/unit"
)

func TestEmitCheckedPackageCorePreservesCanonicalModel(t *testing.T) {
	graph := lowerTestGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import "example.com/case/pkg/lib"

const answer = 42
func appMain() int { return answer + lib.Value() }
`)},
		{Path: "/repo/case/pkg/lib/lib.go", Src: []byte(`package lib

func Value() int { return 1 }
`)},
	})
	checked := check.CheckGraphCore(graph)
	if !checked.Ok {
		t.Fatalf("CheckGraphCore failed: err=%d pkg=%d file=%d tok=%d", checked.Error, checked.ErrorPackage, checked.ErrorFile, checked.ErrorToken)
	}
	root := lowerRootPackage(t, graph)
	result := EmitCheckedPackageCore(graph.Packages[root], checked.Packages[root], false)
	if !result.Ok {
		t.Fatalf("EmitCheckedPackageCore failed: err=%d file=%d tok=%d", result.Error, result.ErrorFile, result.ErrorToken)
	}
	program := result.Program
	if program.Package != "main" || program.ImportPath != "example.com/case/cmd/app" {
		t.Fatalf("program identity = %q %q", program.Package, program.ImportPath)
	}
	if len(program.Imports) != 1 || lowerTokenText(program, program.Imports[0].PathTok) != `"example.com/case/pkg/lib"` {
		t.Fatalf("imports = %#v", program.Imports)
	}
	if len(program.Decls) != 1 || lowerSpanText(program, program.Decls[0].NameStart, program.Decls[0].NameEnd) != "answer" {
		t.Fatalf("declarations = %#v", program.Decls)
	}
	if len(program.Funcs) != 1 || lowerSpanText(program, program.Funcs[0].NameStart, program.Funcs[0].NameEnd) != "appMain" {
		t.Fatalf("functions = %#v", program.Funcs)
	}
	if len(program.Symbols) != 2 {
		t.Fatalf("symbols = %#v", program.Symbols)
	}
	if len(program.Tokens) == 0 || program.Tokens[len(program.Tokens)-1].Kind != unit.TokenEOF {
		t.Fatal("canonical unit has no final EOF token")
	}
	if len(program.Selectors) == 0 || lowerTokenText(program, program.Selectors[0].NameTok) != "Value" {
		t.Fatalf("selectors = %#v", program.Selectors)
	}
}

func TestEmitCheckedPackageCorePreservesFileOrderAndBoundaries(t *testing.T) {
	graph := lowerTestGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/z.go", Src: []byte("package main\nfunc zed() int { return 2 }\n")},
		{Path: "/repo/case/cmd/app/a.go", Src: []byte("package main\nfunc alpha() int { return 1 }")},
	})
	checked := check.CheckGraphCore(graph)
	if !checked.Ok {
		t.Fatalf("CheckGraphCore failed: err=%d", checked.Error)
	}
	result := EmitCheckedPackageCore(graph.Packages[0], checked.Packages[0], false)
	if !result.Ok {
		t.Fatalf("EmitCheckedPackageCore failed: err=%d", result.Error)
	}
	program := result.Program
	alpha := bytes.Index(program.Text, []byte("func alpha"))
	zed := bytes.Index(program.Text, []byte("func zed"))
	if alpha < 0 || zed <= alpha {
		t.Fatalf("file order was not preserved: %q", program.Text)
	}
	if len(program.Funcs) != 2 {
		t.Fatalf("function count = %d, want 2", len(program.Funcs))
	}
	for i := 0; i+1 < len(program.Tokens); i++ {
		if program.Tokens[i].Kind == unit.TokenEOF {
			t.Fatalf("file-local EOF leaked into token %d", i)
		}
	}
	if got := program.Tokens[len(program.Tokens)-1].Start; got != len(program.Text) {
		t.Fatalf("EOF start = %d, want %d", got, len(program.Text))
	}
}

func TestEmitCheckedPackageCoreRejectsInvalidInputs(t *testing.T) {
	if result := EmitCheckedPackageCore(load.Package{}, check.PackageInfo{}, false); result.Ok || result.Error != EmitErrPackage {
		t.Fatalf("empty package result = %#v", result)
	}
	graph := lowerTestGraph(t, []load.SourceFile{{
		Path: "/repo/case/cmd/app/main.go",
		Src:  []byte("package main\nfunc appMain() {}\n"),
	}})
	checked := check.CheckGraphCore(graph)
	if !checked.Ok {
		t.Fatalf("CheckGraphCore failed: %d", checked.Error)
	}
	info := checked.Packages[0]
	info.Name = "wrong"
	if result := EmitCheckedPackageCore(graph.Packages[0], info, false); result.Ok || result.Error != EmitErrCheck {
		t.Fatalf("mismatched checker result = %#v", result)
	}
}

func lowerTestGraph(t *testing.T, files []load.SourceFile) load.Graph {
	t.Helper()
	module := load.Module{Root: "/repo/case", Path: "example.com/case", Ok: true}
	graph := load.LoadGraph(module, "/std", "/repo/case", "./cmd/app", files)
	if !graph.Ok {
		t.Fatalf("LoadGraph failed: err=%d pkg=%d", graph.Error, graph.ErrorPackage)
	}
	return graph
}

func lowerRootPackage(t *testing.T, graph load.Graph) int {
	t.Helper()
	for i := 0; i < len(graph.Packages); i++ {
		if graph.Packages[i].Ref.ImportPath == graph.Root {
			return i
		}
	}
	t.Fatal("root package not found")
	return -1
}

func lowerTokenText(program unit.Program, index int) string {
	if index < 0 || index >= len(program.Tokens) {
		return ""
	}
	tok := program.Tokens[index]
	return lowerSpanText(program, tok.Start, tok.Start+tok.Size)
}

func lowerSpanText(program unit.Program, start int, end int) string {
	if start < 0 || end < start || end > len(program.Text) {
		return ""
	}
	return string(program.Text[start:end])
}
