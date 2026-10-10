package rtg

import "testing"

func TestAssemblerLinePositionsAcrossCommentsAndSeparators(t *testing.T) {
	// Spans retain the untrimmed physical source, including comments and
	// byte-based UTF-8 columns, not just the normalized instruction text.
	source := []byte("# header\n\tadd; /*é*/ sub\n\n/* across\nlines */ mov;ret")
	lines, diagnostics := assemblerLines(source, "positions.s")
	if len(diagnostics) != 0 || len(lines) != 4 {
		t.Fatal(lines, diagnostics)
	}
	want := []struct {
		text       string
		start, end Position
	}{
		{"add", Position{Offset: 9, Line: 2, Column: 1}, Position{Offset: 13, Line: 2, Column: 5}},
		{"sub", Position{Offset: 14, Line: 2, Column: 6}, Position{Offset: 25, Line: 2, Column: 17}},
		{"mov", Position{Offset: 37, Line: 5, Column: 1}, Position{Offset: 49, Line: 5, Column: 13}},
		{"ret", Position{Offset: 50, Line: 5, Column: 14}, Position{Offset: 53, Line: 5, Column: 17}},
	}
	for i, line := range lines {
		if line.text != want[i].text || line.span.Start != want[i].start || line.span.End != want[i].end {
			t.Fatalf("line %d: %+v want %+v", i, line, want[i])
		}
	}
}
