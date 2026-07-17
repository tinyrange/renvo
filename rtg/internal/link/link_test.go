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
	linked := LinkBuildCore(result)
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
	program, ok := LinkUnitsCore(result.Units, result.Root)
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

func TestLinkUnitsAdjustsTokenOffsets(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import "example.com/case/pkg/lib"

func appMain() int { return lib.Value() }
`)},
		{Path: "/repo/case/pkg/lib/lib.go", Src: []byte("package lib\n\nfunc Value() int { return 1 }\n")},
	})
	program, ok := LinkUnitsCore(result.Units, result.Root)
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
	program, ok := LinkUnitsCore(result.Units, result.Root)
	if !ok {
		t.Fatal("LinkUnits failed")
	}
	left := findLinkedFunc(program, "rtgp0_Value")
	right := findLinkedFunc(program, "rtgp1_Value")
	appMain := findLinkedFunc(program, "appMain")
	if left < 0 || right < 0 || appMain < 0 {
		t.Fatalf("linked funcs missing aliases: %#v", program.Funcs)
	}
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
	program, ok := LinkUnitsCore(result.Units, result.Root)
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
	program, ok := LinkUnitsCore(result.Units, result.Root)
	if !ok {
		t.Fatal("LinkUnits failed")
	}
	if bytes.Contains(program.Text, []byte("Pointer(&v)")) || bytes.Contains(program.Text, []byte("(*pair)(p)")) {
		t.Fatalf("linked text still contains unsafe pointer conversions:\n%s", string(program.Text))
	}
	if !bytes.Contains(program.Text, []byte("p := &v")) || !bytes.Contains(program.Text, []byte("q :=p")) {
		t.Fatalf("linked text missing erased unsafe pointer values:\n%s", string(program.Text))
	}
	linked := LinkBuildCore(result)
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
	program, ok := LinkUnitsCore(result.Units, result.Root)
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
	linked := LinkBuildCore(result)
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
		t.Fatalf("LinkBuildCore failed: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	decoded := linked.Program
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
	if _, err := rtgunit.Unmarshal(linked.Data); err != nil {
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
	if _, err := rtgunit.Unmarshal(linked.Data); err != nil {
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
		t.Fatalf("LinkBuildCore failed: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	decoded := linked.Program
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
		t.Fatalf("LinkBuildCore failed: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	decoded := linked.Program
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
		t.Fatalf("LinkBuildCore failed: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	decoded := linked.Program
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
		t.Fatalf("LinkBuildCore failed: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	decoded := linked.Program
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

func TestLinkBuildCoreClassifiesNumbersAfterFunctionValueReparse(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

type callback func() int

func appMain() int {
	fn := callback(func() int { return 0xe9b5dba5 })
	if fn() == 0xe9b5dba5 && 1e2 == 100 && 0x1.ep+2 == 7.5 {
		print("PASS\n")
		return 0
	}
	return 1
}
`)},
	})
	linked := LinkBuildCore(result)
	if !linked.Ok {
		t.Fatalf("LinkBuildCore failed: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	wantKinds := map[string]int{
		"0xe9b5dba5": unit.TokenNumber,
		"1e2":        unit.TokenFloat,
		"0x1.ep+2":   unit.TokenFloat,
	}
	found := make(map[string]bool)
	for i := 0; i < len(linked.Program.Tokens); i++ {
		text := tokenAt(linked.Program, i)
		want, ok := wantKinds[text]
		if !ok {
			continue
		}
		if got := linked.Program.Tokens[i].Kind; got != want {
			t.Fatalf("token %q kind = %d, want %d", text, got, want)
		}
		found[text] = true
	}
	for text := range wantKinds {
		if !found[text] {
			t.Fatalf("linked program missing token %q:\n%s", text, string(linked.Program.Text))
		}
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
		t.Fatalf("LinkBuildCore failed: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	decoded := linked.Program
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
		t.Fatalf("LinkBuildCore failed: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	decoded := linked.Program
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
	decoded := linked.Program
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
	linked := LinkBuildCore(result)
	if !linked.Ok {
		t.Fatalf("LinkBuild failed: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	mainFn := findLinkedFunc(linked.Program, "main")
	appMain := findLinkedFunc(linked.Program, "appMain")
	if mainFn < 0 || appMain < 0 {
		t.Fatalf("linked funcs missing main/appMain: %#v", linked.Program.Funcs)
	}
	if !bytes.Contains(linked.Program.Text, []byte("func appMain() int")) ||
		!bytes.Contains(linked.Program.Text, []byte("main()")) ||
		!bytes.Contains(linked.Program.Text, []byte("return 0")) {
		t.Fatalf("linked entrypoint wrapper is incomplete:\n%s", linked.Program.Text)
	}

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
	linked := LinkBuildCore(result)
	if !linked.Ok {
		t.Fatalf("LinkBuild failed: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	appMain := findLinkedFunc(linked.Program, "appMain")
	if appMain < 0 {
		t.Fatalf("linked funcs missing appMain: %#v", linked.Program.Funcs)
	}
	if !bytes.Contains(linked.Program.Text, []byte("func appMain(args []string, env []string) int")) ||
		!bytes.Contains(linked.Program.Text, []byte("rtg_runtime_SetProcess(args, env)")) ||
		!bytes.Contains(linked.Program.Text, []byte("main()")) {
		t.Fatalf("linked process-aware entrypoint is incomplete:\n%s", linked.Program.Text)
	}

	decoded, err := rtgunit.Unmarshal(linked.Data)
	if err != nil {
		t.Fatalf("linked unit did not decode: %v", err)
	}
	if !bytes.Contains(decoded.Text, []byte("func appMain(args []string, env []string) int")) {
		t.Fatalf("decoded process-aware entrypoint missing:\n%s", string(decoded.Text))
	}
}

func TestLinkBuildRejectsInvalidInput(t *testing.T) {
	badBuild := build.Result{Ok: false, ErrorPackage: 7}
	linked := LinkBuildCore(badBuild)
	if linked.Ok || linked.Error != LinkErrBuild || linked.ErrorPackage != 7 {
		t.Fatalf("bad build link result = %#v", linked)
	}

	linked = LinkBuildCore(build.Result{Ok: true, Root: -1})
	if linked.Ok || linked.Error != LinkErrRoot {
		t.Fatalf("bad root link result = %#v", linked)
	}

	if _, ok := LinkUnitsCore(nil, 0); ok {
		t.Fatal("LinkUnits accepted empty unit list")
	}
}

func buildFromFiles(t *testing.T, files []load.SourceFile) build.Result {
	t.Helper()
	workspace := load.LoadWorkspace("/repo/case", "/std", "./cmd/app", files)
	if !workspace.Ok {
		t.Fatalf("LoadWorkspace failed: err=%d file=%d", workspace.Error, workspace.ErrorFile)
	}
	result := build.BuildPrograms(workspace.Graph)
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
