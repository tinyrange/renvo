package check

import (
	"testing"

	"j5.nz/rtg/rtg/internal/load"
)

func TestCompleteGraphResolvesScopeFieldsAndChainedMethods(t *testing.T) {
	mainSource := []byte(`package main

func (f *MainForm) update() {
	localValue := 1
	_ = loc
	f.mes
	f.messageLabel.SetT
	_ = localValue
}
`)
	files := []load.SourceFile{
		{Path: "/repo/go.mod", Src: []byte("module example.com/app\n")},
		{Path: "/repo/forms/forms.go", Src: []byte(`package forms
type Label struct{}
func (l *Label) SetText(text string) {}
func (l *Label) Text() string { return "" }
`)},
		{Path: "/repo/main_form_generated.go", Src: []byte(`package main
import "example.com/app/forms"
type MainForm struct { messageLabel *forms.Label }
`)},
		{Path: "/repo/main_form.go", Src: mainSource},
	}
	workspace := load.LoadWorkspace("/repo", "/std", ".", files)
	if !workspace.Ok {
		t.Fatalf("workspace failed: %#v", workspace)
	}
	assertCompletion := func(marker, want string) {
		t.Helper()
		offset := completionTestOffset(mainSource, marker)
		items := CompleteGraph(workspace.Graph, "/repo/main_form.go", offset)
		for i := 0; i < len(items); i++ {
			if items[i].Name == want {
				return
			}
		}
		t.Fatalf("completion at %q = %#v, want %q", marker, items, want)
	}
	assertCompletion("loc\n", "localValue")
	assertCompletion("f.mes\n", "messageLabel")
	assertCompletion("f.messageLabel.SetT\n", "SetText")
}

func completionTestOffset(source []byte, marker string) int {
	for i := 0; i+len(marker) <= len(source); i++ {
		if string(source[i:i+len(marker)]) == marker {
			return i + len(marker) - 1
		}
	}
	return -1
}
