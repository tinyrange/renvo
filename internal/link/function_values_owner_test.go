package link

import (
	"bytes"
	"testing"

	"renvo.dev/internal/load"
)

func TestFunctionValueFieldsUseOwningStructIdentity(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

type FirstCallback func()
type SecondCallback func(int)
type First struct { Click FirstCallback }
type Second struct { Click SecondCallback }
type State struct { total int }

func main() {
	state := &State{}
	first := First{Click: func() { state.total++ }}
	second := Second{Click: func(value int) { state.total += value }}
	first.Click()
	second.Click(2)
	if state.total == 3 { print("PASS\n") }
}
`)},
	})
	linked := LinkBuildCore(result)
	if !linked.Ok {
		t.Fatalf("LinkBuildCore failed: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	if !bytes.Contains(linked.Program.Text, []byte("__renvo_call_0(&first.Click)")) ||
		!bytes.Contains(linked.Program.Text, []byte("__renvo_call_1(&second.Click, 2)")) {
		t.Fatalf("same-named callback fields were not independently lowered:\n%s", linked.Program.Text)
	}
}

func TestFunctionValueFieldCallThroughIndexedPointerIsLowered(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

type Callback func()
type Item struct { Click Callback }

func invoke(items []*Item, index int) {
	if items[index].Click != nil {
		items[index].Click()
	}
}

func main() { invoke([]*Item{{}}, 0); print("PASS\n") }
`)},
	})
	linked := LinkBuildCore(result)
	if !linked.Ok {
		t.Fatalf("LinkBuildCore failed: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	if !bytes.Contains(linked.Program.Text, []byte("items[index].Click.kind != 0")) ||
		!bytes.Contains(linked.Program.Text, []byte("__renvo_call_0(&items[index].Click)")) {
		t.Fatalf("indexed callback field was not lowered:\n%s", linked.Program.Text)
	}
}

func TestFunctionValueFieldDirectFunctionAssignmentIsLowered(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

type Button struct { Click func() }
var clicked bool
func click() { clicked = true }

func main() {
	var button Button
	button.Click = click
	button.Click()
	if clicked { print("PASS\n") }
}
`)},
	})
	linked := LinkBuildCore(result)
	if !linked.Ok {
		t.Fatalf("LinkBuildCore failed: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	if !bytes.Contains(linked.Program.Text, []byte("button.Click = __renvo_function_0{kind: 1}")) ||
		!bytes.Contains(linked.Program.Text, []byte("click(); return")) {
		t.Fatalf("direct callback assignment was not lowered with its target:\n%s", linked.Program.Text)
	}
}

func TestFunctionValueFieldClosureAssignmentIsLowered(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

type FlagSet struct {
	called bool
	Usage func()
}

func newFlagSet() *FlagSet {
	f := &FlagSet{}
	f.Usage = func() { f.called = true }
	return f
}

func main() {
	f := newFlagSet()
	f.Usage()
	if f.called { print("PASS\n") }
}
`)},
	})
	linked := LinkBuildCore(result)
	if !linked.Ok {
		t.Fatalf("LinkBuildCore failed: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	for _, want := range [][]byte{
		[]byte("f.Usage = __renvo_function_0{kind: 1"),
		[]byte("f: f"),
		[]byte("env.f.called = true"),
		[]byte("__renvo_call_0(&f.Usage)"),
	} {
		if !bytes.Contains(linked.Program.Text, want) {
			t.Fatalf("closure field assignment did not contain %q:\n%s", want, linked.Program.Text)
		}
	}
}

func TestFunctionValueFieldNameDoesNotRewriteNestedMethod(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

type Handler func()
type Form struct { Resize Handler }
type Surface struct{}
func (s *Surface) Resize(width, height int) {}
type Window struct { surface *Surface }
var current *Window

func draw() { w := current; w.surface.Resize(360, 800) }
func main() { current = &Window{surface: &Surface{}}; draw() }
`)},
	})
	linked := LinkBuildCore(result)
	if !linked.Ok {
		t.Fatalf("LinkBuildCore failed: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	if !bytes.Contains(linked.Program.Text, []byte("w.surface.Resize(360, 800)")) {
		t.Fatalf("nested method sharing a callback field name was rewritten:\n%s", linked.Program.Text)
	}
}

func TestFunctionValuesShareStorageForMethodsOnSameReceiverType(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

type Handler func()
type Button struct { Click Handler }
type Controller struct { total int }
func (c *Controller) one() { c.total++ }
func (c *Controller) two() { c.total += 2 }

func main() {
	c := &Controller{}
	var first, second Button
	first.Click = c.one
	second.Click = c.two
	first.Click()
	second.Click()
	if c.total == 3 { print("PASS\n") }
}
`)},
	})
	linked := LinkBuildCore(result)
	if !linked.Ok {
		t.Fatalf("LinkBuildCore failed: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	if !bytes.Contains(linked.Program.Text, []byte("receiver0: c")) ||
		bytes.Contains(linked.Program.Text, []byte("receiver1")) ||
		!bytes.Contains(linked.Program.Text, []byte("fn.receiver0.one()")) ||
		!bytes.Contains(linked.Program.Text, []byte("fn.receiver0.two()")) {
		t.Fatalf("same-receiver methods did not share function-value storage:\n%s", linked.Program.Text)
	}
}
