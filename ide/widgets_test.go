package ide

import (
	"os"
	"path/filepath"
	"testing"

	"renvo.dev/forms"
	"renvo.dev/std/graphics"
)

func TestIDEControlsAndSyntaxPaletteFollowDarkTheme(t *testing.T) {
	explorer := NewExplorerControl(OpenExplorer(t.TempDir()))
	explorer.SetBounds(graphics.R(0, 0, 160, 220))
	editor := NewEditorControl(NewDocument([]byte("func main() { println(\"dark\", 42) // comment\n}")))
	editor.SetBounds(graphics.R(160, 0, 340, 220))
	var form forms.Form
	form.Initialize(500, 220)
	form.Add(&explorer.Control)
	form.Add(&editor.Control)
	dark := forms.DarkTheme()
	form.ApplyTheme(dark)
	if explorer.Background() != dark.Surface || explorer.Foreground() != dark.Text || editor.Background() != dark.Field || editor.Foreground() != dark.Text {
		t.Fatalf("dark IDE controls = explorer %#v/%#v editor %#v/%#v", explorer.Background(), explorer.Foreground(), editor.Background(), editor.Foreground())
	}
	palette := editorPalette(dark)
	light := editorPalette(forms.LightTheme())
	if palette.background != dark.Field || palette.gutter != dark.Surface || palette.popup != dark.SurfaceRaised || palette.text != dark.Text || palette.border != dark.Border {
		t.Fatalf("dark editor surfaces = %#v", palette)
	}
	for kind := syntaxKeyword; kind <= syntaxNumber; kind++ {
		if syntaxColor(kind, palette) == syntaxColor(kind, light) || syntaxColor(kind, palette) == dark.Field {
			t.Fatalf("syntax kind %d did not receive a readable dark color", kind)
		}
	}
	surface := graphics.NewSurface(500, 220)
	form.Paint(surface)
	if got := widgetTestPixel(surface, 300, 200); got != dark.Field {
		t.Fatalf("painted dark editor background = %#v, want %#v", got, dark.Field)
	}
	if got := widgetTestPixel(surface, 80, 200); got != dark.Surface {
		t.Fatalf("painted dark explorer background = %#v, want %#v", got, dark.Surface)
	}
}

func widgetTestPixel(surface *graphics.Surface, x, y int) graphics.Color {
	offset := y*surface.Stride + x*4
	return graphics.Color{R: surface.Pixels[offset], G: surface.Pixels[offset+1], B: surface.Pixels[offset+2], A: surface.Pixels[offset+3]}
}

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

func TestEditorBackspaceJoiningLinesInvalidatesNewCaretLine(t *testing.T) {
	document := NewDocument([]byte("one\ntwo\nthree\nfour"))
	control := NewEditorControl(document)
	control.SetDocument(document)
	control.SetBounds(graphics.R(20, 10, 400, 160))
	var form forms.Form
	form.Initialize(460, 190)
	form.Add(&control.Control)
	control.Focus()
	surface := graphics.NewSurface(460, 190)
	form.Paint(surface)
	document.SetSelection(document.Offset(1, 0), document.Offset(1, 0))

	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyBackspace})
	line, column := document.Position(document.Caret)
	if line != 0 || column != 3 {
		t.Fatalf("joined-line caret = %d:%d", line, column)
	}
	invalid := form.InvalidRects()
	if len(invalid) == 0 || invalid[0].MinY > control.Bounds().MinY {
		t.Fatalf("joined-line damage missed caret row: %#v", invalid)
	}
	form.Paint(surface)
	document.MoveDocumentStart(false)
	changed := 0
	control.Changed = func() { changed++ }
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyBackspace})
	if changed != 0 || len(form.InvalidRects()) != 0 {
		t.Fatalf("Backspace at document start did work: changed %d damage %#v", changed, form.InvalidRects())
	}
}

func TestEditorTabCompletionSelectsAndReplacesPrefix(t *testing.T) {
	document := NewDocument([]byte("f.SetT"))
	document.MoveDocumentEnd(false)
	control := NewEditorControl(document)
	control.SetAccessibilityID("code-editor")
	control.SetBounds(graphics.R(0, 0, 500, 200))
	control.Complete = func(source []byte, caret int) []Completion {
		return []Completion{{Text: "SetText", Detail: "method"}, {Text: "SetTitle", Detail: "method"}}
	}
	var form forms.Form
	form.Initialize(500, 200)
	form.Add(&control.Control)
	form.TakeAccessibilityUpdate()
	control.Focus()
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyTab})
	form.Dispatch(graphics.Event{Type: graphics.EventTextInput, Text: "\t"})
	if len(control.completions) != 2 {
		t.Fatalf("completion popup = %#v", control.completions)
	}
	driver := forms.NewAutomationDriver(&form)
	items := driver.Find(forms.AccessibilityRoleListItem, "")
	if len(items) != 2 || items[0].Name != "SetText" || items[1].Name != "SetTitle" {
		t.Fatalf("completion semantics = %#v", items)
	}
	opened, ok := form.TakeAccessibilityUpdate()
	if !ok || len(opened.ReplaceChildren) != 1 {
		t.Fatalf("completion-open update = %#v, %v", opened, ok)
	}
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyDown})
	selection, ok := form.TakeAccessibilityUpdate()
	if !ok || len(selection.ReplaceChildren) != 0 {
		t.Fatalf("completion selection rebuilt its semantic children: %#v, %v", selection, ok)
	}
	surface := graphics.NewSurface(500, 200)
	form.Paint(surface)
	popup := control.completionBounds()
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyTab})
	form.Dispatch(graphics.Event{Type: graphics.EventTextInput, Text: "\t"})
	if document.Text() != "f.SetTitle" {
		t.Fatalf("completed document = %q", document.Text())
	}
	if len(control.completions) != 0 {
		t.Fatalf("completion remained after Tab: %#v", control.completions)
	}
	invalidatedPopup := false
	for _, dirty := range form.InvalidRects() {
		if dirty.MinX < popup.MaxX && dirty.MaxX > popup.MinX && dirty.MinY < popup.MaxY && dirty.MaxY > popup.MinY {
			invalidatedPopup = true
			break
		}
	}
	if !invalidatedPopup {
		t.Fatalf("Tab did not invalidate completion popup %#v", popup)
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

func TestEditorCompletionRefiltersWhileTyping(t *testing.T) {
	document := NewDocument([]byte("f.Set"))
	document.MoveDocumentEnd(false)
	control := NewEditorControl(document)
	control.SetBounds(graphics.R(0, 0, 500, 200))
	queries := 0
	control.Complete = func(source []byte, caret int) []Completion {
		queries++
		prefix := string(source[completionWordStart(source, caret):caret])
		if prefix == "SetT" {
			return []Completion{{Text: "SetText", Detail: "(text string)"}, {Text: "SetTitle", Detail: "(title string)"}}
		}
		return []Completion{{Text: "SetText", Detail: "(text string)"}, {Text: "SetTitle", Detail: "(title string)"}, {Text: "SetVisible", Detail: "(visible bool)"}}
	}
	var form forms.Form
	form.Initialize(500, 200)
	form.Add(&control.Control)
	control.Focus()
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyTab})
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyUnknown})
	if len(control.completions) != 3 {
		t.Fatal("printable key-down dismissed completion before text input")
	}
	form.Dispatch(graphics.Event{Type: graphics.EventTextInput, Text: "T"})
	if document.Text() != "f.SetT" || len(control.completions) != 2 {
		t.Fatalf("filtered completion: text=%q items=%#v", document.Text(), control.completions)
	}
	if queries != 1 {
		t.Fatalf("typing re-ran semantic completion %d times", queries)
	}
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyTab})
	if document.Text() != "f.SetText" {
		t.Fatalf("completed document = %q", document.Text())
	}
}

func TestEditorShowsAndUpdatesSignatureHelp(t *testing.T) {
	document := NewDocument([]byte("label.SetPosition"))
	document.MoveDocumentEnd(false)
	control := NewEditorControl(document)
	control.SetBounds(graphics.R(0, 0, 500, 200))
	control.Signature = func(source []byte, caret int, help *SignatureHelp) {
		// A partially typed argument can make semantic analysis temporarily fail.
		// The editor must retain the last valid signature until this call closes.
		if caret == 0 || source[caret-1] != '(' {
			return
		}
		*help = SignatureHelp{Ok: true, Label: "SetPosition(x int, y int)", Parameters: []SignatureParameter{{Name: "x", Type: "int"}, {Name: "y", Type: "int"}}}
	}
	var form forms.Form
	form.Initialize(500, 200)
	form.Add(&control.Control)
	control.Focus()
	form.Dispatch(graphics.Event{Type: graphics.EventTextInput, Text: "("})
	if !control.signature.Ok || control.signature.ActiveParameter != 0 {
		t.Fatalf("initial signature = %#v", control.signature)
	}
	form.Dispatch(graphics.Event{Type: graphics.EventTextInput, Text: "1, "})
	if !control.signature.Ok || control.signature.ActiveParameter != 1 {
		t.Fatalf("updated signature = %#v", control.signature)
	}
	form.Dispatch(graphics.Event{Type: graphics.EventTextInput, Text: "2)"})
	if control.signature.Ok {
		t.Fatalf("signature remained after closing parenthesis: %#v", control.signature)
	}
}

func TestEditorVSCodeCompletionAndParameterShortcuts(t *testing.T) {
	document := NewDocument([]byte("call( "))
	document.MoveDocumentEnd(false)
	control := NewEditorControl(document)
	control.Complete = func(source []byte, caret int) []Completion {
		items := make([]Completion, 12)
		for i := 0; i < len(items); i++ {
			items[i] = Completion{Text: "item" + decimal(i), Detail: "variable"}
		}
		return items
	}
	control.Signature = func(source []byte, caret int, help *SignatureHelp) {
		*help = SignatureHelp{Ok: true, Label: "call(value int)", Parameters: []SignatureParameter{{Name: "value", Type: "int"}}}
	}
	var form forms.Form
	form.Initialize(500, 200)
	form.Add(&control.Control)
	control.Focus()

	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeySpace, Modifiers: graphics.ModifierControl})
	form.Dispatch(graphics.Event{Type: graphics.EventTextInput, Text: " ", Modifiers: graphics.ModifierControl})
	if len(control.completions) != 12 {
		t.Fatalf("Ctrl+Space completions = %d", len(control.completions))
	}
	if document.Text() != "call( " {
		t.Fatalf("Ctrl+Space inserted text: %q", document.Text())
	}
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyPageDown})
	if control.completionPick != 8 {
		t.Fatalf("PageDown selection = %d", control.completionPick)
	}
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyEscape})
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyI, Modifiers: graphics.ModifierCommand})
	if len(control.completions) != 12 {
		t.Fatalf("Cmd+I completions = %d", len(control.completions))
	}
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyEscape})
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeySpace, Modifiers: graphics.ModifierControl | graphics.ModifierShift})
	if !control.signature.Ok {
		t.Fatal("Ctrl+Shift+Space did not show parameter hints")
	}
}
