package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"j5.nz/rtg/rtg/ide"
	"j5.nz/rtg/rtg/std/graphics"
)

func TestMainFormGeneratedLayoutAndOpenSaveCallbacks(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "main.go")
	if err := os.WriteFile(path, []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	form := NewMainForm(root)
	controls := form.Controls()
	if len(controls) != 8 || controls[0] != &form.appBar.Control || controls[1] != &form.explorerFrame.Control || controls[2] != &form.editorFrame.Control || controls[3] != &form.designer.Control || controls[4] != &form.inspector.Control || controls[5] != &form.output.Control || controls[6] != &form.explorer.Control || controls[7] != &form.editor.Control {
		t.Fatalf("generated controls = %#v", controls)
	}
	if form.explorer.Font == nil || form.editor.Font == nil || form.explorer.Font == form.editor.Font {
		t.Fatal("generated form did not separate interface and code fonts")
	}
	form.explorerOpenFile(path)
	if form.currentPath != path || form.editor.Document.Text() != "package main\n" {
		t.Fatalf("opened state = %q, %q", form.currentPath, form.editor.Document.Text())
	}
	form.editor.Document.MoveDocumentEnd(false)
	form.editor.Document.Insert("// saved\n")
	form.saveCurrentFile()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "package main\n// saved\n" || form.editor.Document.Dirty() {
		t.Fatalf("saved state = %q, dirty %v", data, form.editor.Document.Dirty())
	}
	form.lastBuildOK = true
	form.editor.Document.Insert("// changed after build\n")
	form.Dispatch(graphics.Event{Type: graphics.EventNone})
	if form.lastBuildOK {
		t.Fatal("editing did not invalidate the previous build")
	}
}

func TestMainFormRendersAndResizesOnlyItsPanes(t *testing.T) {
	form := NewMainForm(t.TempDir())
	surface := graphics.NewSurface(1000, 700)
	if !form.Paint(surface) {
		t.Fatal("initial form did not paint")
	}
	if _, ok := surface.DirtyRect(); !ok {
		t.Fatal("initial form paint produced no pixels")
	}
	form.Dispatch(graphics.Event{Type: graphics.EventWindowResize, Dirty: graphics.R(0, 0, 720, 480)})
	layout := calculateWorkspaceLayout(720, 480)
	if form.explorer.Bounds() != layout.explorer {
		t.Fatalf("explorer bounds = %#v", form.explorer.Bounds())
	}
	wantEditor := rect(int(layout.editor.MinX), int(layout.editor.MinY), 720-int(layout.editor.MinX), int(layout.editor.Height()))
	if form.editor.Bounds() != wantEditor {
		t.Fatalf("editor bounds = %#v", form.editor.Bounds())
	}
	if form.designer.Bounds() != layout.designer || form.inspector.Bounds() != layout.inspector {
		t.Fatalf("mock workspace bounds = designer %#v inspector %#v", form.designer.Bounds(), form.inspector.Bounds())
	}
}

func TestWorkspaceReferenceGeometry(t *testing.T) {
	layout := calculateWorkspaceLayout(1440, 520)
	if layout.explorerFrame != graphics.R(0, 46, 263, 474) {
		t.Fatalf("explorer frame = %#v", layout.explorerFrame)
	}
	if layout.editorFrame != graphics.R(263, 46, 825, 362) {
		t.Fatalf("editor frame = %#v", layout.editorFrame)
	}
	if layout.designer != graphics.R(263, 46, 825, 362) {
		t.Fatalf("designer frame = %#v", layout.designer)
	}
	if layout.inspector != graphics.R(1088, 46, 352, 474) {
		t.Fatalf("inspector frame = %#v", layout.inspector)
	}
	if layout.output != graphics.R(263, 408, 825, 112) {
		t.Fatalf("output frame = %#v", layout.output)
	}
	if layout.explorer != graphics.R(0, 82, 263, 404) || layout.editor != graphics.R(263, 82, 825, 292) {
		t.Fatalf("live pane geometry = explorer %#v editor %#v", layout.explorer, layout.editor)
	}
}

func TestCodeAndDesignerAreSeparateViewsSharingDocumentBounds(t *testing.T) {
	form := NewMainForm(t.TempDir())
	if !form.editor.Visible() || form.designer.Visible() || form.inspector.Visible() || form.designerView {
		t.Fatal("new form did not start in code view")
	}
	form.showDesigner()
	if form.editor.Visible() || !form.designer.Visible() || !form.inspector.Visible() || !form.designerView {
		t.Fatal("designer view did not replace code view")
	}
	if form.editorFrame.Bounds() != form.designer.Bounds() {
		t.Fatalf("code/designer document bounds differ: %#v %#v", form.editorFrame.Bounds(), form.designer.Bounds())
	}
	form.showCode()
	if !form.editor.Visible() || form.designer.Visible() || form.inspector.Visible() || form.designerView {
		t.Fatal("code view did not replace designer view")
	}
}

func TestDesignerAddsMovesEditsAndWiresAControlThroughGoSource(t *testing.T) {
	root := t.TempDir()
	form := NewMainForm(root)
	form.showDesigner()
	if !form.designerView {
		t.Fatal("designer did not open generated form source")
	}

	// Pick Button from the live palette. It is inserted and selected.
	designer := form.designer.Bounds()
	paletteX := designer.MinX + 48 + 88 + 20
	buttonY := designer.MinY + workspacePaneHeaderHeight + 20
	form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: paletteX, Y: buttonY, Button: 1})
	form.Dispatch(graphics.Event{Type: graphics.EventPointerUp, X: paletteX, Y: buttonY, Button: 1})
	selected := len(form.design.controls) - 1
	if selected != 2 || form.design.controls[selected].name != "button1" || form.designer.selected != selected || form.inspector.selected != selected {
		t.Fatalf("inserted control state = %#v, designer %d inspector %d", form.design.controls, form.designer.selected, form.inspector.selected)
	}

	// Drag the selected control through normal form dispatch. Pointer capture
	// keeps the operation attached to the designer until pointer-up.
	canvas := graphics.R(designer.MinX, designer.MinY+workspacePaneHeaderHeight+workspaceDesignerToolbarHeight, designer.Width(), designer.Height()-workspacePaneHeaderHeight-workspaceDesignerToolbarHeight-workspaceStatusHeight)
	layout := calculateDesignerPreview(canvas, &form.design)
	controlBounds := designerControlBounds(layout, form.design.controls[selected])
	startX := controlBounds.MinX + controlBounds.Width()/2
	startY := controlBounds.MinY + controlBounds.Height()/2
	oldX := form.design.controls[selected].x
	form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: startX, Y: startY, Button: 1})
	form.Dispatch(graphics.Event{Type: graphics.EventPointerMove, X: startX + 24, Y: startY + 12, Button: 1})
	form.Dispatch(graphics.Event{Type: graphics.EventPointerUp, X: startX + 24, Y: startY + 12, Button: 1})
	if form.design.controls[selected].x <= oldX {
		t.Fatalf("drag did not move control: %#v", form.design.controls[selected])
	}
	layout = calculateDesignerPreview(canvas, &form.design)
	controlBounds = designerControlBounds(layout, form.design.controls[selected])
	resizeX := controlBounds.MaxX
	resizeY := controlBounds.MaxY
	oldWidth := form.design.controls[selected].width
	form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: resizeX, Y: resizeY, Button: 1})
	form.Dispatch(graphics.Event{Type: graphics.EventPointerMove, X: resizeX + 20, Y: resizeY + 8, Button: 1})
	form.Dispatch(graphics.Event{Type: graphics.EventPointerUp, X: resizeX + 20, Y: resizeY + 8, Button: 1})
	if form.design.controls[selected].width <= oldWidth {
		t.Fatalf("resize handle did not resize control: %#v", form.design.controls[selected])
	}

	// Replace Text in the property grid, then create the empty Click event.
	inspector := form.inspector.Bounds()
	propertyX := inspector.MinX + 100
	textY := inspector.MinY + workspacePaneHeaderHeight + 48 + 40 + 10
	form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: propertyX, Y: textY, Button: 1})
	form.Dispatch(graphics.Event{Type: graphics.EventTextInput, Text: "Launch"})
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyEnter})
	clickY := inspector.MinY + workspacePaneHeaderHeight + 48 + 6*40 + 10
	form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: propertyX, Y: clickY, Button: 1})

	generated, err := os.ReadFile(filepath.Join(root, projectGeneratedFormFile))
	if err != nil {
		t.Fatal(err)
	}
	generatedText := string(generated)
	if !strings.Contains(generatedText, `f.button1.SetText("Launch")`) || !strings.Contains(generatedText, "f.button1.Click = f.button1Click") {
		t.Fatalf("generated source did not own edited properties and event:\n%s", generated)
	}
	user, err := os.ReadFile(filepath.Join(root, projectUserFormFile))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(user), "func (f *MainForm) button1Click()") != 1 {
		t.Fatalf("user callback was not created exactly once:\n%s", user)
	}

	for paletteIndex := 2; paletteIndex < 8; paletteIndex++ {
		x := designer.MinX + 48 + graphics.Scalar(paletteIndex*88+20)
		y := designer.MinY + workspacePaneHeaderHeight + 20
		form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: x, Y: y, Button: 1})
		form.Dispatch(graphics.Event{Type: graphics.EventPointerUp, X: x, Y: y, Button: 1})
		if paletteIndex == 4 {
			checkedY := inspector.MinY + workspacePaneHeaderHeight + 48 + 6*40 + 10
			form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: propertyX, Y: checkedY, Button: 1})
			form.Dispatch(graphics.Event{Type: graphics.EventPointerUp, X: propertyX, Y: checkedY, Button: 1})
		}
	}
	if len(form.design.controls) != 9 {
		t.Fatalf("complete palette created %d controls, want 9", len(form.design.controls))
	}
	generated, err = os.ReadFile(filepath.Join(root, projectGeneratedFormFile))
	if err != nil {
		t.Fatal(err)
	}
	for _, constructor := range []string{"forms.NewTextBox()", "forms.NewTextArea()", "forms.NewCheckBox()", "forms.NewRadioButton()", "forms.NewPictureBox()", "forms.NewPanel()"} {
		if !strings.Contains(string(generated), constructor) {
			t.Fatalf("generated source missing %s", constructor)
		}
	}
	if !strings.Contains(string(generated), ".SetChecked(true)") {
		t.Fatal("checked property was not regenerated")
	}
	surface := graphics.NewSurface(1440, 520)
	writeUIAutomationScreenshot(t, form, surface, "06_designer_controls.ppm")
}

func TestEditorNavigationDoesNotDamageMockWorkspacePanes(t *testing.T) {
	form := NewMainForm(t.TempDir())
	form.editor.SetDocument(ide.NewDocument([]byte("one\ntwo\nthree\n")))
	surface := graphics.NewSurface(1440, 520)
	form.Paint(surface)

	editorBounds := form.editor.Bounds()
	form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: editorBounds.MinX + 80, Y: editorBounds.MinY + 8, Button: 1})
	form.Dispatch(graphics.Event{Type: graphics.EventPointerUp, X: editorBounds.MinX + 80, Y: editorBounds.MinY + 8, Button: 1})
	form.Paint(surface)
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyDown})

	invalid := form.InvalidRects()
	if len(invalid) == 0 {
		t.Fatal("caret navigation produced no damage")
	}
	for i := 0; i < len(invalid); i++ {
		if invalid[i].MaxX > form.editorFrame.Bounds().MaxX {
			t.Fatalf("editor navigation damaged mocked workspace pane: %#v", invalid[i])
		}
	}
}
