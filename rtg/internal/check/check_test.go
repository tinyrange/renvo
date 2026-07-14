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

func TestCheckGraphFunctionSignatures(t *testing.T) {
	graph := testGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

type item struct{}

func (it *item) run(left, right int, data []byte, callback func(int) string) (total int, ok bool) {
	return 0, true
}

func unnamed(int, string) []byte {
	return nil
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
	body := root.Bodies[bodyIndex]
	file := prog.Graph.Packages[0].Files[body.File].File
	assertField(t, file, body.Signature.Receiver, "it", "*item")
	assertField(t, file, body.Signature.Params, "left", "int")
	assertField(t, file, body.Signature.Params, "right", "int")
	assertField(t, file, body.Signature.Params, "data", "[]byte")
	assertField(t, file, body.Signature.Params, "callback", "func(int) string")
	assertField(t, file, body.Signature.Results, "total", "int")
	assertField(t, file, body.Signature.Results, "ok", "bool")

	unnamedIndex := LookupFuncBody(root, "unnamed")
	if unnamedIndex < 0 {
		t.Fatalf("unnamed body not found: %#v", root.Bodies)
	}
	unnamed := root.Bodies[unnamedIndex]
	if len(unnamed.Signature.Params) != 2 {
		t.Fatalf("unnamed params = %#v", unnamed.Signature.Params)
	}
	assertUnnamedField(t, file, unnamed.Signature.Params[0], "int")
	assertUnnamedField(t, file, unnamed.Signature.Params[1], "string")
	if len(unnamed.Signature.Results) != 1 {
		t.Fatalf("unnamed results = %#v", unnamed.Signature.Results)
	}
	assertUnnamedField(t, file, unnamed.Signature.Results[0], "[]byte")
}

func TestCheckGraphDeclarations(t *testing.T) {
	graph := testGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import lib "example.com/case/pkg/lib"

const packageValue = 3
const message = "ok"
const enabled = true
const negative = -4

const (
	left, right int = 1, 2
)

var current, next = packageValue, lib.Value()
var computed = later()
var first = second + computed
var second = packageValue

type item struct { value int }

func later() int { return 4 }
`)},
		{Path: "/repo/case/pkg/lib/lib.go", Src: []byte(`package lib

func Value() int { return 5 }
`)},
	})
	prog := CheckGraph(graph)
	if !prog.Ok {
		t.Fatalf("CheckGraph failed: err=%d pkg=%d file=%d tok=%d", prog.Error, prog.ErrorPackage, prog.ErrorFile, prog.ErrorToken)
	}
	rootPackage := len(prog.Packages) - 1
	root := prog.Packages[rootPackage]
	file := prog.Graph.Packages[rootPackage].Files[0].File
	assertDeclSpan(t, file, root, "packageValue", SymbolConst, "", "3")
	assertDeclSpan(t, file, root, "message", SymbolConst, "", `"ok"`)
	assertDeclSpan(t, file, root, "enabled", SymbolConst, "", "true")
	assertDeclSpan(t, file, root, "negative", SymbolConst, "", "-4")
	assertDeclSpan(t, file, root, "left", SymbolConst, "int", "1, 2")
	assertDeclSpan(t, file, root, "right", SymbolConst, "int", "1, 2")
	assertDeclSpan(t, file, root, "current", SymbolVar, "", "packageValue, lib.Value()")
	assertDeclSpan(t, file, root, "next", SymbolVar, "", "packageValue, lib.Value()")
	assertDeclSpan(t, file, root, "computed", SymbolVar, "", "later()")
	assertDeclSpan(t, file, root, "first", SymbolVar, "", "second + computed")
	assertDeclSpan(t, file, root, "second", SymbolVar, "", "packageValue")
	assertDeclSpan(t, file, root, "item", SymbolType, "struct { value int }", "")
	assertDeclValues(t, file, root, "packageValue", []string{"3"})
	assertDeclValues(t, file, root, "message", []string{`"ok"`})
	assertDeclValues(t, file, root, "enabled", []string{"true"})
	assertDeclValues(t, file, root, "negative", []string{"-4"})
	assertDeclValues(t, file, root, "left", []string{"1", "2"})
	assertDeclValues(t, file, root, "right", []string{"1", "2"})
	assertDeclValues(t, file, root, "current", []string{"packageValue", "lib.Value()"})
	assertDeclValues(t, file, root, "next", []string{"packageValue", "lib.Value()"})
	assertDeclValues(t, file, root, "computed", []string{"later()"})
	assertDeclValues(t, file, root, "first", []string{"second + computed"})
	assertDeclValues(t, file, root, "second", []string{"packageValue"})
	assertDeclValues(t, file, root, "item", nil)
	assertDeclConstInt(t, root, "packageValue", 3)
	assertDeclConstString(t, root, "message", "ok")
	assertDeclConstBool(t, root, "enabled", true)
	assertDeclConstInt(t, root, "negative", -4)
	assertDeclConstInt(t, root, "left", 1)
	assertDeclConstInt(t, root, "right", 2)
	assertDeclLookupOrder(t, root, []string{"computed", "current", "enabled", "first", "item", "left", "message", "negative", "next", "packageValue", "right", "second"})
	assertDeclSourceOrder(t, root, []string{"packageValue", "message", "enabled", "negative", "left", "right", "current", "next", "computed", "first", "second", "item"})
	assertDeclInitOrder(t, root, []string{"current", "next", "computed", "second", "first"})
	assertDeclRef(t, root, "current", "packageValue", RefPackage)
	assertDeclRef(t, root, "current", "lib", RefImport)
	assertDeclSelector(t, prog, root, "current", "lib", "Value")
	assertDeclCall(t, prog, root, "current", "lib", "Value", CallImportSelector)
	assertDeclRef(t, root, "computed", "later", RefPackage)
	assertDeclCall(t, prog, root, "computed", "", "later", CallPackage)
	assertDeclDeps(t, root, "current", []string{"packageValue"})
	assertDeclDeps(t, root, "first", []string{"second", "computed"})
	assertDeclDeps(t, root, "second", []string{"packageValue"})
}

func TestCheckGraphLocalDeclarations(t *testing.T) {
	graph := testGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import lib "example.com/case/pkg/lib"

func appMain() int {
	const named = 1
	const typed int = 2
	const flag = false
	const text = "local"
	var left, right int = named, typed
	var empty string
	type alias = int
	type local struct { value int }
	var fromHelper = helper(left)
	var imported = lib.Value(fromHelper)
	var (
		groupedA, groupedB = left, right
		groupedC string
	)
	return left + right + groupedA + groupedB + imported
}

func helper(value int) int { return value }
`)},
		{Path: "/repo/case/pkg/lib/lib.go", Src: []byte(`package lib

func Value(value int) int { return value }
`)},
	})
	prog := CheckGraph(graph)
	if !prog.Ok {
		t.Fatalf("CheckGraph failed: err=%d pkg=%d file=%d tok=%d", prog.Error, prog.ErrorPackage, prog.ErrorFile, prog.ErrorToken)
	}
	rootPackage := len(prog.Packages) - 1
	root := prog.Packages[rootPackage]
	bodyIndex := LookupFuncBody(root, "appMain")
	if bodyIndex < 0 {
		t.Fatalf("appMain body not found: %#v", root.Bodies)
	}
	body := root.Bodies[bodyIndex]
	file := prog.Graph.Packages[rootPackage].Files[body.File].File
	if len(body.Locals) != 14 {
		t.Fatalf("local decls = %#v, want 14", body.Locals)
	}
	assertLocalDeclSpan(t, file, body, "named", SymbolConst, "", "1", false)
	assertLocalDeclSpan(t, file, body, "typed", SymbolConst, "int", "2", false)
	assertLocalDeclSpan(t, file, body, "flag", SymbolConst, "", "false", false)
	assertLocalDeclSpan(t, file, body, "text", SymbolConst, "", `"local"`, false)
	assertLocalDeclSpan(t, file, body, "left", SymbolVar, "int", "named, typed", false)
	assertLocalDeclSpan(t, file, body, "right", SymbolVar, "int", "named, typed", false)
	assertLocalDeclSpan(t, file, body, "empty", SymbolVar, "string", "", false)
	assertLocalDeclSpan(t, file, body, "alias", SymbolType, "int", "", true)
	assertLocalDeclSpan(t, file, body, "local", SymbolType, "struct { value int }", "", false)
	assertLocalDeclSpan(t, file, body, "fromHelper", SymbolVar, "", "helper(left)", false)
	assertLocalDeclSpan(t, file, body, "imported", SymbolVar, "", "lib.Value(fromHelper)", false)
	assertLocalDeclSpan(t, file, body, "groupedA", SymbolVar, "", "left, right", false)
	assertLocalDeclSpan(t, file, body, "groupedB", SymbolVar, "", "left, right", false)
	assertLocalDeclSpan(t, file, body, "groupedC", SymbolVar, "string", "", false)
	assertLocalDeclValues(t, file, body, "named", []string{"1"})
	assertLocalDeclValues(t, file, body, "typed", []string{"2"})
	assertLocalDeclValues(t, file, body, "flag", []string{"false"})
	assertLocalDeclValues(t, file, body, "text", []string{`"local"`})
	assertLocalDeclValues(t, file, body, "left", []string{"named", "typed"})
	assertLocalDeclValues(t, file, body, "right", []string{"named", "typed"})
	assertLocalDeclValues(t, file, body, "empty", nil)
	assertLocalDeclValues(t, file, body, "alias", nil)
	assertLocalDeclValues(t, file, body, "fromHelper", []string{"helper(left)"})
	assertLocalDeclValues(t, file, body, "imported", []string{"lib.Value(fromHelper)"})
	assertLocalDeclValues(t, file, body, "groupedA", []string{"left", "right"})
	assertLocalDeclValues(t, file, body, "groupedB", []string{"left", "right"})
	assertLocalDeclValues(t, file, body, "groupedC", nil)
	assertLocalDeclRef(t, body, "fromHelper", "helper", RefPackage)
	assertLocalDeclRef(t, body, "fromHelper", "left", RefScope)
	assertLocalDeclCall(t, prog, body, "fromHelper", "", "helper", CallPackage)
	assertLocalDeclRef(t, body, "imported", "lib", RefImport)
	assertLocalDeclRef(t, body, "imported", "fromHelper", RefScope)
	assertLocalDeclSelector(t, prog, body, "imported", "lib", "Value")
	assertLocalDeclCall(t, prog, body, "imported", "lib", "Value", CallImportSelector)
	assertLocalConstInt(t, body, "named", 1)
	assertLocalConstInt(t, body, "typed", 2)
	assertLocalConstBool(t, body, "flag", false)
	assertLocalConstString(t, body, "text", "local")
}

func TestCheckGraphExpressionShapes(t *testing.T) {
	graph := testGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

type item struct { value int }

var global = []int{1, 2, 3}
var picked = global[1]
var inline = struct { value int }{value: 4}

func appMain() int {
	var local = []item{{value: global[0]}}
	local[0] = item{value: picked}
	return local[0].value
}
`)},
	})
	prog := CheckGraph(graph)
	if !prog.Ok {
		t.Fatalf("CheckGraph failed: err=%d pkg=%d file=%d tok=%d", prog.Error, prog.ErrorPackage, prog.ErrorFile, prog.ErrorToken)
	}
	root := prog.Packages[0]
	file := prog.Graph.Packages[0].Files[0].File
	assertDeclComposite(t, file, root, "global", "[]int", []string{"1", "2", "3"})
	assertDeclIndex(t, file, root, "picked", "global", "1")
	inline := requireDecl(t, root, "inline")
	if len(inline.Composites) != 1 {
		t.Fatalf("inline composites = %#v, want 1", inline.Composites)
	}
	assertDeclComposite(t, file, root, "inline", "struct { value int }", []string{"value: 4"})

	bodyIndex := LookupFuncBody(root, "appMain")
	if bodyIndex < 0 {
		t.Fatalf("appMain body not found: %#v", root.Bodies)
	}
	body := root.Bodies[bodyIndex]
	if len(body.Indexes) != 3 {
		t.Fatalf("body indexes = %#v, want 3", body.Indexes)
	}
	if len(body.Composites) != 2 {
		t.Fatalf("body composites = %#v, want 2", body.Composites)
	}
	assertBodyIndex(t, file, body, "global", "0")
	assertBodyIndex(t, file, body, "local", "0")
	assertBodyComposite(t, file, body, "[]item", []string{"{value: global[0]}"})
	assertBodyComposite(t, file, body, "item", []string{"value: picked"})
	assertLocalIndex(t, file, body, "local", "global", "0")
	assertLocalComposite(t, file, body, "local", "[]item", []string{"{value: global[0]}"})
}

func TestCheckGraphTypes(t *testing.T) {
	graph := testGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

type item struct {
	value int
	left, right string
	data []byte "json:\"data\""
	Embedded
	*Pointer
}

type alias = item
type table map[string]int
type values []int
type fixed [4]int
type ptr *item
type callback func(int) string
type face interface { Value() int }
type reader interface {
	Embedded
	Read(p []byte) (n int, err error)
	Close() error
}
`)},
	})
	prog := CheckGraph(graph)
	if !prog.Ok {
		t.Fatalf("CheckGraph failed: err=%d pkg=%d file=%d tok=%d", prog.Error, prog.ErrorPackage, prog.ErrorFile, prog.ErrorToken)
	}
	root := prog.Packages[0]
	file := prog.Graph.Packages[0].Files[0].File
	assertType(t, file, root, "item", TypeStruct, false, "struct {\n\tvalue int\n\tleft, right string\n\tdata []byte \"json:\\\"data\\\"\"\n\tEmbedded\n\t*Pointer\n}")
	assertType(t, file, root, "alias", TypeNamed, true, "item")
	assertType(t, file, root, "table", TypeMap, false, "map[string]int")
	assertType(t, file, root, "values", TypeSlice, false, "[]int")
	assertType(t, file, root, "fixed", TypeArray, false, "[4]int")
	assertType(t, file, root, "ptr", TypePointer, false, "*item")
	assertType(t, file, root, "callback", TypeFunc, false, "func(int) string")
	assertType(t, file, root, "face", TypeInterface, false, "interface { Value() int }")
	assertType(t, file, root, "reader", TypeInterface, false, "interface {\n\tEmbedded\n\tRead(p []byte) (n int, err error)\n\tClose() error\n}")
	assertTypeShape(t, file, root, "table", "", "string", "int")
	assertTypeShape(t, file, root, "values", "", "", "int")
	assertTypeShape(t, file, root, "fixed", "4", "", "int")
	assertTypeShape(t, file, root, "ptr", "", "", "item")

	callback := root.Types[LookupType(root, "callback")]
	assertSignatureFields(t, file, callback.Signature.Params, []string{":int"})
	assertSignatureFields(t, file, callback.Signature.Results, []string{":string"})

	item := root.Types[LookupType(root, "item")]
	assertField(t, file, item.Fields, "value", "int")
	assertField(t, file, item.Fields, "left", "string")
	assertField(t, file, item.Fields, "right", "string")
	assertField(t, file, item.Fields, "data", "[]byte")
	assertStructUnnamedField(t, file, item.Fields, "Embedded")
	assertStructUnnamedField(t, file, item.Fields, "*Pointer")

	reader := root.Types[LookupType(root, "reader")]
	assertInterfaceEmbed(t, file, reader, "Embedded")
	assertInterfaceMethod(t, file, reader, "Read", []string{"p:[]byte"}, []string{"n:int", "err:error"})
	assertInterfaceMethod(t, file, reader, "Close", nil, []string{":error"})
}

func TestCheckGraphTypeReferences(t *testing.T) {
	graph := testGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import lib "example.com/case/pkg/lib"

type item struct {
	value int
	imported lib.Item
}

type alias = lib.Item
var current item

func appMain(param lib.Item) item {
	type local = item
	var next local
	var other []lib.Item
	_ = next
	_ = other
	return current
}

func importedResult() (lib.Item, int, bool) {
	return lib.Item{}, 1, true
}
`)},
		{Path: "/repo/case/pkg/lib/lib.go", Src: []byte(`package lib

type Item struct {
	Value int
}
`)},
	})
	prog := CheckGraph(graph)
	if !prog.Ok {
		t.Fatalf("CheckGraph failed: err=%d pkg=%d file=%d tok=%d", prog.Error, prog.ErrorPackage, prog.ErrorFile, prog.ErrorToken)
	}
	root := prog.Packages[len(prog.Packages)-1]
	assertTypeRef(t, root.TypeRefs, "", "int", TypeRefBuiltin)
	assertTypeRef(t, root.TypeRefs, "lib", "Item", TypeRefImportSelector)
	assertTypeRef(t, root.TypeRefs, "", "item", TypeRefPackage)

	bodyIndex := LookupFuncBody(root, "appMain")
	if bodyIndex < 0 {
		t.Fatalf("appMain body not found: %#v", root.Bodies)
	}
	body := root.Bodies[bodyIndex]
	assertTypeRef(t, body.TypeRefs, "lib", "Item", TypeRefImportSelector)
	assertTypeRef(t, body.TypeRefs, "", "item", TypeRefPackage)
	assertTypeRef(t, body.TypeRefs, "", "local", TypeRefScope)

	resultIndex := LookupFuncBody(root, "importedResult")
	if resultIndex < 0 {
		t.Fatalf("importedResult body not found: %#v", root.Bodies)
	}
	resultBody := root.Bodies[resultIndex]
	assertTypeRef(t, resultBody.TypeRefs, "lib", "Item", TypeRefImportSelector)
	if LookupScopeName(resultBody.Scope, "lib") >= 0 {
		t.Fatalf("import selector base was added to result scope: %#v", resultBody.Scope.Names)
	}
}

func TestCheckGraphMethodSets(t *testing.T) {
	graph := testGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

type item struct { value int }
type other struct{}

func (it item) Value() int { return it.value }
func (it *item) Set(value int) { it.value = value }
func (o other) Value() int { return 0 }
`)},
	})
	prog := CheckGraph(graph)
	if !prog.Ok {
		t.Fatalf("CheckGraph failed: err=%d pkg=%d file=%d tok=%d", prog.Error, prog.ErrorPackage, prog.ErrorFile, prog.ErrorToken)
	}
	root := prog.Packages[0]
	assertMethod(t, root, "item", "Value", false)
	assertMethod(t, root, "item", "Set", true)
	assertMethod(t, root, "other", "Value", false)
	itemIndex := LookupType(root, "item")
	if itemIndex < 0 {
		t.Fatalf("item type not found: %#v", root.Types)
	}
	itemMethods := root.Types[itemIndex].Methods
	if len(itemMethods) != 2 {
		t.Fatalf("item methods = %#v, want 2 methods in %#v", itemMethods, root.Methods)
	}
}

func TestCheckGraphFunctionReferences(t *testing.T) {
	graph := testGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import lib "example.com/case/pkg/lib"

const packageValue = 3

func appMain(param int) int {
	local := lib.Value(param, packageValue) + later(param + 1) + len("x")
	if local > param {
		goto done
	}
done:
	return local
}

func later(value int) int { return value }
`)},
		{Path: "/repo/case/pkg/lib/lib.go", Src: []byte(`package lib

func Value(left int, right int) int { return 4 }
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
	file := prog.Graph.Packages[len(prog.Packages)-1].Files[body.File].File
	assertBodyRef(t, body, "param", RefScope)
	assertBodyRef(t, body, "local", RefScope)
	assertBodyRef(t, body, "packageValue", RefPackage)
	assertBodyRef(t, body, "later", RefPackage)
	assertBodyRef(t, body, "lib", RefImport)
	assertBodyRef(t, body, "len", RefBuiltin)
	assertBodyRef(t, body, "done", RefLabel)
	assertPackageSelector(t, prog, body, "lib", "Value")
	assertPackageCall(t, prog, body, "later")
	assertBuiltinCall(t, body, "len")
	assertPackageSelectorCall(t, prog, body, "lib", "Value")
	assertCallArgs(t, file, body, "lib", "Value", CallImportSelector, []string{"param", "packageValue"})
	assertCallArgs(t, file, body, "", "later", CallPackage, []string{"param + 1"})
	assertCallArgs(t, file, body, "", "len", CallBuiltin, []string{`"x"`})
}

func TestCheckGraphAssignmentsAndReturns(t *testing.T) {
	graph := testGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

func appMain(param int) (first int, second int) {
	first, second = pair()
	second += len("x")
	value := first + second
	return value, second
}

func pair() (int, int) {
	return 1, 2
}
`)},
	})
	prog := CheckGraph(graph)
	if !prog.Ok {
		t.Fatalf("CheckGraph failed: err=%d pkg=%d file=%d tok=%d", prog.Error, prog.ErrorPackage, prog.ErrorFile, prog.ErrorToken)
	}
	root := prog.Packages[0]
	bodyIndex := LookupFuncBody(root, "appMain")
	if bodyIndex < 0 {
		t.Fatalf("appMain body not found: %#v", root.Bodies)
	}
	body := root.Bodies[bodyIndex]
	file := prog.Graph.Packages[0].Files[body.File].File
	if len(body.Assigns) != 3 {
		t.Fatalf("appMain assignments = %#v, want 3", body.Assigns)
	}
	assertAssign(t, file, body.Assigns[0], AssignSet, []string{"first", "second"}, []string{"pair()"})
	assertAssign(t, file, body.Assigns[1], AssignAdd, []string{"second"}, []string{"len(\"x\")"})
	assertAssign(t, file, body.Assigns[2], AssignDefine, []string{"value"}, []string{"first + second"})
	assertReturnValues(t, file, body.Returns, [][]string{{"value", "second"}})

	pairIndex := LookupFuncBody(root, "pair")
	if pairIndex < 0 {
		t.Fatalf("pair body not found: %#v", root.Bodies)
	}
	pairBody := root.Bodies[pairIndex]
	pairFile := prog.Graph.Packages[0].Files[pairBody.File].File
	assertReturnValues(t, pairFile, pairBody.Returns, [][]string{{"1", "2"}})
}

func TestCheckGraphMultilineReturnList(t *testing.T) {
	graph := testGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

func values() (int, int, int, int) {
	return 1,
		2,
		3,
		4
}
`)},
	})
	prog := CheckGraph(graph)
	if !prog.Ok {
		t.Fatalf("CheckGraph failed: err=%d pkg=%d file=%d tok=%d", prog.Error, prog.ErrorPackage, prog.ErrorFile, prog.ErrorToken)
	}
	root := prog.Packages[0]
	bodyIndex := LookupFuncBody(root, "values")
	if bodyIndex < 0 {
		t.Fatalf("values body not found: %#v", root.Bodies)
	}
	body := root.Bodies[bodyIndex]
	file := prog.Graph.Packages[0].Files[body.File].File
	assertReturnValues(t, file, body.Returns, [][]string{{"1", "2", "3", "4"}})
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

func TestCheckGraphRejectsDefiniteReturnCountMismatch(t *testing.T) {
	graph := testGraph(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nfunc broken() (int, int) { return 1 }\nfunc appMain() int { return 0 }\n")},
	})
	prog := CheckGraph(graph)
	if prog.Ok || prog.Error != CheckErrReturnCount || prog.ErrorPackage != 0 || prog.ErrorFile != 0 || prog.ErrorToken < 0 {
		t.Fatalf("return count mismatch = %#v", prog)
	}
}

func TestCheckGraphRejectsDefiniteAssignmentTypeMismatch(t *testing.T) {
	graph := testGraph(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nfunc appMain() int { var x int; x = \"wrong\"; return x }\n")},
	})
	prog := CheckGraph(graph)
	if prog.Ok || prog.Error != CheckErrType || prog.ErrorPackage != 0 || prog.ErrorFile != 0 || prog.ErrorToken < 0 {
		t.Fatalf("assignment type mismatch = %#v", prog)
	}
}

func TestCheckGraphRejectsExcludedGoroutineBeforeLowering(t *testing.T) {
	graph := testGraph(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nfunc work() {}\nfunc appMain() int { go work(); return 0 }\n")},
	})
	prog := CheckGraph(graph)
	if prog.Ok || prog.Error != CheckErrExcluded || prog.ErrorPackage != 0 || prog.ErrorFile != 0 || prog.ErrorToken < 0 {
		t.Fatalf("excluded goroutine = %#v", prog)
	}
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

func assertField(t *testing.T, file syntax.File, fields []Field, name string, typ string) {
	t.Helper()
	index := LookupField(fields, name)
	if index < 0 {
		t.Fatalf("field %q not found in %#v", name, fields)
	}
	if got := fieldTypeText(file, fields[index]); got != typ {
		t.Fatalf("field %q type = %q, want %q", name, got, typ)
	}
}

func assertUnnamedField(t *testing.T, file syntax.File, field Field, typ string) {
	t.Helper()
	if field.Name != "" || field.NameTok >= 0 {
		t.Fatalf("field = %#v, want unnamed", field)
	}
	if got := fieldTypeText(file, field); got != typ {
		t.Fatalf("unnamed field type = %q, want %q", got, typ)
	}
}

func fieldTypeText(file syntax.File, field Field) string {
	return spanText(file, field.TypeStart, field.TypeEnd)
}

func assertDeclSpan(t *testing.T, file syntax.File, info PackageInfo, name string, kind int, typ string, value string) {
	t.Helper()
	index := LookupDecl(info, name)
	if index < 0 {
		t.Fatalf("decl %q not found in %#v", name, info.Decls)
	}
	decl := info.Decls[index]
	if decl.Kind != kind {
		t.Fatalf("decl %q kind = %d, want %d", name, decl.Kind, kind)
	}
	if decl.Symbol < 0 || decl.Symbol >= len(info.Symbols) || info.Symbols[decl.Symbol].Name != name {
		t.Fatalf("decl %q symbol = %d in %#v", name, decl.Symbol, info.Symbols)
	}
	if got := spanText(file, decl.TypeStart, decl.TypeEnd); got != typ {
		t.Fatalf("decl %q type = %q, want %q", name, got, typ)
	}
	if got := spanText(file, decl.ValueStart, decl.ValueEnd); got != value {
		t.Fatalf("decl %q value = %q, want %q", name, got, value)
	}
}

func assertDeclValues(t *testing.T, file syntax.File, info PackageInfo, name string, values []string) {
	t.Helper()
	index := LookupDecl(info, name)
	if index < 0 {
		t.Fatalf("decl %q not found in %#v", name, info.Decls)
	}
	assertExprSpans(t, file, info.Decls[index].Values, values)
}

func assertDeclIndex(t *testing.T, file syntax.File, info PackageInfo, name string, base string, indexText string) {
	t.Helper()
	decl := requireDecl(t, info, name)
	assertIndexExpr(t, file, decl.Indexes, base, indexText)
}

func assertDeclComposite(t *testing.T, file syntax.File, info PackageInfo, name string, typ string, elems []string) {
	t.Helper()
	decl := requireDecl(t, info, name)
	assertCompositeExpr(t, file, decl.Composites, typ, elems)
}

func assertDeclConstInt(t *testing.T, info PackageInfo, name string, value int) {
	t.Helper()
	decl := requireDecl(t, info, name)
	if !decl.Const.Ok || decl.Const.Kind != ConstInt || decl.Const.Int != value {
		t.Fatalf("decl %q const = %#v, want int %d", name, decl.Const, value)
	}
}

func assertDeclConstString(t *testing.T, info PackageInfo, name string, value string) {
	t.Helper()
	decl := requireDecl(t, info, name)
	if !decl.Const.Ok || decl.Const.Kind != ConstString || decl.Const.String != value {
		t.Fatalf("decl %q const = %#v, want string %q", name, decl.Const, value)
	}
}

func assertDeclConstBool(t *testing.T, info PackageInfo, name string, value bool) {
	t.Helper()
	decl := requireDecl(t, info, name)
	if !decl.Const.Ok || decl.Const.Kind != ConstBool || decl.Const.Bool != value {
		t.Fatalf("decl %q const = %#v, want bool %v", name, decl.Const, value)
	}
}

func requireDecl(t *testing.T, info PackageInfo, name string) DeclInfo {
	t.Helper()
	index := LookupDecl(info, name)
	if index < 0 {
		t.Fatalf("decl %q not found in %#v", name, info.Decls)
	}
	return info.Decls[index]
}

func assertDeclRef(t *testing.T, info PackageInfo, declName string, refName string, kind int) {
	t.Helper()
	index := LookupDecl(info, declName)
	if index < 0 {
		t.Fatalf("decl %q not found in %#v", declName, info.Decls)
	}
	if LookupDeclRef(info.Decls[index], refName, kind) < 0 {
		t.Fatalf("decl %q ref %q kind %d not found in %#v", declName, refName, kind, info.Decls[index].Refs)
	}
}

func assertDeclSelector(t *testing.T, prog Program, info PackageInfo, declName string, base string, name string) {
	t.Helper()
	index := LookupDecl(info, declName)
	if index < 0 {
		t.Fatalf("decl %q not found in %#v", declName, info.Decls)
	}
	selectorIndex := LookupDeclSelector(info.Decls[index], base, name, SelectorImport)
	if selectorIndex < 0 {
		t.Fatalf("decl %q selector %s.%s not found in %#v", declName, base, name, info.Decls[index].Selectors)
	}
	selector := info.Decls[index].Selectors[selectorIndex]
	if selector.Package < 0 || selector.Package >= len(prog.Packages) {
		t.Fatalf("decl %q selector package = %d in %#v", declName, selector.Package, prog.Packages)
	}
	if selector.Symbol < 0 || selector.Symbol >= len(prog.Packages[selector.Package].Symbols) || prog.Packages[selector.Package].Symbols[selector.Symbol].Name != name {
		t.Fatalf("decl %q selector symbol = %d in %#v", declName, selector.Symbol, prog.Packages[selector.Package].Symbols)
	}
}

func assertDeclCall(t *testing.T, prog Program, info PackageInfo, declName string, base string, name string, kind int) {
	t.Helper()
	index := LookupDecl(info, declName)
	if index < 0 {
		t.Fatalf("decl %q not found in %#v", declName, info.Decls)
	}
	callIndex := LookupDeclCall(info.Decls[index], base, name, kind)
	if callIndex < 0 {
		t.Fatalf("decl %q call %s.%s kind %d not found in %#v", declName, base, name, kind, info.Decls[index].Calls)
	}
	call := info.Decls[index].Calls[callIndex]
	if kind == CallPackage || kind == CallImportSelector {
		if call.Package < 0 || call.Package >= len(prog.Packages) {
			t.Fatalf("decl %q call package = %d in %#v", declName, call.Package, prog.Packages)
		}
		if call.Symbol < 0 || call.Symbol >= len(prog.Packages[call.Package].Symbols) || prog.Packages[call.Package].Symbols[call.Symbol].Name != name {
			t.Fatalf("decl %q call symbol = %d in %#v", declName, call.Symbol, prog.Packages[call.Package].Symbols)
		}
	}
}

func assertDeclLookupOrder(t *testing.T, info PackageInfo, names []string) {
	t.Helper()
	if len(info.Decls) != len(names) {
		t.Fatalf("decls = %#v, want %v", info.Decls, names)
	}
	for i := 0; i < len(names); i++ {
		if info.Decls[i].Name != names[i] {
			t.Fatalf("decl lookup order %d = %q, want %q in %#v", i, info.Decls[i].Name, names[i], info.Decls)
		}
	}
}

func assertDeclSourceOrder(t *testing.T, info PackageInfo, names []string) {
	t.Helper()
	if len(info.DeclOrder) != len(names) {
		t.Fatalf("decl order = %#v, want %v", info.DeclOrder, names)
	}
	for i := 0; i < len(names); i++ {
		index := info.DeclOrder[i]
		if index < 0 || index >= len(info.Decls) {
			t.Fatalf("decl order %d index = %d in %#v", i, index, info.DeclOrder)
		}
		if info.Decls[index].Name != names[i] {
			t.Fatalf("decl source order %d = %q, want %q in %#v", i, info.Decls[index].Name, names[i], info.DeclOrder)
		}
	}
}

func assertDeclInitOrder(t *testing.T, info PackageInfo, names []string) {
	t.Helper()
	if len(info.InitOrder) != len(names) {
		t.Fatalf("init order = %#v, want %v", info.InitOrder, names)
	}
	for i := 0; i < len(names); i++ {
		index := info.InitOrder[i]
		if index < 0 || index >= len(info.Decls) {
			t.Fatalf("init order %d index = %d in %#v", i, index, info.InitOrder)
		}
		if info.Decls[index].Name != names[i] {
			t.Fatalf("init order %d = %q, want %q in %#v", i, info.Decls[index].Name, names[i], info.InitOrder)
		}
	}
}

func assertDeclDeps(t *testing.T, info PackageInfo, name string, deps []string) {
	t.Helper()
	index := LookupDecl(info, name)
	if index < 0 {
		t.Fatalf("decl %q not found in %#v", name, info.Decls)
	}
	decl := info.Decls[index]
	if len(decl.Deps) != len(deps) {
		t.Fatalf("decl %q deps = %#v, want %v", name, decl.Deps, deps)
	}
	for i := 0; i < len(deps); i++ {
		dep := decl.Deps[i]
		if dep < 0 || dep >= len(info.Decls) {
			t.Fatalf("decl %q dep %d index = %d in %#v", name, i, dep, decl.Deps)
		}
		if info.Decls[dep].Name != deps[i] {
			t.Fatalf("decl %q dep %d = %q, want %q in %#v", name, i, info.Decls[dep].Name, deps[i], decl.Deps)
		}
	}
}

func assertLocalDeclSpan(t *testing.T, file syntax.File, body FuncBody, name string, kind int, typ string, value string, alias bool) {
	t.Helper()
	index := LookupLocalDecl(body, name)
	if index < 0 {
		t.Fatalf("local decl %q not found in %#v", name, body.Locals)
	}
	decl := body.Locals[index]
	if decl.Kind != kind {
		t.Fatalf("local decl %q kind = %d, want %d", name, decl.Kind, kind)
	}
	if decl.Scope < 0 || decl.Scope >= len(body.Scope.Names) || body.Scope.Names[decl.Scope].Name != name {
		t.Fatalf("local decl %q scope = %d in %#v", name, decl.Scope, body.Scope.Names)
	}
	if decl.Alias != alias {
		t.Fatalf("local decl %q alias = %v, want %v", name, decl.Alias, alias)
	}
	if got := spanText(file, decl.TypeStart, decl.TypeEnd); got != typ {
		t.Fatalf("local decl %q type = %q, want %q", name, got, typ)
	}
	if got := spanText(file, decl.ValueStart, decl.ValueEnd); got != value {
		t.Fatalf("local decl %q value = %q, want %q", name, got, value)
	}
}

func assertLocalDeclValues(t *testing.T, file syntax.File, body FuncBody, name string, values []string) {
	t.Helper()
	index := LookupLocalDecl(body, name)
	if index < 0 {
		t.Fatalf("local decl %q not found in %#v", name, body.Locals)
	}
	assertExprSpans(t, file, body.Locals[index].Values, values)
}

func assertLocalIndex(t *testing.T, file syntax.File, body FuncBody, name string, base string, indexText string) {
	t.Helper()
	decl := requireLocalDecl(t, body, name)
	assertIndexExpr(t, file, decl.Indexes, base, indexText)
}

func assertLocalComposite(t *testing.T, file syntax.File, body FuncBody, name string, typ string, elems []string) {
	t.Helper()
	decl := requireLocalDecl(t, body, name)
	assertCompositeExpr(t, file, decl.Composites, typ, elems)
}

func assertLocalConstInt(t *testing.T, body FuncBody, name string, value int) {
	t.Helper()
	decl := requireLocalDecl(t, body, name)
	if !decl.Const.Ok || decl.Const.Kind != ConstInt || decl.Const.Int != value {
		t.Fatalf("local decl %q const = %#v, want int %d", name, decl.Const, value)
	}
}

func assertLocalConstString(t *testing.T, body FuncBody, name string, value string) {
	t.Helper()
	decl := requireLocalDecl(t, body, name)
	if !decl.Const.Ok || decl.Const.Kind != ConstString || decl.Const.String != value {
		t.Fatalf("local decl %q const = %#v, want string %q", name, decl.Const, value)
	}
}

func assertLocalConstBool(t *testing.T, body FuncBody, name string, value bool) {
	t.Helper()
	decl := requireLocalDecl(t, body, name)
	if !decl.Const.Ok || decl.Const.Kind != ConstBool || decl.Const.Bool != value {
		t.Fatalf("local decl %q const = %#v, want bool %v", name, decl.Const, value)
	}
}

func requireLocalDecl(t *testing.T, body FuncBody, name string) LocalDeclInfo {
	t.Helper()
	index := LookupLocalDecl(body, name)
	if index < 0 {
		t.Fatalf("local decl %q not found in %#v", name, body.Locals)
	}
	return body.Locals[index]
}

func assertLocalDeclRef(t *testing.T, body FuncBody, declName string, refName string, kind int) {
	t.Helper()
	index := LookupLocalDecl(body, declName)
	if index < 0 {
		t.Fatalf("local decl %q not found in %#v", declName, body.Locals)
	}
	if LookupLocalDeclRef(body.Locals[index], refName, kind) < 0 {
		t.Fatalf("local decl %q ref %q kind %d not found in %#v", declName, refName, kind, body.Locals[index].Refs)
	}
}

func assertLocalDeclSelector(t *testing.T, prog Program, body FuncBody, declName string, base string, name string) {
	t.Helper()
	index := LookupLocalDecl(body, declName)
	if index < 0 {
		t.Fatalf("local decl %q not found in %#v", declName, body.Locals)
	}
	selectorIndex := LookupLocalDeclSelector(body.Locals[index], base, name, SelectorImport)
	if selectorIndex < 0 {
		t.Fatalf("local decl %q selector %s.%s not found in %#v", declName, base, name, body.Locals[index].Selectors)
	}
	selector := body.Locals[index].Selectors[selectorIndex]
	if selector.Package < 0 || selector.Package >= len(prog.Packages) {
		t.Fatalf("local decl %q selector package = %d in %#v", declName, selector.Package, prog.Packages)
	}
	if selector.Symbol < 0 || selector.Symbol >= len(prog.Packages[selector.Package].Symbols) || prog.Packages[selector.Package].Symbols[selector.Symbol].Name != name {
		t.Fatalf("local decl %q selector symbol = %d in %#v", declName, selector.Symbol, prog.Packages[selector.Package].Symbols)
	}
}

func assertLocalDeclCall(t *testing.T, prog Program, body FuncBody, declName string, base string, name string, kind int) {
	t.Helper()
	index := LookupLocalDecl(body, declName)
	if index < 0 {
		t.Fatalf("local decl %q not found in %#v", declName, body.Locals)
	}
	callIndex := LookupLocalDeclCall(body.Locals[index], base, name, kind)
	if callIndex < 0 {
		t.Fatalf("local decl %q call %s.%s kind %d not found in %#v", declName, base, name, kind, body.Locals[index].Calls)
	}
	call := body.Locals[index].Calls[callIndex]
	if kind == CallPackage || kind == CallImportSelector {
		if call.Package < 0 || call.Package >= len(prog.Packages) {
			t.Fatalf("local decl %q call package = %d in %#v", declName, call.Package, prog.Packages)
		}
		if call.Symbol < 0 || call.Symbol >= len(prog.Packages[call.Package].Symbols) || prog.Packages[call.Package].Symbols[call.Symbol].Name != name {
			t.Fatalf("local decl %q call symbol = %d in %#v", declName, call.Symbol, prog.Packages[call.Package].Symbols)
		}
	}
}

func assertType(t *testing.T, file syntax.File, info PackageInfo, name string, kind int, alias bool, typ string) {
	t.Helper()
	index := LookupType(info, name)
	if index < 0 {
		t.Fatalf("type %q not found in %#v", name, info.Types)
	}
	tp := info.Types[index]
	if tp.Kind != kind {
		t.Fatalf("type %q kind = %d, want %d", name, tp.Kind, kind)
	}
	if tp.Alias != alias {
		t.Fatalf("type %q alias = %v, want %v", name, tp.Alias, alias)
	}
	if tp.Symbol < 0 || tp.Symbol >= len(info.Symbols) || info.Symbols[tp.Symbol].Name != name {
		t.Fatalf("type %q symbol = %d in %#v", name, tp.Symbol, info.Symbols)
	}
	if got := spanText(file, tp.TypeStart, tp.TypeEnd); got != typ {
		t.Fatalf("type %q span = %q, want %q", name, got, typ)
	}
}

func assertTypeShape(t *testing.T, file syntax.File, info PackageInfo, name string, lenText string, keyText string, elemText string) {
	t.Helper()
	index := LookupType(info, name)
	if index < 0 {
		t.Fatalf("type %q not found in %#v", name, info.Types)
	}
	tp := info.Types[index]
	if got := spanText(file, tp.LenStart, tp.LenEnd); got != lenText {
		t.Fatalf("type %q len span = %q, want %q", name, got, lenText)
	}
	if got := spanText(file, tp.KeyStart, tp.KeyEnd); got != keyText {
		t.Fatalf("type %q key span = %q, want %q", name, got, keyText)
	}
	if got := spanText(file, tp.ElemStart, tp.ElemEnd); got != elemText {
		t.Fatalf("type %q elem span = %q, want %q", name, got, elemText)
	}
}

func assertStructUnnamedField(t *testing.T, file syntax.File, fields []Field, typ string) {
	t.Helper()
	for i := 0; i < len(fields); i++ {
		if fields[i].Name == "" && fields[i].NameTok < 0 && fieldTypeText(file, fields[i]) == typ {
			return
		}
	}
	t.Fatalf("unnamed field type %q not found in %#v", typ, fields)
}

func assertInterfaceMethod(t *testing.T, file syntax.File, tp TypeInfo, name string, params []string, results []string) {
	t.Helper()
	index := LookupInterfaceMethod(tp.InterfaceMethods, name)
	if index < 0 {
		t.Fatalf("interface method %q not found in %#v", name, tp.InterfaceMethods)
	}
	method := tp.InterfaceMethods[index]
	assertSignatureFields(t, file, method.Signature.Params, params)
	assertSignatureFields(t, file, method.Signature.Results, results)
}

func assertInterfaceEmbed(t *testing.T, file syntax.File, tp TypeInfo, typ string) {
	t.Helper()
	for i := 0; i < len(tp.InterfaceEmbeds); i++ {
		if spanText(file, tp.InterfaceEmbeds[i].TypeStart, tp.InterfaceEmbeds[i].TypeEnd) == typ {
			return
		}
	}
	t.Fatalf("interface embed %q not found in %#v", typ, tp.InterfaceEmbeds)
}

func assertTypeRef(t *testing.T, refs []TypeRef, base string, name string, kind int) {
	t.Helper()
	index := LookupTypeRef(refs, base, name, kind)
	if index < 0 {
		t.Fatalf("type ref %s.%s kind %d not found in %#v", base, name, kind, refs)
	}
	ref := refs[index]
	if kind == TypeRefPackage && (ref.Package < 0 || ref.Symbol < 0) {
		t.Fatalf("package type ref %q unresolved: %#v", name, ref)
	}
	if kind == TypeRefImportSelector && (ref.Package < 0 || ref.Symbol < 0) {
		t.Fatalf("import selector type ref %s.%s unresolved: %#v", base, name, ref)
	}
}

func assertSignatureFields(t *testing.T, file syntax.File, fields []Field, want []string) {
	t.Helper()
	if len(fields) != len(want) {
		t.Fatalf("signature fields = %#v, want %d", fields, len(want))
	}
	for i := 0; i < len(want); i++ {
		got := fields[i].Name + ":" + fieldTypeText(file, fields[i])
		if got != want[i] {
			t.Fatalf("signature field %d = %q, want %q in %#v", i, got, want[i], fields)
		}
	}
}

func assertMethod(t *testing.T, info PackageInfo, receiver string, name string, pointer bool) {
	t.Helper()
	index := LookupMethod(info, receiver, name)
	if index < 0 {
		t.Fatalf("method %s.%s not found in %#v", receiver, name, info.Methods)
	}
	method := info.Methods[index]
	if method.Pointer != pointer {
		t.Fatalf("method %s.%s pointer = %v, want %v", receiver, name, method.Pointer, pointer)
	}
	if method.Type < 0 || method.Type >= len(info.Types) || info.Types[method.Type].Name != receiver {
		t.Fatalf("method %s.%s type = %d in %#v", receiver, name, method.Type, info.Types)
	}
	if method.Symbol < 0 || method.Symbol >= len(info.Symbols) || info.Symbols[method.Symbol].Name != receiver+"."+name {
		t.Fatalf("method %s.%s symbol = %d in %#v", receiver, name, method.Symbol, info.Symbols)
	}
	if method.Body < 0 || method.Body >= len(info.Bodies) || info.Bodies[method.Body].Name != receiver+"."+name {
		t.Fatalf("method %s.%s body = %d in %#v", receiver, name, method.Body, info.Bodies)
	}
}

func spanText(file syntax.File, startTok int, endTok int) string {
	if startTok < 0 || endTok <= startTok || endTok > len(file.Tokens) {
		return ""
	}
	start := file.Tokens[startTok].Start
	end := file.Tokens[endTok-1].End
	if start < 0 || end < start || end > len(file.Src) {
		return ""
	}
	return string(file.Src[start:end])
}

func assertBodyRef(t *testing.T, body FuncBody, name string, kind int) {
	t.Helper()
	index := LookupBodyRef(body, name, kind)
	if index < 0 {
		t.Fatalf("body ref %q kind %d not found in %#v", name, kind, body.Refs)
	}
}

func assertPackageSelector(t *testing.T, prog Program, body FuncBody, base string, name string) {
	t.Helper()
	index := LookupSelector(body, base, name, SelectorImport)
	if index < 0 {
		t.Fatalf("selector %s.%s not found in %#v", base, name, body.Selectors)
	}
	selector := body.Selectors[index]
	if selector.Package < 0 || selector.Package >= len(prog.Packages) {
		t.Fatalf("selector %s.%s package = %d in %#v", base, name, selector.Package, prog.Packages)
	}
	target := prog.Packages[selector.Package]
	if selector.Symbol < 0 || selector.Symbol >= len(target.Symbols) || target.Symbols[selector.Symbol].Name != name {
		t.Fatalf("selector %s.%s symbol = %d in package %#v", base, name, selector.Symbol, target)
	}
}

func assertPackageCall(t *testing.T, prog Program, body FuncBody, name string) {
	t.Helper()
	index := LookupCall(body, "", name, CallPackage)
	if index < 0 {
		t.Fatalf("package call %q not found in %#v", name, body.Calls)
	}
	call := body.Calls[index]
	if call.Package < 0 || call.Package >= len(prog.Packages) {
		t.Fatalf("package call %q package = %d", name, call.Package)
	}
	if call.Symbol < 0 || call.Symbol >= len(prog.Packages[call.Package].Symbols) || prog.Packages[call.Package].Symbols[call.Symbol].Name != name {
		t.Fatalf("package call %q symbol = %d in %#v", name, call.Symbol, prog.Packages[call.Package].Symbols)
	}
}

func assertBuiltinCall(t *testing.T, body FuncBody, name string) {
	t.Helper()
	if LookupCall(body, "", name, CallBuiltin) < 0 {
		t.Fatalf("builtin call %q not found in %#v", name, body.Calls)
	}
}

func assertPackageSelectorCall(t *testing.T, prog Program, body FuncBody, base string, name string) {
	t.Helper()
	index := LookupCall(body, base, name, CallImportSelector)
	if index < 0 {
		t.Fatalf("package selector call %s.%s not found in %#v", base, name, body.Calls)
	}
	call := body.Calls[index]
	if call.Package < 0 || call.Package >= len(prog.Packages) {
		t.Fatalf("package selector call %s.%s package = %d", base, name, call.Package)
	}
	if call.Symbol < 0 || call.Symbol >= len(prog.Packages[call.Package].Symbols) || prog.Packages[call.Package].Symbols[call.Symbol].Name != name {
		t.Fatalf("package selector call %s.%s symbol = %d in %#v", base, name, call.Symbol, prog.Packages[call.Package].Symbols)
	}
}

func assertCallArgs(t *testing.T, file syntax.File, body FuncBody, base string, name string, kind int, args []string) {
	t.Helper()
	index := LookupCall(body, base, name, kind)
	if index < 0 {
		t.Fatalf("call %s.%s kind %d not found in %#v", base, name, kind, body.Calls)
	}
	assertExprSpans(t, file, body.Calls[index].Args, args)
}

func assertAssign(t *testing.T, file syntax.File, assign AssignInfo, kind int, targets []string, values []string) {
	t.Helper()
	if assign.Kind != kind {
		t.Fatalf("assignment kind = %d, want %d: %#v", assign.Kind, kind, assign)
	}
	if len(assign.Targets) != len(targets) {
		t.Fatalf("assignment targets = %#v, want %v", assign.Targets, targets)
	}
	for i := 0; i < len(targets); i++ {
		target := assign.Targets[i]
		if target.Name != targets[i] {
			t.Fatalf("assignment target %d = %q, want %q in %#v", i, target.Name, targets[i], assign.Targets)
		}
		if target.Ref.Kind != RefScope {
			t.Fatalf("assignment target %q ref = %#v, want scope ref", target.Name, target.Ref)
		}
		if got := spanText(file, target.Span.StartTok, target.Span.EndTok); got != targets[i] {
			t.Fatalf("assignment target %q span = %q", target.Name, got)
		}
	}
	assertExprSpans(t, file, assign.Values, values)
}

func assertReturnValues(t *testing.T, file syntax.File, returns []ReturnInfo, want [][]string) {
	t.Helper()
	if len(returns) != len(want) {
		t.Fatalf("returns = %#v, want %v", returns, want)
	}
	for i := 0; i < len(want); i++ {
		assertExprSpans(t, file, returns[i].Values, want[i])
	}
}

func assertExprSpans(t *testing.T, file syntax.File, spans []ExprSpan, want []string) {
	t.Helper()
	if len(spans) != len(want) {
		t.Fatalf("expr spans = %#v, want %v", spans, want)
	}
	for i := 0; i < len(want); i++ {
		if got := spanText(file, spans[i].StartTok, spans[i].EndTok); got != want[i] {
			t.Fatalf("expr span %d = %q, want %q in %#v", i, got, want[i], spans)
		}
	}
}

func assertBodyIndex(t *testing.T, file syntax.File, body FuncBody, base string, indexText string) {
	t.Helper()
	assertIndexExpr(t, file, body.Indexes, base, indexText)
}

func assertBodyComposite(t *testing.T, file syntax.File, body FuncBody, typ string, elems []string) {
	t.Helper()
	assertCompositeExpr(t, file, body.Composites, typ, elems)
}

func assertIndexExpr(t *testing.T, file syntax.File, indexes []IndexExpr, base string, indexText string) {
	t.Helper()
	for i := 0; i < len(indexes); i++ {
		index := indexes[i]
		if spanText(file, index.BaseStart, index.BaseEnd) == base && spanText(file, index.IndexStart, index.IndexEnd) == indexText {
			return
		}
	}
	t.Fatalf("index %s[%s] not found in %#v", base, indexText, indexes)
}

func assertCompositeExpr(t *testing.T, file syntax.File, composites []CompositeExpr, typ string, elems []string) {
	t.Helper()
	for i := 0; i < len(composites); i++ {
		composite := composites[i]
		if spanText(file, composite.TypeStart, composite.TypeEnd) == typ {
			assertExprSpans(t, file, composite.Elems, elems)
			return
		}
	}
	t.Fatalf("composite %s not found in %#v", typ, composites)
}

func bodyHasStmt(body syntax.Body, kind int) bool {
	for i := 0; i < len(body.Stmts); i++ {
		if body.Stmts[i].Kind == kind {
			return true
		}
	}
	return false
}
