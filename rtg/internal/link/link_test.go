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
	if linked.Program.ImportPath != "example.com/case/cmd/app" {
		t.Fatalf("linked import path = %q", linked.Program.ImportPath)
	}
	if len(linked.Program.Imports) != 0 {
		t.Fatalf("linked imports = %#v, want none", linked.Program.Imports)
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
	if decoded.ImportPath != "example.com/case/cmd/app" || len(decoded.Imports) != 0 {
		t.Fatalf("decoded imports = %q %#v", decoded.ImportPath, decoded.Imports)
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

func TestLinkUnitsMapsDependencyEOFToPackageBoundary(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import "example.com/case/pkg/lib"

func appMain() int { var value lib.Final; return len(value) }
`)},
		{Path: "/repo/case/pkg/lib/lib.go", Src: []byte(`package lib

type Final []byte
`)},
	})
	program, ok := LinkUnits(result.Units, result.Root)
	if !ok {
		t.Fatal("LinkUnits failed")
	}
	if len(program.Decls) != 1 {
		t.Fatalf("linked declarations = %#v, want dependency type", program.Decls)
	}
	decl := program.Decls[0]
	if decl.EndTok >= len(program.Tokens)-1 {
		t.Fatalf("dependency declaration ends at linked EOF %d", decl.EndTok)
	}
	if got := linkedTokenText(program, decl.EndTok); got != "package" {
		t.Fatalf("dependency declaration boundary = %q, want next package", got)
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

func TestLinkUnitsManglesDuplicatePackageSymbols(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import "example.com/case/pkg/a"
import "example.com/case/pkg/b"

func appMain() int { return a.Value()+b.Value() }
`)},
		{Path: "/repo/case/pkg/a/a.go", Src: []byte(`package a

func Value() int { return 1 }
`)},
		{Path: "/repo/case/pkg/b/b.go", Src: []byte(`package b

func Value() int { return 2 }
`)},
	})
	program, ok := LinkUnits(result.Units, result.Root)
	if !ok {
		t.Fatal("LinkUnits failed")
	}
	left := findLinkedFunc(program, "rtgp0_Value")
	right := findLinkedFunc(program, "rtgp1_Value")
	appMain := findLinkedFunc(program, "appMain")
	if left < 0 || right < 0 || appMain < 0 {
		t.Fatalf("linked funcs missing aliases: %#v", program.Funcs)
	}
	assertLinkedStatement(t, program, appMain, unit.StmtReturn, "rtgp0_Value()+rtgp1_Value()")
	for i := 0; i < len(program.Refs); i++ {
		if program.Refs[i].Kind == unit.RefImport {
			t.Fatalf("linked import ref survived: %#v", program.Refs[i])
		}
	}
}

func TestLinkUnitsPreservesWhitespaceBeforeImportedTypeSelector(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import "example.com/case/pkg/lib"

func Use(graph lib.Graph) int { return graph.Value }
func appMain() int { return Use(lib.Graph{Value: 1}) }
`)},
		{Path: "/repo/case/pkg/lib/lib.go", Src: []byte(`package lib

type Graph struct { Value int }
`)},
	})
	program, ok := LinkUnits(result.Units, result.Root)
	if !ok {
		t.Fatal("LinkUnits failed")
	}
	if bytes.Contains(program.Text, []byte("graphGraph")) {
		t.Fatalf("linked text joined parameter name and imported type:\n%s", string(program.Text))
	}
	if !bytes.Contains(program.Text, []byte("Use(graph Graph)")) {
		t.Fatalf("linked text missing rewritten imported type parameter:\n%s", string(program.Text))
	}
}

func TestLinkBuildCorePreservesImportedTypeAliasMethodIdentity(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import "example.com/case/pkg/lib"

func appMain() {
	var image *lib.Image = lib.NewSurface()
	image.Destroy()
}
`)},
		{Path: "/repo/case/pkg/lib/lib.go", Src: []byte(`package lib

type Surface struct { Value int }
type Image = Surface

func NewSurface() *Surface { return &Surface{Value: 1} }
func (s *Surface) Destroy() { s.Value = 0 }
`)},
	})
	linked := LinkBuildCore(result)
	if !linked.Ok {
		t.Fatalf("LinkBuildCore failed: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	for _, want := range []string{
		"type Image = Surface",
		"func (s *Surface) Destroy()",
		"var image *Image = NewSurface()",
		"image.Destroy()",
	} {
		if !bytes.Contains(linked.Program.Text, []byte(want)) {
			t.Fatalf("linked source missing %q:\n%s", want, string(linked.Program.Text))
		}
	}
}

func TestLinkUnitsErasesUnsafePointerRoundTrip(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/std/unsafe/unsafe.go", Src: []byte(`package unsafe

type Pointer *byte
`)},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import "unsafe"

type pair struct {
	a int32
	b int32
}

func appMain() int {
	v := pair{a: 9, b: 5}
	p := unsafe.Pointer(&v)
	q := (*pair)(p)
	if int(q.a)+int(q.b) == 14 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
`)},
	})
	program, ok := LinkUnits(result.Units, result.Root)
	if !ok {
		t.Fatal("LinkUnits failed")
	}
	if bytes.Contains(program.Text, []byte("Pointer(&v)")) || bytes.Contains(program.Text, []byte("(*pair)(p)")) {
		t.Fatalf("linked text still contains unsafe pointer conversions:\n%s", string(program.Text))
	}
	if !bytes.Contains(program.Text, []byte("p := &v")) || !bytes.Contains(program.Text, []byte("q :=p")) {
		t.Fatalf("linked text missing erased unsafe pointer values:\n%s", string(program.Text))
	}
	linked := LinkBuild(result)
	if !linked.Ok {
		t.Fatalf("LinkBuild failed: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	if _, err := rtgunit.Unmarshal(linked.Data); err != nil {
		t.Fatalf("linked unit did not decode: %v", err)
	}
}

func TestLinkUnitsCoreLowersUnresolvedUnsafeSizeof(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/std/unsafe/unsafe.go", Src: []byte(`package unsafe

type Pointer *byte
`)},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import u "unsafe"

func appMain() int {
	var value int
	return int(u.Sizeof(value))
}
`)},
	})
	result.Units[result.Root].Program.Selectors = nil
	result.Units[result.Root].Program.Calls = nil
	program, ok := LinkUnitsCore(result.Units, result.Root)
	if !ok {
		t.Fatal("LinkUnitsCore failed")
	}
	if bytes.Contains(program.Text, []byte("u.Sizeof")) || !bytes.Contains(program.Text, []byte("Sizeof(value)")) {
		t.Fatalf("linked source did not lower unsafe.Sizeof:\n%s", string(program.Text))
	}
}

func TestLinkUnitsPreservesClosureAdder(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

func makeAdder(base int) func(int) int {
	return func(v int) int {
		return base + v
	}
}

func appMain() int {
	add := makeAdder(0)
	if add(0) == 0 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
`)},
	})
	program, ok := LinkUnits(result.Units, result.Root)
	if !ok {
		t.Fatal("LinkUnits failed")
	}
	if !bytes.Contains(program.Text, []byte("func makeAdder(base int) func(int) int")) ||
		!bytes.Contains(program.Text, []byte("return func(v int) int")) ||
		!bytes.Contains(program.Text, []byte("add(0) == 0")) {
		t.Fatalf("linked text did not preserve closure semantics:\n%s", string(program.Text))
	}
	makeAdder := findLinkedFunc(program, "makeAdder")
	if makeAdder < 0 {
		t.Fatalf("makeAdder not found in %#v", program.Funcs)
	}
	linked := LinkBuild(result)
	if !linked.Ok {
		t.Fatalf("LinkBuild failed: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	if _, err := rtgunit.Unmarshal(linked.Data); err != nil {
		t.Fatalf("linked unit did not decode: %v", err)
	}
}

func TestLinkBuildCorePreservesFunctionValue(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

func add(a int, b int) int {
	return a + b
}

func mul(a int, b int) int {
	return a * b
}

func choose(fn func(int, int) int, a int, b int) int {
	return fn(a, b)
}

func appMain() int {
	fn := add
	if 1%2 == 1 {
		fn = mul
	}
	if choose(fn, 3, 4) == 12 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
`)},
	})
	linked := LinkBuildCore(result)
	if !linked.Ok {
		t.Fatalf("LinkBuildCore failed: err=%d pkg=%d marshal=%d/%d", linked.Error, linked.ErrorPackage, unit.LastMarshalError, unit.LastMarshalIndex)
	}
	decoded, ok := unit.Unmarshal(linked.Data)
	if !ok {
		t.Fatal("linked core unit did not decode")
	}
	if !bytes.Contains(decoded.Text, []byte("fn func(int, int) int")) ||
		!bytes.Contains(decoded.Text, []byte("fn := add")) ||
		!bytes.Contains(decoded.Text, []byte("fn = mul")) ||
		!bytes.Contains(decoded.Text, []byte("choose(fn, 3, 4)")) {
		t.Fatalf("linked text did not preserve function-value semantics:\n%s", string(decoded.Text))
	}
}

func TestLinkBuildCoreLowersBoundCallbackField(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/widgets/widgets.go", Src: []byte(`package widgets

type Event struct { X int }
type ClickHandler func(sender *Button, event Event)
type Button struct { Click ClickHandler }

func (button *Button) Dispatch(event Event) {
	if button.Click != nil {
		button.Click(button, event)
	}
}
`)},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import "example.com/case/widgets"

type form struct {
	button widgets.Button
	total int
}

func (f *form) clicked(sender *widgets.Button, event widgets.Event) {
	f.total += event.X
}

func main() {
	f := &form{}
	f.button.Click = f.clicked
	f.button.Dispatch(widgets.Event{X: 42})
	f.button.Click = nil
	if f.total == 42 && f.button.Click == nil { print("PASS\n"); return }
}
`)},
	})
	linked := LinkBuildCore(result)
	if !linked.Ok {
		t.Fatalf("LinkBuildCore failed: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	for _, forbidden := range [][]byte{
		[]byte("type ClickHandler func"),
		[]byte("= f.clicked"),
		[]byte("button.Click(button"),
	} {
		if bytes.Contains(linked.Program.Text, forbidden) {
			t.Fatalf("linked backend subset still contains %q:\n%s", forbidden, string(linked.Program.Text))
		}
	}
	if _, ok := unit.Unmarshal(linked.Data); !ok {
		t.Fatal("linked callback unit did not decode")
	}
}

func TestLinkBuildCoreLowersEscapingCapturedClosure(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

type counter struct { step func(int) int }

func newCounter(initial int) *counter {
	value := initial
	return &counter{step: func(delta int) int {
		value += delta
		return value
	}}
}

func main() {
	c := newCounter(10)
	if c.step(12) == 22 && c.step(20) == 42 { print("PASS\n"); return }
}
`)},
	})
	linked := LinkBuildCore(result)
	if !linked.Ok {
		t.Fatalf("LinkBuildCore failed: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	for _, forbidden := range [][]byte{
		[]byte("step func(int) int"),
		[]byte("func(delta int) int"),
		[]byte("c.step(12)"),
	} {
		if bytes.Contains(linked.Program.Text, forbidden) {
			t.Fatalf("linked backend subset still contains %q:\n%s", forbidden, string(linked.Program.Text))
		}
	}
	if _, ok := unit.Unmarshal(linked.Data); !ok {
		t.Fatal("linked closure unit did not decode")
	}
}

func TestLinkBuildCoreKeepsMethodThatSharesCallbackFieldName(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

type Surface struct{}
type PaintHandler func(surface *Surface)
type Control struct { Paint PaintHandler }
type Form struct{}
func (f *Form) Paint(surface *Surface) bool { return surface != nil }
type MainForm struct { Form }

func NewForm() *MainForm { return &MainForm{} }
func (c *Control) paint(surface *Surface) {}

func appMain() int {
	form := NewForm()
	if !form.Paint(&Surface{}) { return 1 }
	control := &Control{}
	control.Paint = control.paint
	control.Paint(&Surface{})
	return 0
}
`)},
	})
	linked := LinkBuildCore(result)
	if !linked.Ok {
		t.Fatalf("LinkBuildCore failed: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	if !bytes.Contains(linked.Program.Text, []byte("form.Paint(&Surface{})")) {
		t.Fatalf("ordinary method sharing callback field name was rewritten:\n%s", linked.Program.Text)
	}
	if bytes.Contains(linked.Program.Text, []byte("control.Paint(&Surface{})")) {
		t.Fatalf("callback field call was not lowered:\n%s", linked.Program.Text)
	}
}

func TestLinkBuildCorePreservesSearchClosure(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/sort/sort.go", Src: []byte(`package sort

func Search(n int, threshold int) int {
	if threshold > n {
		return n
	}
	return threshold
}
`)},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import "example.com/case/sort"

func main() {
	idx := sort.Search(5, func(i int) bool { return i >= 3 })
	if idx == 3 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)},
	})
	linked := LinkBuildCore(result)
	if !linked.Ok {
		t.Fatalf("LinkBuildCore failed: err=%d pkg=%d marshal=%d/%d", linked.Error, linked.ErrorPackage, unit.LastMarshalError, unit.LastMarshalIndex)
	}
	decoded, ok := unit.Unmarshal(linked.Data)
	if !ok {
		t.Fatal("linked core unit did not decode")
	}
	if !bytes.Contains(decoded.Text, []byte("func(i int) bool")) ||
		!bytes.Contains(decoded.Text, []byte("return i >= 3")) ||
		!bytes.Contains(decoded.Text, []byte("Search(5, func")) {
		t.Fatalf("linked text did not preserve search closure:\n%s", string(decoded.Text))
	}
	searchFn := findLinkedFunc(decoded, "Search")
	mainFn := findLinkedFunc(decoded, "main")
	appMainFn := findLinkedFunc(decoded, "appMain")
	if searchFn < 0 || mainFn < 0 || appMainFn < 0 {
		t.Fatalf("missing linked functions: %#v", decoded.Funcs)
	}
	if decoded.Funcs[searchFn].EndTok > decoded.Funcs[mainFn].StartTok {
		t.Fatalf("Search function spans into main: search=%#v main=%#v", decoded.Funcs[searchFn], decoded.Funcs[mainFn])
	}
	if decoded.Funcs[mainFn].ReceiverStart != 0 || decoded.Funcs[mainFn].ReceiverEnd != 0 {
		t.Fatalf("main has noncanonical no-receiver span: %#v", decoded.Funcs[mainFn])
	}
	if decoded.Funcs[appMainFn].ReceiverStart != 0 || decoded.Funcs[appMainFn].ReceiverEnd != 0 {
		t.Fatalf("appMain has noncanonical no-receiver span: %#v", decoded.Funcs[appMainFn])
	}
}

func TestLinkBuildCorePreservesStringIntMapSemantics(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

func appMain() int {
	m := map[string]int{"a": 1, "b": 2}
	m["a"] = m["a"] + m["b"]
	if m["a"] == 3 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
`)},
	})
	linked := LinkBuildCore(result)
	if !linked.Ok {
		t.Fatalf("LinkBuildCore failed: err=%d pkg=%d marshal=%d/%d", linked.Error, linked.ErrorPackage, unit.LastMarshalError, unit.LastMarshalIndex)
	}
	decoded, ok := unit.Unmarshal(linked.Data)
	if !ok {
		t.Fatal("linked core unit did not decode")
	}
	if !bytes.Contains(decoded.Text, []byte(`map[string]int{"a": 1, "b": 2}`)) ||
		!bytes.Contains(decoded.Text, []byte(`m["a"] = m["a"] + m["b"]`)) ||
		!bytes.Contains(decoded.Text, []byte(`if m["a"] == 3`)) {
		t.Fatalf("linked text did not preserve map semantics:\\n%s", string(decoded.Text))
	}
}

func TestLinkBuildCoreSplitsEllipsisTokens(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

func Join(sep []byte) []byte {
	var out []byte
	out = append(out, sep...)
	return out
}

func appMain() int {
	joined := Join([]byte("x"))
	if string(joined) == "x" {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
`)},
	})
	linked := LinkBuildCore(result)
	if !linked.Ok {
		t.Fatalf("LinkBuildCore failed: err=%d pkg=%d marshal=%d/%d", linked.Error, linked.ErrorPackage, unit.LastMarshalError, unit.LastMarshalIndex)
	}
	decoded, ok := unit.Unmarshal(linked.Data)
	if !ok {
		t.Fatal("linked core unit did not decode")
	}
	for i := 0; i < len(decoded.Tokens); i++ {
		if tokenAt(decoded, i) == "..." {
			t.Fatalf("linked core unit kept ellipsis token at %d", i)
		}
	}
	found := false
	for i := 0; i+2 < len(decoded.Tokens); i++ {
		if tokenAt(decoded, i) == "." && tokenAt(decoded, i+1) == "." && tokenAt(decoded, i+2) == "." {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("linked core unit missing split ellipsis tokens:\n%s", string(decoded.Text))
	}
}

func TestLinkBuildCoreKeepsEllipsisSplitAfterFunctionValueReparse(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

type transform func(int) int

func apply(fn transform, value int) int {
	return fn(value)
}

func join(src []byte) []byte {
	var out []byte
	out = append(out, src...)
	return out
}

func appMain() int {
	inc := func(value int) int { return value + 1 }
	joined := join([]byte("x"))
	if apply(inc, 1) == 2 && string(joined) == "x" {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
`)},
	})
	linked := LinkBuildCore(result)
	if !linked.Ok {
		t.Fatalf("LinkBuildCore failed: err=%d pkg=%d marshal=%d/%d", linked.Error, linked.ErrorPackage, unit.LastMarshalError, unit.LastMarshalIndex)
	}
	decoded, ok := unit.Unmarshal(linked.Data)
	if !ok {
		t.Fatal("linked core unit did not decode")
	}
	found := false
	for i := 0; i < len(decoded.Tokens); i++ {
		if tokenAt(decoded, i) == "..." {
			t.Fatalf("function-value reparse recombined ellipsis token at %d", i)
		}
		if i+2 < len(decoded.Tokens) && tokenAt(decoded, i) == "." && tokenAt(decoded, i+1) == "." && tokenAt(decoded, i+2) == "." {
			found = true
		}
	}
	if !found {
		t.Fatalf("linked core unit missing split ellipsis after function-value reparse:\n%s", string(decoded.Text))
	}
}

func TestLinkBuildCoreLowersEndianSelectors(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/binary/binary.go", Src: []byte(`package binary

func PutUint32(b []byte, v int) {
	b[0] = byte(v)
}

func Uint32(b []byte) int {
	return int(b[0])
}
`)},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import "example.com/case/binary"

func appMain() int {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, uint32(7))
	if int(binary.LittleEndian.Uint32(buf)) == 7 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
`)},
	})
	linked := LinkBuildCore(result)
	if !linked.Ok {
		t.Fatalf("LinkBuildCore failed: err=%d pkg=%d marshal=%d/%d", linked.Error, linked.ErrorPackage, unit.LastMarshalError, unit.LastMarshalIndex)
	}
	decoded, ok := unit.Unmarshal(linked.Data)
	if !ok {
		t.Fatal("linked core unit did not decode")
	}
	if bytes.Contains(decoded.Text, []byte("LittleEndian.")) {
		t.Fatalf("linked text still contains endian selector:\n%s", string(decoded.Text))
	}
	if !bytes.Contains(decoded.Text, []byte("PutUint32(buf, uint32(7))")) ||
		!bytes.Contains(decoded.Text, []byte("Uint32(buf)")) {
		t.Fatalf("linked text missing lowered endian calls:\n%s", string(decoded.Text))
	}
}

func TestLinkBuildCorePreservesDeferPanicRecover(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

func guarded(v int) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			ok = v == 3
		}
	}()
	if v == 3 {
		panic("expected")
	}
	return false
}

func appMain() int {
	if guarded(3) {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
`)},
	})
	linked := LinkBuildCore(result)
	if !linked.Ok {
		t.Fatalf("LinkBuildCore failed: err=%d pkg=%d marshal=%d/%d", linked.Error, linked.ErrorPackage, unit.LastMarshalError, unit.LastMarshalIndex)
	}
	decoded, ok := unit.Unmarshal(linked.Data)
	if !ok {
		t.Fatal("linked core unit did not decode")
	}
	if !bytes.Contains(decoded.Text, []byte("func guarded(v int) (ok bool)")) ||
		!bytes.Contains(decoded.Text, []byte("defer func()")) ||
		!bytes.Contains(decoded.Text, []byte("recover()")) ||
		!bytes.Contains(decoded.Text, []byte("panic(\"expected\")")) ||
		!bytes.Contains(decoded.Text, []byte("return false")) {
		t.Fatalf("linked text did not preserve panic/defer semantics:\n%s", string(decoded.Text))
	}
}

func TestLinkBuildCorePreservesPackageInitVarDeps(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import "example.com/case/pkg/lib"

func main() {
	if lib.Value() == 8 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)},
		{Path: "/repo/case/pkg/lib/lib.go", Src: []byte(`package lib

var base = 0
var total = base + extra
var extra = 8

func Value() int {
	return total
}
`)},
	})
	linked := LinkBuildCore(result)
	if !linked.Ok {
		t.Fatalf("LinkBuildCore failed: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	decoded, ok := unit.Unmarshal(linked.Data)
	if !ok {
		t.Fatal("linked core unit did not decode")
	}
	total := findLinkedDecl(decoded, "total")
	if total < 0 {
		t.Fatalf("total not found in %#v", decoded.Decls)
	}
	if got := linkedSpanText(decoded, decoded.Decls[total].StartTok, decoded.Decls[total].EndTok); got != "var total = base + extra" {
		t.Fatalf("total decl span = %q, want preserved initializer\nfull text:\n%s", got, string(decoded.Text))
	}
}

func TestLinkBuildAddsRootEntrypointWrapper(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

func main() {
	print("PASS\n")
}
`)},
	})
	linked := LinkBuild(result)
	if !linked.Ok {
		t.Fatalf("LinkBuild failed: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	mainFn := findLinkedFunc(linked.Program, "main")
	appMain := findLinkedFunc(linked.Program, "appMain")
	if mainFn < 0 || appMain < 0 {
		t.Fatalf("linked funcs missing main/appMain: %#v", linked.Program.Funcs)
	}
	assertLinkedSymbol(t, linked.Program, "appMain", unit.SymbolFunc, unit.OwnerFunc, appMain)
	assertLinkedSignature(t, linked.Program, appMain, nil, nil, []string{":int"})
	assertLinkedStatement(t, linked.Program, appMain, unit.StmtExpr, "main()")
	assertLinkedStatement(t, linked.Program, appMain, unit.StmtReturn, "0")

	decoded, err := rtgunit.Unmarshal(linked.Data)
	if err != nil {
		t.Fatalf("linked unit did not decode: %v", err)
	}
	foundAppMain := false
	for i := 0; i < len(decoded.Funcs); i++ {
		if functionName(decoded, decoded.Funcs[i]) == "appMain" {
			foundAppMain = true
		}
	}
	if !foundAppMain {
		t.Fatalf("decoded funcs missing appMain: %#v", decoded.Funcs)
	}
}

func TestLinkBuildPassesProcessStateWhenRuntimeHookIsLinked(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import "example.com/case/process"

func main() { process.Use() }
`)},
		{Path: "/repo/case/process/process.go", Src: []byte(`package process

func rtg_runtime_SetProcess(args []string, env []string) {}
func Use() {}
`)},
	})
	linked := LinkBuild(result)
	if !linked.Ok {
		t.Fatalf("LinkBuild failed: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	appMain := findLinkedFunc(linked.Program, "appMain")
	if appMain < 0 {
		t.Fatalf("linked funcs missing appMain: %#v", linked.Program.Funcs)
	}
	assertLinkedSignature(t, linked.Program, appMain, nil, []string{"args:[]string", "env:[]string"}, []string{":int"})
	assertLinkedStatement(t, linked.Program, appMain, unit.StmtExpr, "rtg_runtime_SetProcess(args, env)")
	assertLinkedStatement(t, linked.Program, appMain, unit.StmtExpr, "main()")
	assertLinkedStatement(t, linked.Program, appMain, unit.StmtReturn, "0")

	decoded, err := rtgunit.Unmarshal(linked.Data)
	if err != nil {
		t.Fatalf("linked unit did not decode: %v", err)
	}
	if !bytes.Contains(decoded.Text, []byte("func appMain(args []string, env []string) int")) {
		t.Fatalf("decoded process-aware entrypoint missing:\n%s", string(decoded.Text))
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

const Answer = -7

type Numbers []int
type Record struct { Value int }
type Reader interface {
	Record
	Read(p []byte) int
}

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
	record := findLinkedDecl(program, "Record")
	reader := findLinkedDecl(program, "Reader")
	answer := findLinkedDecl(program, "Answer")
	values := findLinkedDecl(program, "Values")
	valueFn := findLinkedFunc(program, "Value")
	appMain := findLinkedFunc(program, "appMain")
	if numbers < 0 || record < 0 || reader < 0 || answer < 0 || values < 0 || valueFn < 0 || appMain < 0 {
		t.Fatalf("linked rows missing: decls=%#v funcs=%#v", program.Decls, program.Funcs)
	}
	assertLinkedConstInt(t, program, answer, -7)
	numbersSym := assertLinkedSymbol(t, program, "Numbers", unit.SymbolType, unit.OwnerDecl, numbers)
	assertLinkedSymbol(t, program, "Record", unit.SymbolType, unit.OwnerDecl, record)
	assertLinkedSymbol(t, program, "Reader", unit.SymbolType, unit.OwnerDecl, reader)
	valuesSym := assertLinkedSymbol(t, program, "Values", unit.SymbolVar, unit.OwnerDecl, values)
	assertLinkedSymbol(t, program, "Value", unit.SymbolFunc, unit.OwnerFunc, valueFn)
	assertLinkedSymbol(t, program, "appMain", unit.SymbolFunc, unit.OwnerFunc, appMain)
	assertLinkedDeclMeta(t, program, numbers, numbersSym, "[]int", "", nil)
	assertLinkedDeclMeta(t, program, values, valuesSym, "", "[]int{1, 2}", []string{"[]int{1, 2}"})
	assertLinkedInitOrder(t, program, []int{values})
	assertLinkedSignature(t, program, valueFn, nil, []string{"i:int"}, []string{":int"})
	assertLinkedSignature(t, program, appMain, nil, nil, []string{":int"})
	if len(program.Types) != 3 {
		t.Fatalf("linked types = %#v, want 3", program.Types)
	}
	typ := findLinkedTypeByDecl(program, numbers)
	if typ.Decl != numbers || linkedText(program, typ.NameStart, typ.NameEnd) != "Numbers" ||
		typ.Kind != unit.TypeSlice ||
		linkedSpanText(program, typ.TypeStart, typ.TypeEnd) != "[]int" ||
		linkedSpanText(program, typ.ElemStart, typ.ElemEnd) != "int" {
		t.Fatalf("linked type = %#v, decl %d", typ, numbers)
	}
	assertLinkedTypeFields(t, program, record, []string{"Value:int"})
	readerType := findLinkedTypeByDecl(program, reader)
	if readerType.Decl != reader || linkedText(program, readerType.NameStart, readerType.NameEnd) != "Reader" ||
		readerType.Kind != unit.TypeInterface ||
		linkedSpanText(program, readerType.TypeStart, readerType.TypeEnd) != "interface {\n\tRecord\n\tRead(p []byte) int\n}" {
		t.Fatalf("linked interface type = %#v, decl %d", readerType, reader)
	}
	assertLinkedTypeInterface(t, program, reader, []string{"Record"}, "Read", []string{"p:[]byte"}, []string{":int"})
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
	if len(program.Selectors) != 0 {
		t.Fatalf("linked selectors = %#v, want none", program.Selectors)
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
	if call.OwnerKind != unit.OwnerFunc || call.OwnerIndex != appMain || call.Kind != unit.CallPackage ||
		linkedTokenText(program, call.CalleeTok) != "Value" ||
		len(call.Args) != 1 ||
		linkedSpanText(program, call.Args[0].StartTok, call.Args[0].EndTok) != "1" {
		t.Fatalf("linked call = %#v, owner func %d", call, appMain)
	}
	for i := 0; i < len(program.Refs); i++ {
		ref := program.Refs[i]
		if ref.Kind == unit.RefImport {
			t.Fatalf("linked import ref survived: %#v in %#v", ref, program.Refs)
		}
	}
}

func TestLinkBuildPreservesStatements(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import "example.com/case/pkg/lib"

func appMain() int { return lib.Value(0) }
`)},
		{Path: "/repo/case/pkg/lib/lib.go", Src: []byte(`package lib

func Value(total int) int {
	if total == 0 {
		return 1
	}
	for total < 3 {
		total += 1
	}
	switch total {
	case 7:
		goto done
	default:
		break
	}
done:
	return total
}
`)},
	})
	linked := LinkBuild(result)
	if !linked.Ok {
		t.Fatalf("LinkBuild failed: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	valueFn := findLinkedFunc(linked.Program, "Value")
	appMain := findLinkedFunc(linked.Program, "appMain")
	if valueFn < 0 || appMain < 0 {
		t.Fatalf("linked funcs missing: %#v", linked.Program.Funcs)
	}
	assertLinkedStatement(t, linked.Program, valueFn, unit.StmtIf, "total == 0")
	assertLinkedStatement(t, linked.Program, valueFn, unit.StmtFor, "total < 3")
	assertLinkedStatement(t, linked.Program, valueFn, unit.StmtSwitch, "total")
	assertLinkedStatement(t, linked.Program, valueFn, unit.StmtCase, "7")
	assertLinkedStatement(t, linked.Program, valueFn, unit.StmtDefault, "")
	assertLinkedStatement(t, linked.Program, valueFn, unit.StmtGoto, "done")
	assertLinkedStatement(t, linked.Program, valueFn, unit.StmtBreak, "")
	assertLinkedStatement(t, linked.Program, valueFn, unit.StmtLabel, "")
	assertLinkedStatement(t, linked.Program, valueFn, unit.StmtReturn, "total")
	assertLinkedStatement(t, linked.Program, appMain, unit.StmtReturn, "Value(0)")

	decoded, err := rtgunit.Unmarshal(linked.Data)
	if err != nil {
		t.Fatalf("linked unit did not decode: %v", err)
	}
	if len(decoded.Stmts) != len(linked.Program.Stmts) {
		t.Fatalf("decoded statements = %d, want %d", len(decoded.Stmts), len(linked.Program.Stmts))
	}
}

func TestLinkUnitsPreservesMethodSets(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import "example.com/case/pkg/lib"

func appMain() int {
	var item lib.Item
	return item.Read()
}
`)},
		{Path: "/repo/case/pkg/lib/lib.go", Src: []byte(`package lib

type Item struct { Value int }

func (it Item) Read() int { return it.Value }
func (it *Item) Set(value int) int { return value }
`)},
	})
	program, ok := LinkUnits(result.Units, result.Root)
	if !ok {
		t.Fatal("LinkUnits failed")
	}
	item := findLinkedDecl(program, "Item")
	readFn := findLinkedFunc(program, "Read")
	setFn := findLinkedFunc(program, "Set")
	if item < 0 || readFn < 0 || setFn < 0 {
		t.Fatalf("linked rows missing: decls=%#v funcs=%#v", program.Decls, program.Funcs)
	}
	if len(program.Methods) != 2 {
		t.Fatalf("linked methods = %#v, want 2", program.Methods)
	}
	assertLinkedSignature(t, program, readFn, []string{"it:Item"}, nil, []string{":int"})
	assertLinkedSignature(t, program, setFn, []string{"it:*Item"}, []string{"value:int"}, []string{":int"})
	assertLinkedMethod(t, program, item, "Read", false, readFn)
	assertLinkedMethod(t, program, item, "Set", true, setFn)
}

func TestLinkUnitsPreservesFunctionTypes(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import "example.com/case/pkg/lib"

func use(callback lib.Callback) {}
func appMain() int { return lib.Value() }
`)},
		{Path: "/repo/case/pkg/lib/lib.go", Src: []byte(`package lib

type Callback func(value int) string

func Value() int { return 0 }
`)},
	})
	program, ok := LinkUnits(result.Units, result.Root)
	if !ok {
		t.Fatal("LinkUnits failed")
	}
	callback := findLinkedDecl(program, "Callback")
	if callback < 0 {
		t.Fatalf("linked callback decl missing: %#v", program.Decls)
	}
	if len(program.TypeFuncs) != 1 {
		t.Fatalf("linked type funcs = %#v, want 1", program.TypeFuncs)
	}
	typ := findLinkedTypeByDecl(program, callback)
	if typ.Decl != callback || linkedText(program, typ.NameStart, typ.NameEnd) != "Callback" ||
		typ.Kind != unit.TypeFunc ||
		linkedSpanText(program, typ.TypeStart, typ.TypeEnd) != "func(value int) string" {
		t.Fatalf("linked function type = %#v, decl %d", typ, callback)
	}
	assertLinkedTypeFunc(t, program, callback, []string{"value:int"}, []string{":string"})
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

func findLinkedToken(program unit.Program, text string) int {
	for i := 0; i < len(program.Tokens); i++ {
		if tokenAt(program, i) == text {
			return i
		}
	}
	return -1
}

func tokenAt(program unit.Program, tok int) string {
	if tok < 0 || tok >= len(program.Tokens) {
		return ""
	}
	token := program.Tokens[tok]
	start := token.Start
	end := token.Start + token.Size
	if start < 0 || end < start || end > len(program.Text) {
		return ""
	}
	return string(program.Text[start:end])
}

func findLinkedTypeByDecl(program unit.Program, decl int) unit.TypeInfo {
	for i := 0; i < len(program.Types); i++ {
		typ := program.Types[i]
		if typ.Decl == decl {
			return typ
		}
	}
	return unit.TypeInfo{Decl: -1}
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

func assertLinkedSignature(t *testing.T, program unit.Program, funcIndex int, receiver []string, params []string, results []string) {
	t.Helper()
	for i := 0; i < len(program.Signatures); i++ {
		sig := program.Signatures[i]
		if sig.FuncIndex != funcIndex {
			continue
		}
		assertLinkedFields(t, program, sig.Receiver, receiver)
		assertLinkedFields(t, program, sig.Params, params)
		assertLinkedFields(t, program, sig.Results, results)
		return
	}
	t.Fatalf("linked signature func=%d not found in %#v", funcIndex, program.Signatures)
}

func assertLinkedSymbol(t *testing.T, program unit.Program, name string, kind int, ownerKind int, ownerIndex int) int {
	t.Helper()
	for i := 0; i < len(program.Symbols); i++ {
		symbol := program.Symbols[i]
		if symbol.Name == name && symbol.Kind == kind && symbol.OwnerKind == ownerKind && symbol.OwnerIndex == ownerIndex {
			if linkedTokenText(program, symbol.Token) == "" {
				t.Fatalf("linked symbol %s token text is empty: %#v", name, symbol)
			}
			return i
		}
	}
	t.Fatalf("linked symbol %s kind=%d owner=%d/%d not found in %#v", name, kind, ownerKind, ownerIndex, program.Symbols)
	return -1
}

func assertLinkedDeclMeta(t *testing.T, program unit.Program, declIndex int, symbol int, typ string, value string, values []string) {
	t.Helper()
	for i := 0; i < len(program.DeclMeta); i++ {
		meta := program.DeclMeta[i]
		if meta.DeclIndex != declIndex || meta.Symbol != symbol {
			continue
		}
		if linkedSpanText(program, meta.TypeStart, meta.TypeEnd) != typ {
			t.Fatalf("linked decl meta %d type = %q, want %q", declIndex, linkedSpanText(program, meta.TypeStart, meta.TypeEnd), typ)
		}
		if linkedSpanText(program, meta.ValueStart, meta.ValueEnd) != value {
			t.Fatalf("linked decl meta %d value = %q, want %q", declIndex, linkedSpanText(program, meta.ValueStart, meta.ValueEnd), value)
		}
		if len(meta.Values) != len(values) {
			t.Fatalf("linked decl meta %d values = %#v, want %v", declIndex, meta.Values, values)
		}
		for j := 0; j < len(values); j++ {
			if got := linkedSpanText(program, meta.Values[j].StartTok, meta.Values[j].EndTok); got != values[j] {
				t.Fatalf("linked decl meta %d value %d = %q, want %q", declIndex, j, got, values[j])
			}
		}
		return
	}
	t.Fatalf("linked decl metadata index=%d symbol=%d not found in %#v", declIndex, symbol, program.DeclMeta)
}

func assertLinkedInitOrder(t *testing.T, program unit.Program, want []int) {
	t.Helper()
	if len(program.InitOrder) != len(want) {
		t.Fatalf("linked init order = %#v, want %#v", program.InitOrder, want)
	}
	for i := 0; i < len(want); i++ {
		if program.InitOrder[i] != want[i] {
			t.Fatalf("linked init order = %#v, want %#v", program.InitOrder, want)
		}
	}
}

func assertLinkedConstInt(t *testing.T, program unit.Program, declIndex int, value int) {
	t.Helper()
	for i := 0; i < len(program.Consts); i++ {
		c := program.Consts[i]
		if c.DeclIndex == declIndex && c.Kind == unit.ConstInt && c.Int == value {
			return
		}
	}
	t.Fatalf("linked const int decl=%d value=%d not found in %#v", declIndex, value, program.Consts)
}

func assertLinkedImport(t *testing.T, program unit.Program, name string, importPath string, dot bool, blank bool) {
	t.Helper()
	for i := 0; i < len(program.Imports); i++ {
		imp := program.Imports[i]
		if imp.Name != name || imp.ImportPath != importPath || imp.Dot != dot || imp.Blank != blank {
			continue
		}
		if imp.Package < 0 {
			t.Fatalf("linked import %s package = %d", importPath, imp.Package)
		}
		if linkedTokenText(program, imp.PathTok) != `"`+importPath+`"` {
			t.Fatalf("linked import %s path token = %q", importPath, linkedTokenText(program, imp.PathTok))
		}
		if name != "." && name != "_" && imp.NameTok >= 0 && linkedTokenText(program, imp.NameTok) != name {
			t.Fatalf("linked import %s name token = %q", importPath, linkedTokenText(program, imp.NameTok))
		}
		return
	}
	t.Fatalf("linked import %s %s not found in %#v", name, importPath, program.Imports)
}

func assertLinkedTypeFields(t *testing.T, program unit.Program, decl int, want []string) {
	t.Helper()
	typeIndex := -1
	for i := 0; i < len(program.Types); i++ {
		if program.Types[i].Decl == decl {
			typeIndex = i
			break
		}
	}
	if typeIndex < 0 {
		t.Fatalf("linked type decl=%d not found in %#v", decl, program.Types)
	}
	for i := 0; i < len(program.TypeFields); i++ {
		fields := program.TypeFields[i]
		if fields.TypeIndex == typeIndex {
			assertLinkedFields(t, program, fields.Fields, want)
			return
		}
	}
	t.Fatalf("linked type fields for type=%d not found in %#v", typeIndex, program.TypeFields)
}

func assertLinkedTypeInterface(t *testing.T, program unit.Program, decl int, embeds []string, method string, params []string, results []string) {
	t.Helper()
	typeIndex := -1
	for i := 0; i < len(program.Types); i++ {
		if program.Types[i].Decl == decl {
			typeIndex = i
			break
		}
	}
	if typeIndex < 0 {
		t.Fatalf("linked type decl=%d not found in %#v", decl, program.Types)
	}
	for i := 0; i < len(program.TypeIfaces); i++ {
		iface := program.TypeIfaces[i]
		if iface.TypeIndex != typeIndex {
			continue
		}
		if len(iface.Embeds) != len(embeds) {
			t.Fatalf("linked interface embeds = %#v, want %v", iface.Embeds, embeds)
		}
		for j := 0; j < len(embeds); j++ {
			got := linkedSpanText(program, iface.Embeds[j].TypeStart, iface.Embeds[j].TypeEnd)
			if got != embeds[j] {
				t.Fatalf("linked interface embed %d = %q, want %q", j, got, embeds[j])
			}
		}
		for j := 0; j < len(iface.Methods); j++ {
			m := iface.Methods[j]
			if linkedTokenText(program, m.NameTok) == method {
				assertLinkedFields(t, program, m.Params, params)
				assertLinkedFields(t, program, m.Results, results)
				return
			}
		}
		t.Fatalf("linked interface method %s not found in %#v", method, iface.Methods)
	}
	t.Fatalf("linked interface row for type=%d not found in %#v", typeIndex, program.TypeIfaces)
}

func assertLinkedTypeFunc(t *testing.T, program unit.Program, decl int, params []string, results []string) {
	t.Helper()
	typeIndex := -1
	for i := 0; i < len(program.Types); i++ {
		if program.Types[i].Decl == decl {
			typeIndex = i
			break
		}
	}
	if typeIndex < 0 {
		t.Fatalf("linked type decl=%d not found in %#v", decl, program.Types)
	}
	for i := 0; i < len(program.TypeFuncs); i++ {
		fn := program.TypeFuncs[i]
		if fn.TypeIndex == typeIndex {
			assertLinkedFields(t, program, fn.Params, params)
			assertLinkedFields(t, program, fn.Results, results)
			return
		}
	}
	t.Fatalf("linked function type row for type=%d not found in %#v", typeIndex, program.TypeFuncs)
}

func assertLinkedMethod(t *testing.T, program unit.Program, decl int, name string, pointer bool, funcIndex int) {
	t.Helper()
	typeIndex := -1
	typeName := ""
	for i := 0; i < len(program.Types); i++ {
		if program.Types[i].Decl == decl {
			typeIndex = i
			typeName = linkedText(program, program.Types[i].NameStart, program.Types[i].NameEnd)
			break
		}
	}
	if typeIndex < 0 {
		t.Fatalf("linked type decl=%d not found in %#v", decl, program.Types)
	}
	for i := 0; i < len(program.Methods); i++ {
		method := program.Methods[i]
		if method.TypeIndex != typeIndex || method.FuncIndex != funcIndex || method.Pointer != pointer || linkedTokenText(program, method.NameTok) != name {
			continue
		}
		if method.Symbol < 0 || method.Symbol >= len(program.Symbols) {
			t.Fatalf("linked method %s symbol = %d in %#v", name, method.Symbol, program.Symbols)
		}
		symbol := program.Symbols[method.Symbol]
		if symbol.Kind != unit.SymbolMethod || symbol.Name != typeName+"."+name || symbol.OwnerKind != unit.OwnerFunc || symbol.OwnerIndex != funcIndex {
			t.Fatalf("linked method %s symbol row = %#v, type %s func %d", name, symbol, typeName, funcIndex)
		}
		return
	}
	t.Fatalf("linked method %s pointer=%v func=%d type=%d not found in %#v", name, pointer, funcIndex, typeIndex, program.Methods)
}

func assertLinkedStatement(t *testing.T, program unit.Program, funcIndex int, kind int, expr string) {
	t.Helper()
	for i := 0; i < len(program.Stmts); i++ {
		stmt := program.Stmts[i]
		if stmt.FuncIndex != funcIndex || stmt.Kind != kind {
			continue
		}
		if linkedSpanText(program, stmt.ExprStart, stmt.ExprEnd) != expr {
			continue
		}
		if stmt.StartTok < 0 || stmt.EndTok < stmt.StartTok {
			t.Fatalf("linked statement kind=%d has invalid span: %#v", kind, stmt)
		}
		return
	}
	t.Fatalf("linked statement func=%d kind=%d expr=%q not found in %#v", funcIndex, kind, expr, program.Stmts)
}

func assertLinkedFields(t *testing.T, program unit.Program, fields []unit.Field, want []string) {
	t.Helper()
	if len(fields) != len(want) {
		t.Fatalf("linked fields = %#v, want %v", fields, want)
	}
	for i := 0; i < len(want); i++ {
		got := linkedTokenText(program, fields[i].NameTok) + ":" + linkedSpanText(program, fields[i].TypeStart, fields[i].TypeEnd)
		if got != want[i] {
			t.Fatalf("linked field %d = %q, want %q in %#v", i, got, want[i], fields)
		}
	}
}
