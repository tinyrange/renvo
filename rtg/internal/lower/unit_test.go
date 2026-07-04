package lower

import (
	"bytes"
	"testing"

	"j5.nz/rtg/rtg/internal/check"
	"j5.nz/rtg/rtg/internal/load"
	"j5.nz/rtg/rtg/internal/unit"
	"j5.nz/rtg/rtgunit"
)

func TestEmitRootPackageUnit(t *testing.T) {
	graph := loadTestGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

const answer = 42

func appMain() int {
	return answer
}
`)},
	})
	result := EmitRoot(graph)
	if !result.Ok {
		t.Fatalf("EmitRoot failed: err=%d file=%d tok=%d", result.Error, result.ErrorFile, result.ErrorToken)
	}
	data, ok := unit.Marshal(result.Program)
	if !ok {
		t.Fatal("unit Marshal failed")
	}
	decoded, err := rtgunit.Unmarshal(data)
	if err != nil {
		t.Fatalf("host unit decode failed: %v", err)
	}
	if decoded.Package != "main" {
		t.Fatalf("package = %q, want main", decoded.Package)
	}
	if !bytes.Equal(decoded.Text, result.Program.Text) {
		t.Fatalf("decoded text mismatch")
	}
	if len(decoded.Decls) != 1 || decoded.Decls[0].NameStart >= decoded.Decls[0].NameEnd {
		t.Fatalf("decls = %#v", decoded.Decls)
	}
	if string(decoded.Text[decoded.Decls[0].NameStart:decoded.Decls[0].NameEnd]) != "answer" {
		t.Fatalf("decl name = %q", string(decoded.Text[decoded.Decls[0].NameStart:decoded.Decls[0].NameEnd]))
	}
	if len(decoded.Funcs) != 1 {
		t.Fatalf("funcs = %#v", decoded.Funcs)
	}
	fn := decoded.Funcs[0]
	if string(decoded.Text[fn.NameStart:fn.NameEnd]) != "appMain" {
		t.Fatalf("func name = %q", string(decoded.Text[fn.NameStart:fn.NameEnd]))
	}
	if tokenText(decoded, fn.BodyStart) != "{" || tokenText(decoded, fn.BodyEnd) != "}" {
		t.Fatalf("body tokens = %q:%q", tokenText(decoded, fn.BodyStart), tokenText(decoded, fn.BodyEnd))
	}
}

func TestEmitPackagePreservesTextAndFileOrder(t *testing.T) {
	graph := loadTestGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/z.go", Src: []byte("package main\n\nfunc z() int { return 2 }\n")},
		{Path: "/repo/case/cmd/app/a.go", Src: []byte("package main\n\nfunc a() int { return 1 }\n")},
	})
	result := EmitRoot(graph)
	if !result.Ok {
		t.Fatalf("EmitRoot failed: err=%d file=%d tok=%d", result.Error, result.ErrorFile, result.ErrorToken)
	}
	wantText := []byte("package main\n\nfunc a() int { return 1 }\npackage main\n\nfunc z() int { return 2 }\n")
	if !bytes.Equal(result.Program.Text, wantText) {
		t.Fatalf("text = %q, want %q", string(result.Program.Text), string(wantText))
	}
	if len(result.Program.Funcs) != 2 {
		t.Fatalf("func count = %d, want 2", len(result.Program.Funcs))
	}
	if string(result.Program.Text[result.Program.Funcs[0].NameStart:result.Program.Funcs[0].NameEnd]) != "a" {
		t.Fatalf("first func = %q", string(result.Program.Text[result.Program.Funcs[0].NameStart:result.Program.Funcs[0].NameEnd]))
	}
	if string(result.Program.Text[result.Program.Funcs[1].NameStart:result.Program.Funcs[1].NameEnd]) != "z" {
		t.Fatalf("second func = %q", string(result.Program.Text[result.Program.Funcs[1].NameStart:result.Program.Funcs[1].NameEnd]))
	}
}

func TestEmitCheckedPackageUsesCheckedDeclOrder(t *testing.T) {
	graph := loadTestGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

const first = 1
const second = 2

func appMain() int {
	return first + second
}
`)},
	})
	prog := check.CheckGraph(graph)
	if !prog.Ok {
		t.Fatalf("CheckGraph failed: err=%d pkg=%d file=%d tok=%d", prog.Error, prog.ErrorPackage, prog.ErrorFile, prog.ErrorToken)
	}
	info := prog.Packages[0]
	if len(info.DeclOrder) != 2 {
		t.Fatalf("decl order = %#v, want two declarations", info.DeclOrder)
	}
	info.DeclOrder[0], info.DeclOrder[1] = info.DeclOrder[1], info.DeclOrder[0]

	result := EmitCheckedPackage(graph.Packages[0], info)
	if !result.Ok {
		t.Fatalf("EmitCheckedPackage failed: err=%d file=%d tok=%d", result.Error, result.ErrorFile, result.ErrorToken)
	}
	if len(result.Program.Decls) != 2 {
		t.Fatalf("decl count = %d, want 2", len(result.Program.Decls))
	}
	if declName(result.Program, 0) != "second" || declName(result.Program, 1) != "first" {
		t.Fatalf("decl order = %q, %q; want second, first", declName(result.Program, 0), declName(result.Program, 1))
	}
}

func TestEmitCheckedPackageUsesCheckedFunctionOrder(t *testing.T) {
	graph := loadTestGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

func first() int { return 1 }
func second() int { return 2 }
func appMain() int { return first() + second() }
`)},
	})
	prog := check.CheckGraph(graph)
	if !prog.Ok {
		t.Fatalf("CheckGraph failed: err=%d pkg=%d file=%d tok=%d", prog.Error, prog.ErrorPackage, prog.ErrorFile, prog.ErrorToken)
	}
	info := prog.Packages[0]
	first := check.LookupFuncBody(info, "first")
	second := check.LookupFuncBody(info, "second")
	if first < 0 || second < 0 {
		t.Fatalf("missing function bodies in %#v", info.Bodies)
	}
	info.Bodies[0], info.Bodies[1] = info.Bodies[second], info.Bodies[first]

	result := EmitCheckedPackage(graph.Packages[0], info)
	if !result.Ok {
		t.Fatalf("EmitCheckedPackage failed: err=%d file=%d tok=%d", result.Error, result.ErrorFile, result.ErrorToken)
	}
	if len(result.Program.Funcs) != 3 {
		t.Fatalf("func count = %d, want 3", len(result.Program.Funcs))
	}
	if funcName(result.Program, 0) != "second" || funcName(result.Program, 1) != "first" {
		t.Fatalf("func order = %q, %q; want second, first", funcName(result.Program, 0), funcName(result.Program, 1))
	}
}

func TestEmitCheckedPackageExpressionShapes(t *testing.T) {
	graph := loadTestGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

type item struct { value int }

var global = []int{1, 2, 3}
var picked = choose(global[1])

func choose(v int) int { return v }

func appMain() int {
	var typed item
	var local = []item{{value: global[0]}}
	local[0] = item{value: picked}
	return choose(local[0].value)
}
`)},
	})
	prog := check.CheckGraph(graph)
	if !prog.Ok {
		t.Fatalf("CheckGraph failed: err=%d pkg=%d file=%d tok=%d", prog.Error, prog.ErrorPackage, prog.ErrorFile, prog.ErrorToken)
	}
	result := EmitCheckedPackage(graph.Packages[0], prog.Packages[0])
	if !result.Ok {
		t.Fatalf("EmitCheckedPackage failed: err=%d file=%d tok=%d", result.Error, result.ErrorFile, result.ErrorToken)
	}
	if len(result.Program.Indexes) != 4 {
		t.Fatalf("indexes = %#v, want 4", result.Program.Indexes)
	}
	if len(result.Program.Composites) != 3 {
		t.Fatalf("composites = %#v, want 3", result.Program.Composites)
	}
	if len(result.Program.Assigns) != 1 {
		t.Fatalf("assignments = %#v, want 1", result.Program.Assigns)
	}
	if len(result.Program.Returns) != 2 {
		t.Fatalf("returns = %#v, want 2", result.Program.Returns)
	}
	if len(result.Program.Calls) != 2 {
		t.Fatalf("calls = %#v, want 2", result.Program.Calls)
	}
	if len(result.Program.Types) != 1 {
		t.Fatalf("types = %#v, want 1", result.Program.Types)
	}
	if len(result.Program.Signatures) != len(result.Program.Funcs) {
		t.Fatalf("signatures = %#v, want %d", result.Program.Signatures, len(result.Program.Funcs))
	}
	if len(result.Program.TypeRefs) == 0 {
		t.Fatalf("type refs = %#v, want refs", result.Program.TypeRefs)
	}
	if len(result.Program.Locals) != 2 {
		t.Fatalf("locals = %#v, want 2", result.Program.Locals)
	}
	if len(result.Program.Refs) == 0 {
		t.Fatalf("refs = %#v, want refs", result.Program.Refs)
	}
	item := findUnitDecl(result.Program, "item")
	global := findUnitDecl(result.Program, "global")
	picked := findUnitDecl(result.Program, "picked")
	appMain := findUnitFunc(result.Program, "appMain")
	choose := findUnitFunc(result.Program, "choose")
	if item < 0 || global < 0 || picked < 0 || appMain < 0 || choose < 0 {
		t.Fatalf("unit rows missing: decls=%#v funcs=%#v", result.Program.Decls, result.Program.Funcs)
	}
	assertUnitType(t, result.Program, "item", unit.TypeStruct, item, "struct { value int }", "", "", "")
	assertUnitSignature(t, result.Program, choose, nil, []string{"v:int"}, []string{":int"})
	assertUnitSignature(t, result.Program, appMain, nil, nil, []string{":int"})
	assertUnitComposite(t, result.Program, unit.OwnerDecl, global, "[]int", []string{"1", "2", "3"})
	assertUnitIndex(t, result.Program, unit.OwnerDecl, picked, "global", "1")
	assertUnitCall(t, result.Program, unit.OwnerDecl, picked, unit.CallPackage, "", "choose", []string{"global[1]"})
	assertUnitRef(t, result.Program, unit.OwnerDecl, picked, unit.RefPackage, "choose")
	assertUnitRef(t, result.Program, unit.OwnerDecl, picked, unit.RefPackage, "global")
	assertUnitComposite(t, result.Program, unit.OwnerFunc, appMain, "[]item", []string{"{value: global[0]}"})
	assertUnitComposite(t, result.Program, unit.OwnerFunc, appMain, "item", []string{"value: picked"})
	assertUnitTypeRef(t, result.Program, unit.OwnerDecl, item, unit.TypeRefBuiltin, "", "int")
	assertUnitTypeRef(t, result.Program, unit.OwnerFunc, choose, unit.TypeRefBuiltin, "", "int")
	assertUnitTypeRef(t, result.Program, unit.OwnerFunc, appMain, unit.TypeRefPackage, "", "item")
	assertUnitLocal(t, result.Program, appMain, unit.TokenVar, "typed", "item", "", nil, false)
	assertUnitLocal(t, result.Program, appMain, unit.TokenVar, "local", "", "[]item{{value: global[0]}}", []string{"[]item{{value: global[0]}}"}, false)
	assertUnitIndex(t, result.Program, unit.OwnerFunc, appMain, "global", "0")
	assertUnitIndex(t, result.Program, unit.OwnerFunc, appMain, "local", "0")
	assertUnitAssign(t, result.Program, appMain, unit.AssignSet, "local[0]", "item{value: picked}")
	assertUnitReturn(t, result.Program, choose, []string{"v"})
	assertUnitReturn(t, result.Program, appMain, []string{"choose(local[0].value)"})
	assertUnitCall(t, result.Program, unit.OwnerFunc, appMain, unit.CallPackage, "", "choose", []string{"local[0].value"})
	assertUnitRef(t, result.Program, unit.OwnerFunc, appMain, unit.RefPackage, "choose")
	assertUnitRef(t, result.Program, unit.OwnerFunc, appMain, unit.RefScope, "local")

	data, ok := unit.Marshal(result.Program)
	if !ok {
		t.Fatal("unit Marshal failed")
	}
	decoded, err := rtgunit.Unmarshal(data)
	if err != nil {
		t.Fatalf("host unit decode failed: %v", err)
	}
	if len(decoded.Indexes) != len(result.Program.Indexes) || len(decoded.Composites) != len(result.Program.Composites) {
		t.Fatalf("decoded shapes = %d/%d, want %d/%d", len(decoded.Indexes), len(decoded.Composites), len(result.Program.Indexes), len(result.Program.Composites))
	}
	if len(decoded.Assigns) != len(result.Program.Assigns) || len(decoded.Returns) != len(result.Program.Returns) {
		t.Fatalf("decoded flow = %d/%d, want %d/%d", len(decoded.Assigns), len(decoded.Returns), len(result.Program.Assigns), len(result.Program.Returns))
	}
	if len(decoded.Calls) != len(result.Program.Calls) {
		t.Fatalf("decoded calls = %d, want %d", len(decoded.Calls), len(result.Program.Calls))
	}
	if len(decoded.Types) != len(result.Program.Types) {
		t.Fatalf("decoded types = %d, want %d", len(decoded.Types), len(result.Program.Types))
	}
	if len(decoded.Signatures) != len(result.Program.Signatures) {
		t.Fatalf("decoded signatures = %d, want %d", len(decoded.Signatures), len(result.Program.Signatures))
	}
	if len(decoded.TypeRefs) != len(result.Program.TypeRefs) {
		t.Fatalf("decoded type refs = %d, want %d", len(decoded.TypeRefs), len(result.Program.TypeRefs))
	}
	if len(decoded.Locals) != len(result.Program.Locals) {
		t.Fatalf("decoded locals = %d, want %d", len(decoded.Locals), len(result.Program.Locals))
	}
	if len(decoded.Refs) != len(result.Program.Refs) || len(decoded.Selectors) != len(result.Program.Selectors) {
		t.Fatalf("decoded resolution = %d/%d, want %d/%d", len(decoded.Refs), len(decoded.Selectors), len(result.Program.Refs), len(result.Program.Selectors))
	}
}

func TestEmitCheckedPackageRejectsInvalidCheckedMetadata(t *testing.T) {
	graph := loadTestGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

const first = 1
const second = 2
`)},
	})
	prog := check.CheckGraph(graph)
	if !prog.Ok {
		t.Fatalf("CheckGraph failed: err=%d pkg=%d file=%d tok=%d", prog.Error, prog.ErrorPackage, prog.ErrorFile, prog.ErrorToken)
	}
	info := prog.Packages[0]
	info.DeclOrder[1] = info.DeclOrder[0]

	result := EmitCheckedPackage(graph.Packages[0], info)
	if result.Ok || result.Error != EmitErrCheck {
		t.Fatalf("invalid metadata result = %#v", result)
	}
}

func TestEmitPackagePreservesLinkStaticDirective(t *testing.T) {
	graph := loadTestGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

// rtg:linkstatic libc,puts
func puts(s string) int { return 0 }

func appMain() int { return puts("PASS\n") }
`)},
	})
	result := EmitRoot(graph)
	if !result.Ok {
		t.Fatalf("EmitRoot failed: err=%d file=%d tok=%d", result.Error, result.ErrorFile, result.ErrorToken)
	}
	if !bytes.Contains(result.Program.Text, []byte("// rtg:linkstatic libc,puts\nfunc puts")) {
		t.Fatalf("linkstatic directive was not preserved: %q", string(result.Program.Text))
	}
	data, ok := unit.Marshal(result.Program)
	if !ok {
		t.Fatal("unit Marshal failed")
	}
	decoded, err := rtgunit.Unmarshal(data)
	if err != nil {
		t.Fatalf("host unit decode failed: %v", err)
	}
	if !bytes.Contains(decoded.Text, []byte("// rtg:linkstatic libc,puts\nfunc puts")) {
		t.Fatalf("decoded text lost directive: %q", string(decoded.Text))
	}
}

func TestEmitTokenKinds(t *testing.T) {
	graph := loadTestGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

const whole = 1
const fractional = 1.5
const imported = import

func appMain() int {
	if whole > 0 {
		return whole
	}
	return 0
}
`)},
	})
	result := EmitRoot(graph)
	if !result.Ok {
		t.Fatalf("EmitRoot failed: err=%d file=%d tok=%d", result.Error, result.ErrorFile, result.ErrorToken)
	}
	foundFloat := false
	foundImportIdent := false
	foundIf := false
	for i := 0; i < len(result.Program.Tokens); i++ {
		tok := result.Program.Tokens[i]
		text := string(result.Program.Text[tok.Start : tok.Start+tok.Size])
		if text == "1.5" && tok.Kind == unit.TokenFloat {
			foundFloat = true
		}
		if text == "import" && tok.Kind == unit.TokenIdent {
			foundImportIdent = true
		}
		if text == "if" && tok.Kind == unit.TokenIf {
			foundIf = true
		}
	}
	if !foundFloat {
		t.Fatal("float literal was not emitted as unit.TokenFloat")
	}
	if !foundImportIdent {
		t.Fatal("unsupported import keyword was not downgraded to unit.TokenIdent")
	}
	if !foundIf {
		t.Fatal("if keyword was not emitted as unit.TokenIf")
	}
}

func TestEmitPackageRejectsInvalidPackage(t *testing.T) {
	result := EmitPackage(load.Package{})
	if result.Ok || result.Error != EmitErrPackage {
		t.Fatalf("empty package result = %#v", result)
	}
}

func loadTestGraph(t *testing.T, files []load.SourceFile) load.Graph {
	t.Helper()
	mod := load.Module{Root: "/repo/case", Path: "example.com/case", Ok: true}
	graph := load.LoadGraph(mod, "/std", "/repo/case", "./cmd/app", files)
	if !graph.Ok {
		t.Fatalf("LoadGraph failed: err=%d pkg=%d graph=%#v", graph.Error, graph.ErrorPackage, graph)
	}
	return graph
}

func tokenText(program rtgunit.Program, index int) string {
	if index < 0 || index*8+8 > len(program.Tokens) {
		return ""
	}
	pos := index * 8
	start := int(program.Tokens[pos+1]) | int(program.Tokens[pos+2])<<8 | int(program.Tokens[pos+3])<<16
	size := int(program.Tokens[pos+4])
	if int(program.Tokens[pos]) != unit.TokenOp {
		size = size | int(program.Tokens[pos+5])<<8
	}
	if start < 0 || size < 0 || start+size > len(program.Text) {
		return ""
	}
	return string(program.Text[start : start+size])
}

func declName(program unit.Program, index int) string {
	if index < 0 || index >= len(program.Decls) {
		return ""
	}
	decl := program.Decls[index]
	if decl.NameStart < 0 || decl.NameEnd > len(program.Text) || decl.NameEnd < decl.NameStart {
		return ""
	}
	return string(program.Text[decl.NameStart:decl.NameEnd])
}

func funcName(program unit.Program, index int) string {
	if index < 0 || index >= len(program.Funcs) {
		return ""
	}
	fn := program.Funcs[index]
	if fn.NameStart < 0 || fn.NameEnd > len(program.Text) || fn.NameEnd < fn.NameStart {
		return ""
	}
	return string(program.Text[fn.NameStart:fn.NameEnd])
}

func findUnitDecl(program unit.Program, name string) int {
	for i := 0; i < len(program.Decls); i++ {
		if declName(program, i) == name {
			return i
		}
	}
	return -1
}

func findUnitFunc(program unit.Program, name string) int {
	for i := 0; i < len(program.Funcs); i++ {
		if funcName(program, i) == name {
			return i
		}
	}
	return -1
}

func assertUnitIndex(t *testing.T, program unit.Program, ownerKind int, ownerIndex int, base string, indexText string) {
	t.Helper()
	for i := 0; i < len(program.Indexes); i++ {
		index := program.Indexes[i]
		if index.OwnerKind != ownerKind || index.OwnerIndex != ownerIndex {
			continue
		}
		if unitSpanText(program, index.BaseStart, index.BaseEnd) == base && unitSpanText(program, index.IndexStart, index.IndexEnd) == indexText {
			return
		}
	}
	t.Fatalf("index owner=%d/%d %s[%s] not found in %#v", ownerKind, ownerIndex, base, indexText, program.Indexes)
}

func assertUnitComposite(t *testing.T, program unit.Program, ownerKind int, ownerIndex int, typ string, elems []string) {
	t.Helper()
	for i := 0; i < len(program.Composites); i++ {
		composite := program.Composites[i]
		if composite.OwnerKind != ownerKind || composite.OwnerIndex != ownerIndex {
			continue
		}
		if unitSpanText(program, composite.TypeStart, composite.TypeEnd) == typ {
			if len(composite.Elems) != len(elems) {
				t.Fatalf("composite %s elems = %#v, want %v", typ, composite.Elems, elems)
			}
			for j := 0; j < len(elems); j++ {
				if got := unitSpanText(program, composite.Elems[j].StartTok, composite.Elems[j].EndTok); got != elems[j] {
					t.Fatalf("composite %s elem %d = %q, want %q", typ, j, got, elems[j])
				}
			}
			return
		}
	}
	t.Fatalf("composite owner=%d/%d %s not found in %#v", ownerKind, ownerIndex, typ, program.Composites)
}

func assertUnitType(t *testing.T, program unit.Program, name string, kind int, decl int, typeText string, lenText string, keyText string, elemText string) {
	t.Helper()
	for i := 0; i < len(program.Types); i++ {
		typ := program.Types[i]
		if unitText(program, typ.NameStart, typ.NameEnd) != name || typ.Kind != kind || typ.Decl != decl {
			continue
		}
		if unitSpanText(program, typ.TypeStart, typ.TypeEnd) != typeText {
			t.Fatalf("type %s span = %q, want %q", name, unitSpanText(program, typ.TypeStart, typ.TypeEnd), typeText)
		}
		if unitSpanText(program, typ.LenStart, typ.LenEnd) != lenText {
			t.Fatalf("type %s length span = %q, want %q", name, unitSpanText(program, typ.LenStart, typ.LenEnd), lenText)
		}
		if unitSpanText(program, typ.KeyStart, typ.KeyEnd) != keyText {
			t.Fatalf("type %s key span = %q, want %q", name, unitSpanText(program, typ.KeyStart, typ.KeyEnd), keyText)
		}
		if unitSpanText(program, typ.ElemStart, typ.ElemEnd) != elemText {
			t.Fatalf("type %s element span = %q, want %q", name, unitSpanText(program, typ.ElemStart, typ.ElemEnd), elemText)
		}
		return
	}
	t.Fatalf("type name=%s kind=%d decl=%d not found in %#v", name, kind, decl, program.Types)
}

func assertUnitSignature(t *testing.T, program unit.Program, funcIndex int, receiver []string, params []string, results []string) {
	t.Helper()
	for i := 0; i < len(program.Signatures); i++ {
		sig := program.Signatures[i]
		if sig.FuncIndex != funcIndex {
			continue
		}
		assertUnitFields(t, program, sig.Receiver, receiver)
		assertUnitFields(t, program, sig.Params, params)
		assertUnitFields(t, program, sig.Results, results)
		return
	}
	t.Fatalf("signature func=%d not found in %#v", funcIndex, program.Signatures)
}

func assertUnitFields(t *testing.T, program unit.Program, fields []unit.Field, want []string) {
	t.Helper()
	if len(fields) != len(want) {
		t.Fatalf("fields = %#v, want %v", fields, want)
	}
	for i := 0; i < len(want); i++ {
		got := tokenTextUnit(program, fields[i].NameTok) + ":" + unitSpanText(program, fields[i].TypeStart, fields[i].TypeEnd)
		if got != want[i] {
			t.Fatalf("field %d = %q, want %q in %#v", i, got, want[i], fields)
		}
	}
}

func assertUnitTypeRef(t *testing.T, program unit.Program, ownerKind int, ownerIndex int, kind int, base string, name string) {
	t.Helper()
	for i := 0; i < len(program.TypeRefs); i++ {
		ref := program.TypeRefs[i]
		if ref.OwnerKind == ownerKind && ref.OwnerIndex == ownerIndex && ref.Kind == kind &&
			tokenTextUnit(program, ref.BaseTok) == base && tokenTextUnit(program, ref.Token) == name {
			return
		}
	}
	t.Fatalf("type ref owner=%d/%d kind=%d %s.%s not found in %#v", ownerKind, ownerIndex, kind, base, name, program.TypeRefs)
}

func assertUnitLocal(t *testing.T, program unit.Program, funcIndex int, kind int, name string, typ string, value string, values []string, alias bool) {
	t.Helper()
	for i := 0; i < len(program.Locals); i++ {
		local := program.Locals[i]
		if local.FuncIndex != funcIndex || local.Kind != kind || unitText(program, local.NameStart, local.NameEnd) != name || local.Alias != alias {
			continue
		}
		if unitSpanText(program, local.TypeStart, local.TypeEnd) != typ {
			t.Fatalf("local %s type = %q, want %q", name, unitSpanText(program, local.TypeStart, local.TypeEnd), typ)
		}
		if unitSpanText(program, local.ValueStart, local.ValueEnd) != value {
			t.Fatalf("local %s value = %q, want %q", name, unitSpanText(program, local.ValueStart, local.ValueEnd), value)
		}
		if len(local.Values) != len(values) {
			t.Fatalf("local %s values = %#v, want %v", name, local.Values, values)
		}
		for j := 0; j < len(values); j++ {
			if got := unitSpanText(program, local.Values[j].StartTok, local.Values[j].EndTok); got != values[j] {
				t.Fatalf("local %s value %d = %q, want %q", name, j, got, values[j])
			}
		}
		return
	}
	t.Fatalf("local func=%d kind=%d name=%s not found in %#v", funcIndex, kind, name, program.Locals)
}

func assertUnitAssign(t *testing.T, program unit.Program, funcIndex int, kind int, left string, right string) {
	t.Helper()
	for i := 0; i < len(program.Assigns); i++ {
		assign := program.Assigns[i]
		if assign.FuncIndex != funcIndex || assign.Kind != kind {
			continue
		}
		if unitSpanText(program, assign.LeftStart, assign.LeftEnd) == left && unitSpanText(program, assign.RightStart, assign.RightEnd) == right {
			return
		}
	}
	t.Fatalf("assignment func=%d kind=%d %s = %s not found in %#v", funcIndex, kind, left, right, program.Assigns)
}

func assertUnitReturn(t *testing.T, program unit.Program, funcIndex int, values []string) {
	t.Helper()
	for i := 0; i < len(program.Returns); i++ {
		ret := program.Returns[i]
		if ret.FuncIndex != funcIndex {
			continue
		}
		if len(ret.Values) != len(values) {
			continue
		}
		matched := true
		for j := 0; j < len(values); j++ {
			if unitSpanText(program, ret.Values[j].StartTok, ret.Values[j].EndTok) != values[j] {
				matched = false
			}
		}
		if matched {
			return
		}
	}
	t.Fatalf("return func=%d values=%v not found in %#v", funcIndex, values, program.Returns)
}

func assertUnitCall(t *testing.T, program unit.Program, ownerKind int, ownerIndex int, kind int, base string, callee string, args []string) {
	t.Helper()
	for i := 0; i < len(program.Calls); i++ {
		call := program.Calls[i]
		if call.OwnerKind != ownerKind || call.OwnerIndex != ownerIndex || call.Kind != kind {
			continue
		}
		if tokenTextUnit(program, call.BaseTok) != base || tokenTextUnit(program, call.CalleeTok) != callee {
			continue
		}
		if len(call.Args) != len(args) {
			continue
		}
		matched := true
		for j := 0; j < len(args); j++ {
			if unitSpanText(program, call.Args[j].StartTok, call.Args[j].EndTok) != args[j] {
				matched = false
			}
		}
		if matched {
			return
		}
	}
	t.Fatalf("call owner=%d/%d %s.%s args=%v not found in %#v", ownerKind, ownerIndex, base, callee, args, program.Calls)
}

func assertUnitRef(t *testing.T, program unit.Program, ownerKind int, ownerIndex int, kind int, name string) {
	t.Helper()
	for i := 0; i < len(program.Refs); i++ {
		ref := program.Refs[i]
		if ref.OwnerKind == ownerKind && ref.OwnerIndex == ownerIndex && ref.Kind == kind && tokenTextUnit(program, ref.Token) == name {
			return
		}
	}
	t.Fatalf("ref owner=%d/%d kind=%d name=%s not found in %#v", ownerKind, ownerIndex, kind, name, program.Refs)
}

func tokenTextUnit(program unit.Program, tok int) string {
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

func unitText(program unit.Program, start int, end int) string {
	if start < 0 || end < start || end > len(program.Text) {
		return ""
	}
	return string(program.Text[start:end])
}

func unitSpanText(program unit.Program, startTok int, endTok int) string {
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
