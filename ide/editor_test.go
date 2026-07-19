package ide

import "testing"

func TestDocumentPreservesLineEndingsAndPositions(t *testing.T) {
	document := NewDocument([]byte("alpha\r\nβeta\r\n"))
	if document.LineEnding() != LineEndingCRLF {
		t.Fatalf("line ending = %d", document.LineEnding())
	}
	if document.Text() != "alpha\nβeta\n" {
		t.Fatalf("normalized text = %q", document.Text())
	}
	if string(document.Bytes()) != "alpha\r\nβeta\r\n" {
		t.Fatalf("saved bytes = %q", document.Bytes())
	}
	if document.LineCount() != 3 || document.LineText(1) != "βeta" {
		t.Fatalf("lines = %d, line 1 = %q", document.LineCount(), document.LineText(1))
	}
	offset := document.Offset(1, 1)
	line, column := document.Position(offset)
	if line != 1 || column != 1 || document.Text()[offset:] != "eta\n" {
		t.Fatalf("position round trip = offset %d, line %d, column %d", offset, line, column)
	}
}

func TestDocumentUnicodeSelectionAndEditing(t *testing.T) {
	document := NewDocument([]byte("aβc"))
	document.MoveCharacter(2, false)
	if document.Caret != len("aβ") {
		t.Fatalf("caret = %d", document.Caret)
	}
	document.MoveCharacter(-1, true)
	if document.SelectedText() != "β" {
		t.Fatalf("selected = %q", document.SelectedText())
	}
	document.Insert("λ")
	if document.Text() != "aλc" || !document.Dirty() {
		t.Fatalf("text/dirty = %q/%v", document.Text(), document.Dirty())
	}
	document.Backspace()
	if document.Text() != "ac" {
		t.Fatalf("after backspace = %q", document.Text())
	}
	document.Delete()
	if document.Text() != "a" {
		t.Fatalf("after delete = %q", document.Text())
	}
}

func TestDocumentVerticalMovementRetainsDesiredColumn(t *testing.T) {
	document := NewDocument([]byte("abcdef\nx\nuvwxyz\n"))
	document.SetSelection(document.Offset(0, 5), document.Offset(0, 5))
	document.MoveLine(1, false)
	line, column := document.Position(document.Caret)
	if line != 1 || column != 1 {
		t.Fatalf("short line position = %d:%d", line, column)
	}
	document.MoveLine(1, false)
	line, column = document.Position(document.Caret)
	if line != 2 || column != 5 {
		t.Fatalf("restored desired column = %d:%d", line, column)
	}
}

func TestDocumentVisualColumnsExpandTabsForArrowMovement(t *testing.T) {
	document := NewDocument([]byte("\tabcdef\n    z\n\tabcdef"))
	document.SetSelection(document.Offset(0, 2), document.Offset(0, 2))
	line, visual := document.VisualPosition(document.Caret)
	if line != 0 || visual != 5 {
		t.Fatalf("initial visual position = %d:%d", line, visual)
	}
	document.MoveLine(1, false)
	line, visual = document.VisualPosition(document.Caret)
	if line != 1 || visual != 5 {
		t.Fatalf("space-indented visual position = %d:%d", line, visual)
	}
	document.MoveLine(1, false)
	line, visual = document.VisualPosition(document.Caret)
	logicalLine, logicalColumn := document.Position(document.Caret)
	if line != 2 || visual != 5 || logicalLine != 2 || logicalColumn != 2 {
		t.Fatalf("restored tab position = visual %d:%d, logical %d:%d", line, visual, logicalLine, logicalColumn)
	}
	if offset := document.VisualOffset(0, 4); offset != document.Offset(0, 1) {
		t.Fatalf("visual tab boundary offset = %d, want %d", offset, document.Offset(0, 1))
	}
}

func TestDocumentTypingAndDeletionCoalesceUndo(t *testing.T) {
	document := NewDocument(nil)
	document.Insert("a")
	document.Insert("β")
	document.Insert("c")
	if document.Text() != "aβc" {
		t.Fatalf("typed text = %q", document.Text())
	}
	if !document.Undo() || document.Text() != "" || document.Dirty() {
		t.Fatalf("coalesced typing undo = %q, dirty %v", document.Text(), document.Dirty())
	}
	if !document.Redo() || document.Text() != "aβc" {
		t.Fatalf("typing redo = %q", document.Text())
	}

	document.Backspace()
	document.Backspace()
	if document.Text() != "a" {
		t.Fatalf("backspace text = %q", document.Text())
	}
	if !document.Undo() || document.Text() != "aβc" {
		t.Fatalf("coalesced backspace undo = %q", document.Text())
	}
}

func TestDocumentDeletionReportsWhetherTextChanged(t *testing.T) {
	document := NewDocument([]byte("a"))
	if document.Backspace() {
		t.Fatal("Backspace at document start reported a change")
	}
	if !document.Delete() || document.Text() != "" {
		t.Fatalf("Delete did not remove text: %q", document.Text())
	}
	if document.Delete() || document.Backspace() {
		t.Fatal("deletion in an empty document reported a change")
	}
}

func TestDocumentSavePointSurvivesUndoAndBranches(t *testing.T) {
	document := NewDocument([]byte("one"))
	document.MoveDocumentEnd(false)
	document.Insert(" two")
	document.MarkSaved()
	if document.Dirty() {
		t.Fatal("document remained dirty after save")
	}
	document.Insert(" three")
	if !document.Dirty() {
		t.Fatal("document did not become dirty")
	}
	if !document.Undo() || document.Text() != "one two" || document.Dirty() {
		t.Fatalf("undo to save point = %q, dirty %v", document.Text(), document.Dirty())
	}
	document.Insert("!")
	redo := document.Redo()
	if !document.Dirty() || redo {
		t.Fatalf("branch dirty/redo = %v/%v", document.Dirty(), redo)
	}
}

func TestDocumentSelectionDirectionAndHomeEnd(t *testing.T) {
	document := NewDocument([]byte("first\nsecond"))
	document.SetSelection(document.Offset(1, 4), document.Offset(1, 1))
	if document.SelectedText() != "eco" {
		t.Fatalf("reverse selected text = %q", document.SelectedText())
	}
	document.MoveHome(false)
	line, column := document.Position(document.Caret)
	if line != 1 || column != 0 {
		t.Fatalf("home = %d:%d", line, column)
	}
	document.MoveEnd(true)
	if document.SelectedText() != "second" {
		t.Fatalf("shift end selection = %q", document.SelectedText())
	}
}
