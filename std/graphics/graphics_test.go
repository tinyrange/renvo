package graphics

import "testing"

func pixel(s *Surface, x, y int) Color {
	o := y*s.Stride + x*4
	return Color{s.Pixels[o], s.Pixels[o+1], s.Pixels[o+2], s.Pixels[o+3]}
}

func TestPremultipliedColorAndSourceOver(t *testing.T) {
	s := NewSurface(2, 2)
	s.Clear(RGBA(0, 0, 255, 255))
	s.FillRect(R(0, 0, 1, 1), RGBA(255, 0, 0, 128))
	c := pixel(s, 0, 0)
	if c.R != 128 || c.G != 0 || c.B != 127 || c.A != 255 {
		t.Fatalf("source-over pixel = %#v", c)
	}
}

func TestFillRectBlendsEachCoveredPixelOnce(t *testing.T) {
	s := NewSurface(8, 8)
	s.FillRect(R(1, 1, 6, 6), RGBA(255, 0, 0, 128))
	want := Color{R: 128, A: 128}
	for y := 1; y < 7; y++ {
		for x := 1; x < 7; x++ {
			if got := pixel(s, x, y); got != want {
				t.Fatalf("pixel %d,%d = %#v, want %#v", x, y, got, want)
			}
		}
	}
}

func TestHalfOpenClipAndTransform(t *testing.T) {
	s := NewSurface(5, 5)
	s.PushClipRect(R(1, 1, 2, 2))
	transform := Translate(1, 1)
	s.SetTransform(&transform)
	s.FillRect(R(0, 0, 4, 4), White)
	s.PopClip()
	for y := 0; y < 5; y++ {
		for x := 0; x < 5; x++ {
			want := x >= 1 && x < 3 && y >= 1 && y < 3
			if (pixel(s, x, y).A != 0) != want {
				t.Fatalf("pixel %d,%d coverage mismatch", x, y)
			}
		}
	}
}

func TestTriangleLineAndImage(t *testing.T) {
	s := NewSurface(8, 8)
	s.FillTriangle(Point{0, 0}, Point{6, 0}, Point{0, 6}, White)
	if pixel(s, 1, 1).A != 255 || pixel(s, 6, 6).A != 0 {
		t.Fatalf("triangle coverage failed")
	}
	s.DrawLine(Point{1, 7}, Point{6, 7}, 1, RGBA(0, 255, 0, 255))
	if pixel(s, 3, 7).G != 255 {
		t.Fatalf("line coverage failed")
	}
	image := NewSurface(1, 1)
	image.Clear(RGBA(255, 0, 0, 255))
	s.DrawImage(image, R(0, 0, 1, 1), R(6, 0, 2, 2), SamplingNearest, White)
	if pixel(s, 7, 1).R != 255 {
		t.Fatalf("image sampling failed")
	}
}

func TestIntegerAlignedScaledImageFastPath(t *testing.T) {
	image := NewImage(2, 2, []byte{
		255, 0, 0, 255, 0, 255, 0, 255,
		0, 0, 255, 255, 255, 255, 255, 255,
	})
	s := NewSurface(6, 6)
	s.DrawImage(image, R(0, 0, 2, 2), R(1, 1, 4, 4), SamplingNearest, White)
	checks := []struct {
		x, y int
		want Color
	}{
		{x: 1, y: 1, want: Color{R: 255, A: 255}},
		{x: 4, y: 1, want: Color{G: 255, A: 255}},
		{x: 1, y: 4, want: Color{B: 255, A: 255}},
		{x: 4, y: 4, want: White},
	}
	for _, check := range checks {
		if got := pixel(s, check.x, check.y); got != check.want {
			t.Fatalf("scaled pixel %d,%d = %#v, want %#v", check.x, check.y, got, check.want)
		}
	}
	if pixel(s, 0, 0).A != 0 || pixel(s, 5, 5).A != 0 {
		t.Fatal("scaled image escaped its half-open destination")
	}
}

func TestDiagonalLineHasContinuousCoverage(t *testing.T) {
	s := NewSurface(40, 40)
	s.DrawLine(Point{0, 0}, Point{32, 32}, 2, White)
	for _, coordinate := range []int{2, 5, 11, 18, 27, 31} {
		if pixel(s, coordinate, coordinate).A != 255 {
			t.Fatalf("diagonal line missing pixel %d,%d", coordinate, coordinate)
		}
	}
}

func TestImageUpdateAndConvexPolygon(t *testing.T) {
	image := NewImage(2, 1, []byte{1, 2, 3, 4, 5, 6, 7, 8})
	image.UpdateImage(R(1, 0, 1, 1), []byte{9, 10, 11, 12})
	if got := pixel(image, 1, 0); got != (Color{9, 10, 11, 12}) {
		t.Fatalf("updated pixel = %#v", got)
	}
	s := NewSurface(4, 4)
	s.FillConvexPolygon([]Point{{0, 0}, {4, 0}, {4, 4}, {0, 4}}, White)
	if pixel(s, 2, 2).A != 255 {
		t.Fatalf("convex polygon coverage failed")
	}
}

func TestConcavePathAndEvenOddHole(t *testing.T) {
	s := NewSurface(8, 8)
	var path Path
	path.MoveTo(Point{0, 0})
	path.LineTo(Point{8, 0})
	path.LineTo(Point{8, 8})
	path.LineTo(Point{0, 8})
	path.Close()
	path.MoveTo(Point{2, 2})
	path.LineTo(Point{6, 2})
	path.LineTo(Point{6, 6})
	path.LineTo(Point{2, 6})
	path.Close()
	s.FillPath(&path, FillEvenOdd, White)
	if pixel(s, 1, 1).A != 255 || pixel(s, 3, 3).A != 0 {
		t.Fatal("even-odd path fill failed")
	}
}

func TestQuadraticPathFill(t *testing.T) {
	s := NewSurface(64, 64)
	var path Path
	path.MoveTo(Point{X: 8, Y: 48})
	path.QuadTo(Point{X: 32, Y: 8}, Point{X: 56, Y: 48})
	path.LineTo(Point{X: 48, Y: 56})
	path.QuadTo(Point{X: 32, Y: 32}, Point{X: 16, Y: 56})
	path.Close()
	s.FillPath(&path, FillEvenOdd, White)
	if pixel(s, 32, 40).A != 255 || pixel(s, 4, 4).A != 0 {
		t.Fatal("quadratic path fill failed")
	}
}

func TestTranslatedPathFill(t *testing.T) {
	s := NewSurface(16, 16)
	s.SetTranslation(4, 5)
	var path Path
	path.MoveTo(Point{0, 0})
	path.LineTo(Point{4, 0})
	path.LineTo(Point{4, 4})
	path.LineTo(Point{0, 4})
	path.Close()
	s.FillPath(&path, FillNonZero, White)
	if pixel(s, 5, 6).A != 255 || pixel(s, 1, 1).A != 0 {
		t.Fatal("translated path fill failed")
	}
}

func TestA8MaskLinearSamplingAndEllipse(t *testing.T) {
	mask := NewMask(2, 1, []byte{0, 255})
	s := NewSurface(8, 8)
	s.DrawImage(mask, R(0, 0, 2, 1), R(0, 0, 4, 2), SamplingLinear, RGBA(255, 0, 0, 255))
	if pixel(s, 0, 0).A >= pixel(s, 3, 0).A || pixel(s, 3, 0).R == 0 {
		t.Fatal("A8 linear image sampling failed")
	}
	s.Clear(Transparent)
	s.FillEllipse(R(1, 1, 6, 4), White)
	if pixel(s, 4, 3).A != 255 || pixel(s, 0, 0).A != 0 {
		t.Fatal("ellipse coverage failed")
	}
}

func TestAffineImageDraw(t *testing.T) {
	image := NewImage(1, 1, []byte{255, 0, 0, 255})
	s := NewSurface(8, 8)
	matrix := Mat2x3{A: 1, B: 0, C: 0.5, D: 1, TX: 1, TY: 2}
	s.SetTransform(&matrix)
	s.DrawImage(image, R(0, 0, 1, 1), R(0, 0, 2, 2), SamplingNearest, White)
	if pixel(s, 2, 3).R != 255 || pixel(s, 0, 0).A != 0 {
		t.Fatal("affine image drawing failed")
	}
}

func TestTextMetricsUTF8AndGlyphDrawing(t *testing.T) {
	font := NewBuiltinFont(1)
	metrics := MeasureText(font, "Hi, 世界")
	if metrics.Width != 36 || metrics.Height != 10 {
		t.Fatalf("text metrics = %#v", metrics)
	}
	s := NewSurface(40, 12)
	s.DrawText(font, Point{X: 1, Y: 8}, "Hi!", White)
	if pixel(s, 1, 1).A == 0 || pixel(s, 20, 1).A != 0 {
		t.Fatal("builtin text rendering failed")
	}
	mask := NewMask(1, 1, []byte{128})
	s.DrawGlyphRun(Point{X: 30, Y: 2}, []Glyph{{Mask: mask, Source: R(0, 0, 1, 1)}}, RGBA(255, 0, 0, 255))
	if pixel(s, 30, 2).R != 128 {
		t.Fatal("glyph mask rendering failed")
	}
}

func TestBuiltinFontPreservesLowercaseShapes(t *testing.T) {
	upper := glyphRows('A')
	lower := glyphRows('a')
	if lower == upper {
		t.Fatal("builtin font folded lowercase into uppercase")
	}
	if lower[0] != 0 || lower[1] != 0 || lower[2] == 0 {
		t.Fatalf("lowercase a rows = %#v", lower)
	}
	if glyphRow('a', 0) != 0 || glyphRow('A', 0) == 0 {
		t.Fatal("RENVO runtime glyph table folded lowercase into uppercase")
	}
}

func putTestU16(data []byte, at int, value int) {
	data[at] = byte(value >> 8)
	data[at+1] = byte(value)
}

func putTestU32(data []byte, at int, value int) {
	data[at] = byte(value >> 24)
	data[at+1] = byte(value >> 16)
	data[at+2] = byte(value >> 8)
	data[at+3] = byte(value)
}

func appendTestU16(data []byte, value int) []byte {
	return append(data, byte(value>>8), byte(value))
}

func testTrueTypeData() []byte {
	cmap := make([]byte, 44)
	putTestU16(cmap, 2, 1)
	putTestU16(cmap, 4, 3)
	putTestU16(cmap, 6, 1)
	putTestU32(cmap, 8, 12)
	putTestU16(cmap, 12, 4)
	putTestU16(cmap, 14, 32)
	putTestU16(cmap, 18, 4)
	putTestU16(cmap, 20, 4)
	putTestU16(cmap, 22, 1)
	putTestU16(cmap, 26, 'B')
	putTestU16(cmap, 28, 0xffff)
	putTestU16(cmap, 32, 'A')
	putTestU16(cmap, 34, 0xffff)
	putTestU16(cmap, 36, 0xffc0)
	putTestU16(cmap, 38, 1)

	head := make([]byte, 54)
	putTestU32(head, 0, 0x00010000)
	putTestU16(head, 18, 1000)
	putTestU16(head, 36, 0xffce)
	putTestU16(head, 40, 600)
	putTestU16(head, 42, 700)

	hhea := make([]byte, 36)
	putTestU32(hhea, 0, 0x00010000)
	putTestU16(hhea, 4, 800)
	putTestU16(hhea, 6, 0xff38)
	putTestU16(hhea, 8, 100)
	putTestU16(hhea, 34, 3)

	hmtx := make([]byte, 12)
	putTestU16(hmtx, 0, 500)
	putTestU16(hmtx, 4, 700)
	putTestU16(hmtx, 8, 700)

	maxp := make([]byte, 6)
	putTestU32(maxp, 0, 0x00010000)
	putTestU16(maxp, 4, 3)

	glyf := make([]byte, 0, 48)
	glyf = appendTestU16(glyf, 1)
	glyf = appendTestU16(glyf, 0)
	glyf = appendTestU16(glyf, 0)
	glyf = appendTestU16(glyf, 600)
	glyf = appendTestU16(glyf, 700)
	glyf = appendTestU16(glyf, 2)
	glyf = appendTestU16(glyf, 0)
	glyf = append(glyf, 1, 1, 1)
	glyf = appendTestU16(glyf, 0)
	glyf = appendTestU16(glyf, 300)
	glyf = appendTestU16(glyf, 300)
	glyf = appendTestU16(glyf, 0)
	glyf = appendTestU16(glyf, 700)
	glyf = appendTestU16(glyf, -700)
	glyf = append(glyf, 0)
	glyf = appendTestU16(glyf, -1)
	glyf = appendTestU16(glyf, -50)
	glyf = appendTestU16(glyf, 0)
	glyf = appendTestU16(glyf, 550)
	glyf = appendTestU16(glyf, 700)
	glyf = appendTestU16(glyf, 3)
	glyf = appendTestU16(glyf, 1)
	glyf = appendTestU16(glyf, -50)
	glyf = appendTestU16(glyf, 0)

	loca := make([]byte, 8)
	putTestU16(loca, 0, 0)
	putTestU16(loca, 2, 0)
	putTestU16(loca, 4, 15)
	putTestU16(loca, 6, 24)

	names := []string{"cmap", "glyf", "head", "hhea", "hmtx", "loca", "maxp"}
	tables := [][]byte{cmap, glyf, head, hhea, hmtx, loca, maxp}
	header := 12 + len(names)*16
	total := header
	for i := 0; i < len(tables); i++ {
		total += (len(tables[i]) + 3) & ^3
	}
	font := make([]byte, total)
	putTestU32(font, 0, 0x00010000)
	putTestU16(font, 4, len(names))
	offset := header
	for i := 0; i < len(names); i++ {
		record := 12 + i*16
		copy(font[record:record+4], []byte(names[i]))
		putTestU32(font, record+8, offset)
		putTestU32(font, record+12, len(tables[i]))
		copy(font[offset:], tables[i])
		offset += (len(tables[i]) + 3) & ^3
	}
	return font
}

func TestTrueTypeFontAntialiasingMetricsAndCache(t *testing.T) {
	if _, err := NewTrueTypeFont([]byte{0, 1, 2}, 20); err == nil {
		t.Fatal("short TrueType data was accepted")
	}
	font, err := NewTrueTypeFont(testTrueTypeData(), 20)
	if err != nil {
		t.Fatal(err)
	}
	if font.Metrics.Ascent != 16 || font.Metrics.Descent != 4 || font.Metrics.LineGap != 2 {
		t.Fatalf("font metrics = %#v", font.Metrics)
	}
	metrics := MeasureText(font, "AB")
	if metrics.Width != 28 || metrics.Height != 22 {
		t.Fatalf("text metrics = %#v", metrics)
	}
	surface := NewSurface(48, 28)
	surface.DrawText(font, Point{X: 4, Y: 20}, "AB", White)
	partial := 0
	covered := 0
	for y := 0; y < surface.Height; y++ {
		for x := 0; x < surface.Width; x++ {
			alpha := pixel(surface, x, y).A
			if alpha != 0 {
				covered++
			}
			if alpha > 0 && alpha < 255 {
				partial++
			}
		}
	}
	if covered == 0 || partial == 0 {
		t.Fatalf("TrueType coverage: covered=%d partial=%d", covered, partial)
	}
	if len(font.glyphs) != 2 {
		t.Fatalf("glyph cache size = %d", len(font.glyphs))
	}
	surface.DrawText(font, Point{X: 4, Y: 20}, "BA", White)
	if len(font.glyphs) != 2 {
		t.Fatalf("glyph cache grew after reuse: %d", len(font.glyphs))
	}
	scaled := NewSurface(96, 56)
	scaled.setDeviceScale(2)
	scaled.ResetTransform()
	scaled.DrawText(font, Point{X: 4, Y: 20}, "AB", White)
	if len(font.glyphs) != 4 {
		t.Fatalf("scaled glyph cache size = %d, want 4", len(font.glyphs))
	}
	placedAtDeviceScale := false
	for y := 0; y < scaled.Height; y++ {
		for x := 36; x < scaled.Width; x++ {
			if pixel(scaled, x, y).A != 0 {
				placedAtDeviceScale = true
			}
		}
	}
	if !placedAtDeviceScale {
		t.Fatal("scaled TrueType text was not placed in device coordinates")
	}
}

func TestScaledDamageAndClipUseDeviceCoordinates(t *testing.T) {
	surface := NewSurface(32, 32)
	surface.ResetDirty()
	surface.setDeviceScale(2)
	surface.ResetTransform()
	surface.BeginDamage(R(2, 3, 4, 5))
	surface.EndDamage()
	dirty, ok := surface.DirtyRect()
	if !ok || dirty != R(4, 6, 8, 10) {
		t.Fatalf("scaled dirty rect = %#v, %v", dirty, ok)
	}
	surface.PushClipRect(R(2, 2, 2, 2))
	surface.FillRect(R(0, 0, 10, 10), White)
	surface.PopClip()
	if pixel(surface, 3, 4).A != 0 || pixel(surface, 4, 4).A != 255 || pixel(surface, 7, 7).A != 255 || pixel(surface, 8, 7).A != 0 {
		t.Fatal("scaled clip did not cover exactly the transformed rectangle")
	}
}

func TestDeviceScaledImageUsesAxisAlignedSampling(t *testing.T) {
	surface := NewSurface(8, 6)
	surface.setDeviceScale(2)
	mask := NewMask(2, 1, []byte{255, 0})
	source := R(0, 0, 2, 1)
	destination := R(1, 1, 2, 1)
	if !surface.drawImageAxisAligned(mask, source, destination, SamplingNearest, White) {
		t.Fatal("device-scaled image missed the axis-aligned path")
	}
	if pixel(surface, 1, 2).A != 0 || pixel(surface, 2, 2).A != 255 || pixel(surface, 3, 3).A != 255 || pixel(surface, 4, 2).A != 0 {
		t.Fatal("device-scaled mask sampled outside its transformed destination")
	}
}

func TestWindowReadPixelsReturnsIndependentTopDownImage(t *testing.T) {
	window := NewWindow(WindowOptions{Width: 3, Height: 2, Hidden: true})
	if window == nil {
		t.Fatal("window was not created")
	}
	window.Surface().Clear(Black)
	window.Surface().FillRect(R(1, 0, 1, 1), RGBA(255, 0, 0, 255))
	capture := window.ReadPixels()
	if capture == nil || capture.Width != 3 || capture.Height != 2 {
		t.Fatalf("capture = %#v", capture)
	}
	if pixel(capture, 1, 0).R != 255 || pixel(capture, 1, 1).R != 0 {
		t.Fatal("capture is not top-down")
	}
	window.Surface().Clear(White)
	if pixel(capture, 0, 0) != Black {
		t.Fatal("capture aliases the window surface")
	}
	window.Close()
	if window.ReadPixels() != nil {
		t.Fatal("closed window produced a capture")
	}
}

func TestEncodePPM(t *testing.T) {
	image := NewImage(2, 1, []byte{255, 1, 2, 255, 3, 254, 4, 255})
	got := image.EncodePPM()
	want := []byte{'P', '6', '\n', '2', ' ', '1', '\n', '2', '5', '5', '\n', 255, 1, 2, 3, 254, 4}
	if len(got) != len(want) {
		t.Fatalf("PPM length = %d, want %d", len(got), len(want))
	}
	for i := 0; i < len(want); i++ {
		if got[i] != want[i] {
			t.Fatalf("PPM byte %d = %d, want %d", i, got[i], want[i])
		}
	}
	if NewMask(1, 1, []byte{255}).EncodePPM() != nil {
		t.Fatal("A8 image encoded as PPM")
	}
}

func TestPlatformKeyNormalization(t *testing.T) {
	if windowsKeyFromVirtual(37) != KeyLeft || windowsKeyFromVirtual(66) != KeyB || windowsKeyFromVirtual(78) != KeyN || windowsKeyFromVirtual(79) != KeyO || windowsKeyFromVirtual(81) != KeyQ || windowsKeyFromVirtual(83) != KeyS || windowsKeyFromVirtual(-1) != KeyUnknown {
		t.Fatal("Windows key normalization drifted")
	}
	if darwinKeyFromCode(123) != KeyLeft || darwinKeyFromCode(11) != KeyB || darwinKeyFromCode(45) != KeyN || darwinKeyFromCode(31) != KeyO || darwinKeyFromCode(12) != KeyQ || darwinKeyFromCode(1) != KeyS || darwinKeyFromCode(-1) != KeyUnknown {
		t.Fatal("Darwin key normalization drifted")
	}
}

func TestNavigationKeysCannotBecomeTextInput(t *testing.T) {
	navigation := []Key{
		KeyBackspace, KeyDelete, KeyEscape,
		KeyLeft, KeyRight, KeyUp, KeyDown,
		KeyHome, KeyEnd, KeyPageUp, KeyPageDown,
	}
	for _, key := range navigation {
		if got := textInputForKey(key, "\xef\x9c\x80"); got != "" {
			t.Fatalf("navigation key %d produced text %q", key, got)
		}
	}

	textKeys := []Key{KeyUnknown, KeyEnter, KeyTab, KeySpace, KeyA, KeyS}
	for _, key := range textKeys {
		if got := textInputForKey(key, "x"); got != "x" {
			t.Fatalf("text key %d discarded text input", key)
		}
	}
}

func TestSurfaceDamageRegionsPreserveDisjointUpdates(t *testing.T) {
	surface := NewSurface(100, 60)
	surface.ResetDirty()
	surface.BeginDamage(R(2, 3, 4, 5))
	surface.FillRect(R(2, 3, 4, 5), White)
	surface.EndDamage()
	surface.BeginDamage(R(80, 40, 3, 2))
	surface.FillRect(R(80, 40, 3, 2), White)
	surface.EndDamage()
	regions := surface.DirtyRects()
	if len(regions) != 2 || regions[0] != R(2, 3, 4, 5) || regions[1] != R(80, 40, 3, 2) {
		t.Fatalf("damage regions = %#v", regions)
	}
	bounding, ok := surface.DirtyRect()
	if !ok || bounding != (Rect{MinX: 2, MinY: 3, MaxX: 83, MaxY: 42}) {
		t.Fatalf("damage bounding rect = %#v, %v", bounding, ok)
	}

	surface.ResetDirty()
	surface.FillRect(R(7, 8, 1, 1), White)
	regions = surface.DirtyRects()
	if len(regions) != 1 || regions[0] != R(7, 8, 1, 1) {
		t.Fatalf("single-pixel immediate damage = %#v", regions)
	}
}
