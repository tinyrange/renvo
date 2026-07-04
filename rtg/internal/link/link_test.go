package link

import (
	"bytes"
	"testing"

	"j5.nz/rtg/rtg/internal/build"
	"j5.nz/rtg/rtg/internal/load"
	"j5.nz/rtg/rtg/internal/unit"
	"j5.nz/rtg/rtgunit"
)

func TestLinkBuildCombinesPackageUnits(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import "example.com/case/pkg/lib"

func appMain() int { return lib.Value() }
`)},
		{Path: "/repo/case/pkg/lib/lib.go", Src: []byte(`package lib

const answer = 42

func Value() int { return answer }
`)},
	})
	linked := LinkBuild(result)
	if !linked.Ok {
		t.Fatalf("LinkBuild failed: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	if linked.Program.Package != "main" {
		t.Fatalf("linked package = %q, want main", linked.Program.Package)
	}
	if !bytes.Contains(linked.Program.Text, []byte("package lib")) || !bytes.Contains(linked.Program.Text, []byte("package main")) {
		t.Fatalf("linked text missing package sources: %q", string(linked.Program.Text))
	}
	decoded, err := rtgunit.Unmarshal(linked.Data)
	if err != nil {
		t.Fatalf("linked unit did not decode: %v", err)
	}
	if decoded.Package != "main" {
		t.Fatalf("decoded package = %q, want main", decoded.Package)
	}
	if len(decoded.Decls) != 1 {
		t.Fatalf("decoded decl count = %d, want 1", len(decoded.Decls))
	}
	if len(decoded.Funcs) != 2 {
		t.Fatalf("decoded func count = %d, want 2", len(decoded.Funcs))
	}
	if string(decoded.Text[decoded.Decls[0].NameStart:decoded.Decls[0].NameEnd]) != "answer" {
		t.Fatalf("decl name = %q", string(decoded.Text[decoded.Decls[0].NameStart:decoded.Decls[0].NameEnd]))
	}
	if functionName(decoded, decoded.Funcs[0]) != "Value" || functionName(decoded, decoded.Funcs[1]) != "appMain" {
		t.Fatalf("function names = %q %q", functionName(decoded, decoded.Funcs[0]), functionName(decoded, decoded.Funcs[1]))
	}
}

func TestLinkBuildUsesSerializedPackageUnitData(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import "example.com/case/pkg/lib"

func appMain() int { return lib.Value() }
`)},
		{Path: "/repo/case/pkg/lib/lib.go", Src: []byte(`package lib

func Value() int { return 42 }
`)},
	})
	for i := 0; i < len(result.Units); i++ {
		result.Units[i].Program = unit.Program{}
	}
	linked := LinkBuild(result)
	if !linked.Ok {
		t.Fatalf("LinkBuild failed after clearing in-memory programs: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	if len(linked.Program.Funcs) != 2 {
		t.Fatalf("linked func count = %d, want 2", len(linked.Program.Funcs))
	}

	result.Units[0].Data = append(result.Units[0].Data, 0)
	linked = LinkBuild(result)
	if linked.Ok || linked.Error != LinkErrUnit || linked.ErrorPackage != 0 {
		t.Fatalf("corrupt serialized unit result = %#v", linked)
	}
}

func TestLinkUnitsAdjustsTokenOffsets(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import "example.com/case/pkg/lib"

func appMain() int { return lib.Value() }
`)},
		{Path: "/repo/case/pkg/lib/lib.go", Src: []byte("package lib\n\nfunc Value() int { return 1 }\n")},
	})
	program, ok := LinkUnits(result.Units, result.Root)
	if !ok {
		t.Fatal("LinkUnits failed")
	}
	if len(program.Funcs) != 2 {
		t.Fatalf("func count = %d, want 2", len(program.Funcs))
	}
	first := program.Funcs[0]
	second := program.Funcs[1]
	if string(program.Text[first.NameStart:first.NameEnd]) != "Value" {
		t.Fatalf("first func name = %q", string(program.Text[first.NameStart:first.NameEnd]))
	}
	if string(program.Text[second.NameStart:second.NameEnd]) != "appMain" {
		t.Fatalf("second func name = %q", string(program.Text[second.NameStart:second.NameEnd]))
	}
	if program.Tokens[first.NameTok].Start != first.NameStart {
		t.Fatalf("first func name token start = %d, want %d", program.Tokens[first.NameTok].Start, first.NameStart)
	}
	if program.Tokens[second.NameTok].Start != second.NameStart {
		t.Fatalf("second func name token start = %d, want %d", program.Tokens[second.NameTok].Start, second.NameStart)
	}
	if program.Tokens[second.NameTok].Line <= program.Tokens[first.NameTok].Line {
		t.Fatalf("line offsets were not adjusted: first=%d second=%d", program.Tokens[first.NameTok].Line, program.Tokens[second.NameTok].Line)
	}
	if program.Tokens[len(program.Tokens)-1].Kind != unit.TokenEOF {
		t.Fatalf("last token kind = %d, want EOF", program.Tokens[len(program.Tokens)-1].Kind)
	}
}

func TestLinkUnitsPreservesExpressionShapes(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import "example.com/case/pkg/lib"

func appMain() int { return lib.Value(1) }
`)},
		{Path: "/repo/case/pkg/lib/lib.go", Src: []byte(`package lib

type Numbers []int

var Values = []int{1, 2}

func Value(i int) int {
	var typed Numbers = Values
	out := Values[i]
	return out
}
`)},
	})
	program, ok := LinkUnits(result.Units, result.Root)
	if !ok {
		t.Fatal("LinkUnits failed")
	}
	numbers := findLinkedDecl(program, "Numbers")
	values := findLinkedDecl(program, "Values")
	valueFn := findLinkedFunc(program, "Value")
	appMain := findLinkedFunc(program, "appMain")
	if numbers < 0 || values < 0 || valueFn < 0 || appMain < 0 {
		t.Fatalf("linked rows missing: decls=%#v funcs=%#v", program.Decls, program.Funcs)
	}
	if len(program.Types) != 1 {
		t.Fatalf("linked types = %#v, want 1", program.Types)
	}
	typ := program.Types[0]
	if typ.Decl != numbers || linkedText(program, typ.NameStart, typ.NameEnd) != "Numbers" ||
		typ.Kind != unit.TypeSlice ||
		linkedSpanText(program, typ.TypeStart, typ.TypeEnd) != "[]int" ||
		linkedSpanText(program, typ.ElemStart, typ.ElemEnd) != "int" {
		t.Fatalf("linked type = %#v, decl %d", typ, numbers)
	}
	foundNumberElemRef := false
	for i := 0; i < len(program.TypeRefs); i++ {
		ref := program.TypeRefs[i]
		if ref.OwnerKind == unit.OwnerDecl && ref.OwnerIndex == numbers &&
			ref.Kind == unit.TypeRefBuiltin && linkedTokenText(program, ref.Token) == "int" {
			foundNumberElemRef = true
		}
	}
	if !foundNumberElemRef {
		t.Fatalf("linked Numbers int type ref not found in %#v", program.TypeRefs)
	}
	foundTypedLocal := false
	for i := 0; i < len(program.Locals); i++ {
		local := program.Locals[i]
		if local.FuncIndex == valueFn &&
			local.Kind == unit.TokenVar &&
			linkedText(program, local.NameStart, local.NameEnd) == "typed" &&
			linkedSpanText(program, local.TypeStart, local.TypeEnd) == "Numbers" &&
			linkedSpanText(program, local.ValueStart, local.ValueEnd) == "Values" {
			foundTypedLocal = true
		}
	}
	if !foundTypedLocal {
		t.Fatalf("linked typed local not found in %#v", program.Locals)
	}
	if len(program.Composites) != 1 {
		t.Fatalf("linked composites = %#v, want 1", program.Composites)
	}
	if len(program.Indexes) != 1 {
		t.Fatalf("linked indexes = %#v, want 1", program.Indexes)
	}
	if len(program.Assigns) != 1 {
		t.Fatalf("linked assignments = %#v, want 1", program.Assigns)
	}
	if len(program.Returns) != 2 {
		t.Fatalf("linked returns = %#v, want 2", program.Returns)
	}
	if len(program.Calls) != 1 {
		t.Fatalf("linked calls = %#v, want 1", program.Calls)
	}
	if len(program.Selectors) != 1 {
		t.Fatalf("linked selectors = %#v, want 1", program.Selectors)
	}
	composite := program.Composites[0]
	if composite.OwnerKind != unit.OwnerDecl || composite.OwnerIndex != values || linkedSpanText(program, composite.TypeStart, composite.TypeEnd) != "[]int" {
		t.Fatalf("linked composite = %#v, owner decl %d", composite, values)
	}
	index := program.Indexes[0]
	if index.OwnerKind != unit.OwnerFunc || index.OwnerIndex != valueFn ||
		linkedSpanText(program, index.BaseStart, index.BaseEnd) != "Values" ||
		linkedSpanText(program, index.IndexStart, index.IndexEnd) != "i" {
		t.Fatalf("linked index = %#v, owner func %d", index, valueFn)
	}
	assign := program.Assigns[0]
	if assign.FuncIndex != valueFn || assign.Kind != unit.AssignDefine ||
		linkedSpanText(program, assign.LeftStart, assign.LeftEnd) != "out" ||
		linkedSpanText(program, assign.RightStart, assign.RightEnd) != "Values[i]" {
		t.Fatalf("linked assignment = %#v, owner func %d", assign, valueFn)
	}
	foundValueReturn := false
	for i := 0; i < len(program.Returns); i++ {
		ret := program.Returns[i]
		if ret.FuncIndex == valueFn && len(ret.Values) == 1 && linkedSpanText(program, ret.Values[0].StartTok, ret.Values[0].EndTok) == "out" {
			foundValueReturn = true
		}
	}
	if !foundValueReturn {
		t.Fatalf("linked Value return not found in %#v", program.Returns)
	}
	call := program.Calls[0]
	if call.OwnerKind != unit.OwnerFunc || call.OwnerIndex != appMain || call.Kind != unit.CallImportSelector ||
		linkedTokenText(program, call.BaseTok) != "lib" ||
		linkedTokenText(program, call.CalleeTok) != "Value" ||
		len(call.Args) != 1 ||
		linkedSpanText(program, call.Args[0].StartTok, call.Args[0].EndTok) != "1" {
		t.Fatalf("linked call = %#v, owner func %d", call, appMain)
	}
	selector := program.Selectors[0]
	if selector.OwnerKind != unit.OwnerFunc || selector.OwnerIndex != appMain || selector.Kind != unit.SelectorImport ||
		linkedTokenText(program, selector.BaseTok) != "lib" ||
		linkedTokenText(program, selector.NameTok) != "Value" ||
		selector.BaseKind != unit.RefImport {
		t.Fatalf("linked selector = %#v, owner func %d", selector, appMain)
	}
	foundLibRef := false
	for i := 0; i < len(program.Refs); i++ {
		ref := program.Refs[i]
		if ref.OwnerKind == unit.OwnerFunc && ref.OwnerIndex == appMain && ref.Kind == unit.RefImport && linkedTokenText(program, ref.Token) == "lib" {
			foundLibRef = true
		}
	}
	if !foundLibRef {
		t.Fatalf("linked lib ref not found in %#v", program.Refs)
	}
}

func TestLinkBuildRejectsInvalidInput(t *testing.T) {
	badBuild := build.Result{Ok: false, ErrorPackage: 7}
	linked := LinkBuild(badBuild)
	if linked.Ok || linked.Error != LinkErrBuild || linked.ErrorPackage != 7 {
		t.Fatalf("bad build link result = %#v", linked)
	}

	linked = LinkBuild(build.Result{Ok: true, Root: -1})
	if linked.Ok || linked.Error != LinkErrRoot {
		t.Fatalf("bad root link result = %#v", linked)
	}

	if _, ok := LinkUnits(nil, 0); ok {
		t.Fatal("LinkUnits accepted empty unit list")
	}
}

func buildFromFiles(t *testing.T, files []load.SourceFile) build.Result {
	t.Helper()
	workspace := load.LoadWorkspace("/repo/case", "/std", "./cmd/app", files)
	if !workspace.Ok {
		t.Fatalf("LoadWorkspace failed: err=%d file=%d", workspace.Error, workspace.ErrorFile)
	}
	result := build.BuildUnits(workspace.Graph)
	if !result.Ok {
		t.Fatalf("BuildUnits failed: err=%d pkg=%d file=%d tok=%d", result.Error, result.ErrorPackage, result.ErrorFile, result.ErrorToken)
	}
	return result
}

func functionName(program rtgunit.Program, fn rtgunit.Func) string {
	if fn.NameStart < 0 || fn.NameEnd < fn.NameStart || fn.NameEnd > len(program.Text) {
		return ""
	}
	return string(program.Text[fn.NameStart:fn.NameEnd])
}

func findLinkedDecl(program unit.Program, name string) int {
	for i := 0; i < len(program.Decls); i++ {
		decl := program.Decls[i]
		if decl.NameStart >= 0 && decl.NameEnd <= len(program.Text) && string(program.Text[decl.NameStart:decl.NameEnd]) == name {
			return i
		}
	}
	return -1
}

func findLinkedFunc(program unit.Program, name string) int {
	for i := 0; i < len(program.Funcs); i++ {
		fn := program.Funcs[i]
		if fn.NameStart >= 0 && fn.NameEnd <= len(program.Text) && string(program.Text[fn.NameStart:fn.NameEnd]) == name {
			return i
		}
	}
	return -1
}

func linkedSpanText(program unit.Program, startTok int, endTok int) string {
	if startTok < 0 || endTok <= startTok || endTok > len(program.Tokens) {
		return ""
	}
	start := program.Tokens[startTok].Start
	last := program.Tokens[endTok-1]
	end := last.Start + last.Size
	if start < 0 || end < start || end > len(program.Text) {
		return ""
	}
	return string(program.Text[start:end])
}

func linkedTokenText(program unit.Program, tok int) string {
	if tok < 0 || tok >= len(program.Tokens) {
		return ""
	}
	token := program.Tokens[tok]
	if token.Kind == unit.TokenEOF || token.Size == 0 {
		return ""
	}
	if token.Start < 0 || token.Start+token.Size > len(program.Text) {
		return ""
	}
	return string(program.Text[token.Start : token.Start+token.Size])
}

func linkedText(program unit.Program, start int, end int) string {
	if start < 0 || end < start || end > len(program.Text) {
		return ""
	}
	return string(program.Text[start:end])
}
