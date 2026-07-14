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
