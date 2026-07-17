package main

import (
	"os"
	"path/filepath"
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
	if form.editor.Bounds() != layout.editor {
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
	if !form.editor.Visible() || form.designer.Visible() || form.designerView {
		t.Fatal("new form did not start in code view")
	}
	form.showDesigner()
	if form.editor.Visible() || !form.designer.Visible() || !form.designerView {
		t.Fatal("designer view did not replace code view")
	}
	if form.editorFrame.Bounds() != form.designer.Bounds() {
		t.Fatalf("code/designer document bounds differ: %#v %#v", form.editorFrame.Bounds(), form.designer.Bounds())
	}
	form.showCode()
	if !form.editor.Visible() || form.designer.Visible() || form.designerView {
		t.Fatal("code view did not replace designer view")
	}
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
