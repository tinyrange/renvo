package ide

import (
	"testing"

	"renvo.dev/std/graphics"
	"renvo.dev/std/graphics/gofont"
)

func TestUIFontHasReadableEditorMetrics(t *testing.T) {
	font := NewUIFont()
	metrics := graphics.MeasureText(font, "M")
	if metrics.Width < 7 || metrics.Height < 14 {
		t.Fatalf("UI font metrics are too small: %#v", metrics)
	}
}

func TestEmbeddedUIFontUsesOneStableEditorCellWidth(t *testing.T) {
	font := gofont.NewMono(16)
	if font == nil {
		t.Fatal("embedded editor font failed to load")
	}
	want := graphics.MeasureText(font, "M").Width
	for _, text := range []string{"i", "W", "0", " "} {
		got := graphics.MeasureText(font, text).Width
		if got < want-0.01 || got > want+0.01 {
			t.Fatalf("glyph %q width = %v, M width = %v", text, got, want)
		}
	}
}
