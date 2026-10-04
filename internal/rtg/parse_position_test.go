package rtg

import "testing"

func TestParserSourcePositions(t *testing.T) {
	// Columns count bytes, including multibyte UTF-8 and the CR in CRLF.
	parser := documentParser{document: &Document{Source: []byte("α\r\n\nxyz\n")}}
	cases := []Position{
		{Offset: 0, Line: 1, Column: 1},
		{Offset: 1, Line: 1, Column: 2},
		{Offset: 2, Line: 1, Column: 3},
		{Offset: 3, Line: 1, Column: 4},
		{Offset: 4, Line: 2, Column: 1},
		{Offset: 5, Line: 3, Column: 1},
		{Offset: 8, Line: 3, Column: 4},
		{Offset: 9, Line: 4, Column: 1},
	}
	// Resolve in reverse order so lookup does not depend on monotonically
	// increasing offsets, as nested declarations can revisit earlier spans.
	for i := len(cases) - 1; i >= 0; i-- {
		want := cases[i]
		if got := parser.sourcePosition(want.Offset); got != want {
			t.Fatalf("offset %d: got %+v, want %+v", want.Offset, got, want)
		}
	}
	want := Span{Start: cases[0], End: cases[len(cases)-1]}
	if got := parser.sourceSpan(-1, 100); got != want {
		t.Fatalf("clamped span: got %+v, want %+v", got, want)
	}
	empty := documentParser{document: &Document{}}
	if got := empty.sourcePosition(1); got != (Position{Offset: 0, Line: 1, Column: 1}) {
		t.Fatalf("empty source: %+v", got)
	}
}
