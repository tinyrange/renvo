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
