package ide

import (
	"os"
	"path/filepath"
	"testing"

	"j5.nz/rtg/rtg/forms"
	"j5.nz/rtg/rtg/std/graphics"
)

func TestExplorerControlSelectionInvalidatesOnlyChangedRows(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "b.go"), []byte("package b\n"), 0644); err != nil {
		t.Fatal(err)
	}
	model := OpenExplorer(root)
	control := NewExplorerControl(model)
	control.SetBounds(graphics.R(0, 0, 180, 100))
	var form forms.Form
	form.Initialize(300, 200)
	form.Add(&control.Control)
	surface := graphics.NewSurface(300, 200)
	form.Paint(surface)

	control.pointerDown(20, graphics.Scalar(control.rowHeight+1))
	invalid := form.InvalidRects()
	if len(invalid) != 2 || invalid[0].Height() != graphics.Scalar(control.rowHeight) || invalid[1].Height() != graphics.Scalar(control.rowHeight) {
		t.Fatalf("selection damage = %#v", invalid)
	}
}

func TestEditorControlUnicodeInputDamagesCurrentLine(t *testing.T) {
	document := NewDocument([]byte("package main\n"))
	control := NewEditorControl(document)
	control.SetBounds(graphics.R(180, 0, 300, 200))
	var form forms.Form
	form.Initialize(500, 220)
	form.Add(&control.Control)
	surface := graphics.NewSurface(500, 220)
	form.Paint(surface)
	control.Focus()
	form.Paint(surface)

	document.MoveEnd(false)
	control.textInput(" λ")
	if document.Text() != "package main λ\n" {
		t.Fatalf("document text = %q", document.Text())
	}
	invalid := form.InvalidRects()
	if len(invalid) == 0 {
		t.Fatal("typing did not invalidate editor")
	}
	for i := 0; i < len(invalid); i++ {
		if invalid[i].Height() > graphics.Scalar(control.lineHeight) {
			t.Fatalf("single-line edit damaged too much: %#v", invalid)
		}
	}
}

func TestEditorControlNewlineDamagesOnlyFollowingEditorArea(t *testing.T) {
	document := NewDocument([]byte("one\ntwo\nthree"))
	control := NewEditorControl(document)
	control.SetBounds(graphics.R(200, 10, 300, 150))
	var form forms.Form
	form.Initialize(520, 180)
	form.Add(&control.Control)
	surface := graphics.NewSurface(520, 180)
	form.Paint(surface)

	document.SetSelection(document.Offset(1, 1), document.Offset(1, 1))
	control.textInput("\n")
	invalid := form.InvalidRects()
	if len(invalid) != 1 {
		t.Fatalf("newline invalid regions = %#v", invalid)
	}
	wantY := graphics.Scalar(10 + control.lineHeight)
	if invalid[0].MinX != 200 || invalid[0].MinY != wantY || invalid[0].MaxX != 500 || invalid[0].MaxY != 160 {
		t.Fatalf("newline damage = %#v", invalid[0])
	}
}

func TestEditorTabCompletionSelectsAndReplacesPrefix(t *testing.T) {
	document := NewDocument([]byte("f.SetT"))
	document.MoveDocumentEnd(false)
	control := NewEditorControl(document)
	control.SetBounds(graphics.R(0, 0, 500, 200))
	control.Complete = func(source []byte, caret int) []Completion {
		return []Completion{{Text: "SetText", Detail: "method"}, {Text: "SetTitle", Detail: "method"}}
	}
	var form forms.Form
	form.Initialize(500, 200)
	form.Add(&control.Control)
	control.Focus()
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyTab})
	form.Dispatch(graphics.Event{Type: graphics.EventTextInput, Text: "\t"})
	if len(control.completions) != 2 {
		t.Fatalf("completion popup = %#v", control.completions)
	}
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyDown})
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyTab})
	form.Dispatch(graphics.Event{Type: graphics.EventTextInput, Text: "\t"})
	if document.Text() != "f.SetTitle" {
		t.Fatalf("completed document = %q", document.Text())
	}
}

func TestEditorTabIndentsWhitespaceWithoutTextInputDuplication(t *testing.T) {
	document := NewDocument([]byte("    "))
	document.MoveDocumentEnd(false)
	control := NewEditorControl(document)
	control.Complete = func(source []byte, caret int) []Completion {
		return []Completion{{Text: "ignored", Detail: "variable"}}
	}
	var form forms.Form
	form.Initialize(500, 200)
	form.Add(&control.Control)
	control.Focus()
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyTab})
	form.Dispatch(graphics.Event{Type: graphics.EventTextInput, Text: "\t"})
	if document.Text() != "    \t" {
		t.Fatalf("indented document = %q", document.Text())
	}
}
