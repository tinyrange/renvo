package ide

import (
	"testing"

	"renvo.dev/std/graphics"
)

func TestUIFontHasReadableEditorMetrics(t *testing.T) {
	font := NewUIFont()
	metrics := graphics.MeasureText(font, "M")
	if metrics.Width < 7 || metrics.Height < 14 {
		t.Fatalf("UI font metrics are too small: %#v", metrics)
	}
}

func TestUIFontUsesOneStableEditorCellWidth(t *testing.T) {
	font := NewUIFont()
	want := graphics.MeasureText(font, "M").Width
	for _, text := range []string{"i", "W", "0", " "} {
		got := graphics.MeasureText(font, text).Width
		if got < want-0.01 || got > want+0.01 {
			t.Fatalf("glyph %q width = %v, M width = %v", text, got, want)
		}
	}
}
