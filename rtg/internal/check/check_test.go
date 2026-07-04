package check

import (
	"testing"

	"j5.nz/rtg/rtg/internal/load"
	"j5.nz/rtg/rtg/internal/syntax"
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
	assertBody(t, root, "run", SymbolFunc, 1)
	assertBody(t, root, "item.Score", SymbolMethod, 2)

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

func TestCheckGraphBodyIndexes(t *testing.T) {
	graph := testGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

func appMain(v int) int {
	if v > 0 {
		return v
	}
	return 0
}

func helper() {
	defer cleanup()
}
`)},
	})
	prog := CheckGraph(graph)
	if !prog.Ok {
		t.Fatalf("CheckGraph failed: err=%d pkg=%d file=%d tok=%d", prog.Error, prog.ErrorPackage, prog.ErrorFile, prog.ErrorToken)
	}
	root := prog.Packages[0]
	if len(root.Bodies) != 2 {
		t.Fatalf("body count = %d, want 2: %#v", len(root.Bodies), root.Bodies)
	}
	appBody := LookupFuncBody(root, "appMain")
	if appBody < 0 {
		t.Fatalf("appMain body not found: %#v", root.Bodies)
	}
	if !bodyHasStmt(root.Bodies[appBody].Body, syntax.StmtIf) || !bodyHasStmt(root.Bodies[appBody].Body, syntax.StmtReturn) {
		t.Fatalf("appMain body statements = %#v", root.Bodies[appBody].Body.Stmts)
	}
	helperBody := LookupFuncBody(root, "helper")
	if helperBody < 0 {
		t.Fatalf("helper body not found: %#v", root.Bodies)
	}
	if !bodyHasStmt(root.Bodies[helperBody].Body, syntax.StmtDefer) {
		t.Fatalf("helper body statements = %#v", root.Bodies[helperBody].Body.Stmts)
	}
}

func TestCheckGraphFunctionScopes(t *testing.T) {
	graph := testGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

type item struct{}

func (it item) run(left, right int) (total int) {
	var local int
	const named = 1
	type alias int
	short, other := left, right
start:
	total = local + named + int(short) + other
	return total
}
`)},
	})
	prog := CheckGraph(graph)
	if !prog.Ok {
		t.Fatalf("CheckGraph failed: err=%d pkg=%d file=%d tok=%d", prog.Error, prog.ErrorPackage, prog.ErrorFile, prog.ErrorToken)
	}
	root := prog.Packages[0]
	bodyIndex := LookupFuncBody(root, "item.run")
	if bodyIndex < 0 {
		t.Fatalf("item.run body not found: %#v", root.Bodies)
	}
	scope := root.Bodies[bodyIndex].Scope
	assertScopeName(t, scope, "it", NameReceiver)
	assertScopeName(t, scope, "left", NameParam)
	assertScopeName(t, scope, "right", NameParam)
	assertScopeName(t, scope, "total", NameResult)
	assertScopeName(t, scope, "local", NameLocal)
	assertScopeName(t, scope, "named", NameLocal)
	assertScopeName(t, scope, "alias", NameLocal)
	assertScopeName(t, scope, "short", NameLocal)
	assertScopeName(t, scope, "other", NameLocal)
	assertScopeName(t, scope, "start", NameLabel)
}

func TestCheckGraphFunctionReferences(t *testing.T) {
	graph := testGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import lib "example.com/case/pkg/lib"

const packageValue = 3

func appMain(param int) int {
	local := lib.Value() + packageValue + later() + len("x")
	if local > param {
		goto done
	}
done:
	return local
}

func later() int { return 2 }
`)},
		{Path: "/repo/case/pkg/lib/lib.go", Src: []byte(`package lib

func Value() int { return 4 }
`)},
	})
	prog := CheckGraph(graph)
	if !prog.Ok {
		t.Fatalf("CheckGraph failed: err=%d pkg=%d file=%d tok=%d", prog.Error, prog.ErrorPackage, prog.ErrorFile, prog.ErrorToken)
	}
	root := prog.Packages[len(prog.Packages)-1]
	bodyIndex := LookupFuncBody(root, "appMain")
	if bodyIndex < 0 {
		t.Fatalf("appMain body not found: %#v", root.Bodies)
	}
	body := root.Bodies[bodyIndex]
	assertBodyRef(t, body, "param", RefScope)
	assertBodyRef(t, body, "local", RefScope)
	assertBodyRef(t, body, "packageValue", RefPackage)
	assertBodyRef(t, body, "later", RefPackage)
	assertBodyRef(t, body, "lib", RefImport)
	assertBodyRef(t, body, "len", RefBuiltin)
	assertBodyRef(t, body, "done", RefLabel)
}

func TestCheckGraphDuplicateParamScope(t *testing.T) {
	graph := testGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

func appMain(value int, value string) {}
`)},
	})
	prog := CheckGraph(graph)
	if prog.Ok || prog.Error != CheckErrScope || prog.ErrorPackage != 0 || prog.ErrorFile != 0 {
		t.Fatalf("duplicate param scope check = %#v", prog)
	}
}

func TestCheckGraphDuplicateLabelScope(t *testing.T) {
	graph := testGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

func appMain() {
again:
	goto again
again:
	return
}
`)},
	})
	prog := CheckGraph(graph)
	if prog.Ok || prog.Error != CheckErrScope || prog.ErrorPackage != 0 || prog.ErrorFile != 0 {
		t.Fatalf("duplicate label scope check = %#v", prog)
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

func TestCheckGraphBodyError(t *testing.T) {
	file := syntax.ParseFile([]byte(`package main

func appMain() int {
	return 0
}
`))
	if !file.Ok {
		t.Fatalf("ParseFile failed: err=%d tok=%d", file.Error, file.ErrorTok)
	}
	file.Funcs[0].BodyStart = -1
	graph := load.Graph{
		Root: "example.com/case/cmd/app",
		Ok:   true,
		Packages: []load.Package{{
			Ref:  load.PackageRef{Kind: load.PackageInModule, ImportPath: "example.com/case/cmd/app", Dir: "/repo/case/cmd/app", Ok: true},
			Name: "main",
			Ok:   true,
			Files: []load.ParsedFile{{
				Path: "/repo/case/cmd/app/main.go",
				File: file,
			}},
		}},
	}
	prog := CheckGraph(graph)
	if prog.Ok || prog.Error != CheckErrBody || prog.ErrorPackage != 0 || prog.ErrorFile != 0 {
		t.Fatalf("body error check = %#v", prog)
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

func assertBody(t *testing.T, info PackageInfo, name string, kind int, minStmts int) {
	t.Helper()
	index := LookupFuncBody(info, name)
	if index < 0 {
		t.Fatalf("body %q not found in %#v", name, info.Bodies)
	}
	if info.Bodies[index].Kind != kind {
		t.Fatalf("body %q kind = %d, want %d", name, info.Bodies[index].Kind, kind)
	}
	if !info.Bodies[index].Body.Ok {
		t.Fatalf("body %q parse failed: %#v", name, info.Bodies[index].Body)
	}
	if len(info.Bodies[index].Body.Stmts) < minStmts {
		t.Fatalf("body %q stmt count = %d, want at least %d", name, len(info.Bodies[index].Body.Stmts), minStmts)
	}
}

func assertScopeName(t *testing.T, scope FuncScope, name string, kind int) {
	t.Helper()
	index := LookupScopeName(scope, name)
	if index < 0 {
		t.Fatalf("scope name %q not found in %#v", name, scope.Names)
	}
	if scope.Names[index].Kind != kind {
		t.Fatalf("scope name %q kind = %d, want %d", name, scope.Names[index].Kind, kind)
	}
}

func assertBodyRef(t *testing.T, body FuncBody, name string, kind int) {
	t.Helper()
	index := LookupBodyRef(body, name, kind)
	if index < 0 {
		t.Fatalf("body ref %q kind %d not found in %#v", name, kind, body.Refs)
	}
}

func bodyHasStmt(body syntax.Body, kind int) bool {
	for i := 0; i < len(body.Stmts); i++ {
		if body.Stmts[i].Kind == kind {
			return true
		}
	}
	return false
}
