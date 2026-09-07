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
	if !bytes.Contains(linked.Program.Text, []byte("__renvo_call_0(first.Click)")) ||
		!bytes.Contains(linked.Program.Text, []byte("__renvo_call_1(second.Click, 2)")) {
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
		!bytes.Contains(linked.Program.Text, []byte("__renvo_call_0(items[index].Click)")) {
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

func TestFunctionValueFieldDirectFunctionCompositeIsLowered(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

type Methods struct {
	allocate func(int32) *byte
	release func(*byte)
}
type Config struct { methods Methods }

func allocate(size int32) *byte { return nil }
func release(value *byte) {}

var config = Config{methods: Methods{allocate: allocate, release: release}}
var allocators [1]func(int32) *byte = [1]func(int32) *byte{allocate}
func callAllocator() *byte { return allocators[0](1) }
func main() {}
`)},
	})
	linked := LinkBuildCore(result)
	if !linked.Ok {
		t.Fatalf("LinkBuildCore failed: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	if !bytes.Contains(linked.Program.Text, []byte("allocate: __renvo_function_0{kind: 1}")) ||
		!bytes.Contains(linked.Program.Text, []byte("release: __renvo_function_1{kind: 1}")) ||
		!bytes.Contains(linked.Program.Text, []byte("[1]__renvo_function_0{__renvo_function_0{kind: 1}}")) ||
		!bytes.Contains(linked.Program.Text, []byte("__renvo_call_0(allocators[0], 1)")) {
		t.Fatalf("direct callbacks in a composite literal were not lowered:\n%s", linked.Program.Text)
	}
}

func TestFunctionValueFieldCallThroughParenthesizedPointerIsLowered(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

type Callbacks struct { run func(*Callbacks, int) int }

func invoke(callbacks **Callbacks) int {
	return (*callbacks).run((*callbacks), 41)
}

func main() {}
`)},
	})
	linked := LinkBuildCore(result)
	if !linked.Ok {
		t.Fatalf("LinkBuildCore failed: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	if !bytes.Contains(linked.Program.Text, []byte("__renvo_call_0((*callbacks).run, (*callbacks), 41)")) {
		t.Fatalf("callback field through a parenthesized pointer was not lowered:\n%s", linked.Program.Text)
	}
}

func TestFunctionValueFieldParenthesizedNilComparisonIsLowered(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

type Callbacks struct { run func() }

func available(callbacks *Callbacks) bool {
	return (callbacks.run) != nil
}

func main() {}
`)},
	})
	linked := LinkBuildCore(result)
	if !linked.Ok {
		t.Fatalf("LinkBuildCore failed: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	if !bytes.Contains(linked.Program.Text, []byte("(callbacks.run.kind) != 0")) {
		t.Fatalf("parenthesized callback nil comparison was not lowered:\n%s", linked.Program.Text)
	}
}

func TestFunctionValueFieldCallOnTypeSwitchBindingIsLowered(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

type Value interface { value() int }
type Number int
func (n Number) value() int { return int(n) }
type Tuple []Value
type Callback func(*Builtin, Tuple, []Tuple) (Value, error)
type Builtin struct { fn Callback }

func invoke(value Value, args Tuple, kwargs []Tuple) (Value, error) {
	switch fn := value.(type) {
	case *Builtin:
		return fn.fn(fn, args, kwargs)
	default:
		return nil, nil
	}
}

func main() {}
`)},
	})
	linked := LinkBuildCore(result)
	if !linked.Ok {
		t.Fatalf("LinkBuildCore failed: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	if !bytes.Contains(linked.Program.Text, []byte("__renvo_call_0(fn.fn, fn, args, kwargs)")) {
		t.Fatalf("callback field on a type-switch binding was not lowered:\n%s", linked.Program.Text)
	}
	if !bytes.Contains(linked.Program.Text, []byte("arg1 Tuple, arg2 []Tuple")) {
		t.Fatalf("unnamed named-type callback parameters were grouped incorrectly:\n%s", linked.Program.Text)
	}
	if !bytes.Contains(linked.Program.Text, []byte("var __renvo_zero_0 Value")) ||
		!bytes.Contains(linked.Program.Text, []byte("var __renvo_zero_1 error")) {
		t.Fatalf("multi-result callback dispatcher has invalid zero results:\n%s", linked.Program.Text)
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

func TestFunctionValueFieldDirectFunctionComparisonIsLowered(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

type Value struct { destroy func(*byte) }
func release(value *byte) {}

func matches(value *Value) bool {
	return value.destroy == (release)
}

func present(destroy func(*byte)) bool {
	return (destroy) != nil
}

func choose(destroy bool) func(*byte) {
	if destroy { return release }
	return nil
}

func identity(destroy func(*byte)) func(*byte) { return destroy }
func roundtrip(destroy func(*byte)) bool { return identity(destroy) != nil }

func main() {}
`)},
	})
	linked := LinkBuildCore(result)
	if !linked.Ok {
		t.Fatalf("LinkBuildCore failed: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	if !bytes.Contains(linked.Program.Text, []byte("value.destroy.kind == (1)")) ||
		!bytes.Contains(linked.Program.Text, []byte("(destroy.kind) != 0")) ||
		!bytes.Contains(linked.Program.Text, []byte("return __renvo_function_0{kind: 1}")) ||
		!bytes.Contains(linked.Program.Text, []byte("return __renvo_function_0{}")) ||
		!bytes.Contains(linked.Program.Text, []byte("identity(destroy).kind != 0")) ||
		!bytes.Contains(linked.Program.Text, []byte("release(arg0)")) {
		t.Fatalf("direct callback comparison was not lowered through its tag:\n%s", linked.Program.Text)
	}
}
