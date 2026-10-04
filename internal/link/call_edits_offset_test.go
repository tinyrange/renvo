package link

import "testing"

func TestBuiltinCallOffsetsAcrossMixedEdits(t *testing.T) {
	edits := []functionValueEdit{{start: 2, end: 5, text: "longer"}, {start: 7, end: 7, text: "insert"}, {start: 7, end: 10, text: "x"}, {start: 14, end: 20, text: ""}}
	var changes []callTokenEdit
	delta := 0
	for _, edit := range edits {
		changes = append(changes, callTokenEdit{offset: edit.start + delta})
		delta += len(edit.text) - (edit.end - edit.start)
	}
	for position := -1; position <= 24; position++ {
		got := mapBuiltinCallOffset(position, edits, changes, 22)
		want := mapFunctionValueOffset(position, edits, 22)
		if got != want {
			t.Fatalf("offset %d: got %d, want %d", position, got, want)
		}
	}
}
