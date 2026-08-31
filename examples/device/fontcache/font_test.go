package fontcache

import (
	"testing"

	"renvo.dev/std/graphics"
)

func TestTitleCacheRetainsTrueTypeCoverage(t *testing.T) {
	font := TitleSubset("A")
	if font == nil {
		t.Fatal("26-pixel title cache failed to load")
	}
	if height := graphics.MeasureText(font, "A").Height; height < 25 {
		t.Fatalf("title cache line height = %v, want at least 25", height)
	}
	canvas := graphics.NewImageFormat(48, 48, graphics.PixelA8, nil)
	canvas.DrawText(font, graphics.Point{X: 4, Y: 4 + font.Metrics.Ascent}, "A", graphics.White)
	partial := 0
	for _, coverage := range canvas.Pixels {
		if coverage != 0 && coverage != 255 {
			partial++
		}
	}
	if partial == 0 {
		t.Fatal("cached TrueType glyph was reduced to a binary mask")
	}
}
