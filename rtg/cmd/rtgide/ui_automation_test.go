package main

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"j5.nz/rtg/rtg/std/graphics"
)

func TestIDEUIAutomationScreenshots(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "main.go")
	source := "package main\n\nimport \"fmt\"\n\nfunc main() {\n"
	for i := 0; i < 32; i++ {
		n := strconv.Itoa(i)
		source += "\tvalue" + n + " := \"this deliberately long highlighted editor line " + n + " keeps the caret away from every viewport edge even in the expanded code-only layout\" // row " + n + "\n"
	}
	source += "\tfmt.Println(value31)\n}\n"
	if err := os.WriteFile(path, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}

	form := NewMainForm(root)
	form.Dispatch(graphics.Event{Type: graphics.EventWindowResize, Dirty: graphics.R(0, 0, 1440, 520)})
	surface := graphics.NewSurface(1440, 520)
	form.Paint(surface)

	// Activate main.go through the same pointer-down/up route as the window.
	explorerBounds := form.explorer.Bounds()
	explorerX := explorerBounds.MinX + 20
	explorerY := explorerBounds.MinY + graphics.Scalar(form.explorer.RowHeight()+form.explorer.RowHeight()/2)
	form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: explorerX, Y: explorerY, Button: 1})
	form.Dispatch(graphics.Event{Type: graphics.EventPointerUp, X: explorerX, Y: explorerY, Button: 1})
	if form.currentPath != path {
		t.Fatalf("automation did not open main.go: %q", form.currentPath)
	}
	writeUIAutomationScreenshot(t, form, surface, "01_syntax.ppm")

	// Focus the editor and walk the caret far enough to exercise contextual
	// vertical scrolling.
	editorBounds := form.editor.Bounds()
	editorX := editorBounds.MinX + 80
	editorY := editorBounds.MinY + 8
	form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: editorX, Y: editorY, Button: 1})
	form.Dispatch(graphics.Event{Type: graphics.EventPointerUp, X: editorX, Y: editorY, Button: 1})
	for i := 0; i < 27; i++ {
		form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyDown})
	}
	lineScroll, _ := form.editor.ScrollPosition()
	visibleLines, _ := form.editor.VisibleGrid()
	caretLine, _ := form.editor.Document.Position(form.editor.Document.Caret)
	caretScreenLine := caretLine - lineScroll
	if lineScroll == 0 || caretScreenLine <= 0 || caretScreenLine >= visibleLines-1 {
		t.Fatalf("vertical caret context = caret %d, scroll %d, visible %d", caretLine, lineScroll, visibleLines)
	}
	writeUIAutomationScreenshot(t, form, surface, "02_vertical_scroll.ppm")

	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyEnd})
	_, columnScroll := form.editor.ScrollPosition()
	_, visibleColumns := form.editor.VisibleGrid()
	_, caretColumn := form.editor.Document.VisualPosition(form.editor.Document.Caret)
	caretScreenColumn := caretColumn - columnScroll
	if columnScroll == 0 || caretScreenColumn <= 0 || caretScreenColumn >= visibleColumns-1 {
		t.Fatalf("horizontal caret context = caret %d, scroll %d, visible %d", caretColumn, columnScroll, visibleColumns)
	}
	writeUIAutomationScreenshot(t, form, surface, "03_horizontal_scroll.ppm")

	// Go source is normally tab-indented. Arrowing across a tab must advance
	// by the same four visual cells used by text rendering, while the document
	// offset advances by one rune.
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyHome})
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyRight})
	_, logicalColumn := form.editor.Document.Position(form.editor.Document.Caret)
	_, visualColumn := form.editor.Document.VisualPosition(form.editor.Document.Caret)
	if logicalColumn != 1 || visualColumn != 4 {
		t.Fatalf("right over tab = logical %d, visual %d", logicalColumn, visualColumn)
	}
	writeUIAutomationScreenshot(t, form, surface, "04_arrow_after_tab.ppm")

	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyRight})
	_, logicalColumn = form.editor.Document.Position(form.editor.Document.Caret)
	_, visualColumn = form.editor.Document.VisualPosition(form.editor.Document.Caret)
	if logicalColumn != 2 || visualColumn != 5 {
		t.Fatalf("right over first letter = logical %d, visual %d", logicalColumn, visualColumn)
	}
	writeUIAutomationScreenshot(t, form, surface, "05_arrow_after_letter.ppm")
}

func writeUIAutomationScreenshot(t *testing.T, form *MainForm, surface *graphics.Surface, name string) {
	t.Helper()
	form.Paint(surface)
	dir := os.Getenv("RTG_UI_SCREENSHOT_DIR")
	if dir == "" {
		dir = filepath.Join(os.TempDir(), "rtgide-ui-screenshots")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, surface.EncodePPM(), 0644); err != nil {
		t.Fatal(err)
	}
	t.Logf("UI screenshot: %s", path)
}
