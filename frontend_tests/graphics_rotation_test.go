package frontend_tests

import (
	"bytes"
	. "renvo.dev/std/graphics"
	"testing"
)

// This independent coordinate oracle catches mismatched logical/native stride
// even when drawing and sampling accidentally share the same incorrect mapping.
func rotatedOffset(rotation Rotation, width, height, stride, bpp, x, y int) int {
	switch rotation {
	case Rotation90:
		return (width-1-x)*stride + y*bpp
	case Rotation180:
		return (height-1-y)*stride + (width-1-x)*bpp
	case Rotation270:
		return x*stride + (height-1-y)*bpp
	}
	return y*stride + x*bpp
}

func rotationScene(s *Surface) {
	s.Clear(RGBA(7, 21, 35, 255))
	s.FillRect(R(1, 2, 13, 9), RGBA(210, 40, 90, 255))
	s.PushClipRect(R(3, 4, 23, 20))
	s.FillEllipse(R(-2, 2, 24, 18), RGBA(40, 200, 70, 137))
	s.SetTranslation(3, 1)
	s.DrawText(NewBuiltinFont(1), Point{3, 18}, "Ab!", White)
	s.ResetTransform()
	s.DrawLine(Point{0, 0}, Point{30, 22}, 2, RGBA(200, 80, 30, 230))
	s.PopClip()
	mask := NewMask(4, 2, []byte{0, 128, 255, 255, 255, 42, 0, 255})
	s.DrawGlyphRun(Point{7, 6}, []Glyph{{Mask: mask, Source: R(0, 0, 4, 2)}}, White)
	s.DrawImage(mask, R(0, 0, 4, 2), R(20, 4, 8, 6), SamplingLinear, RGBA(150, 200, 220, 190))
}

func TestRotatedDrawingMatchesUnrotated(t *testing.T) {
	for _, format := range []PixelFormat{PixelRGBA8, PixelRGB565, PixelA8} {
		for rotation := Rotation0; rotation <= Rotation270; rotation++ {
			s := NewRotatedSurface(32, 40, format, rotation)
			plain := NewImageFormat(s.Width, s.Height, format, nil)
			rotationScene(s)
			rotationScene(plain)
			bpp := 4
			if format == PixelRGB565 {
				bpp = 2
			} else if format == PixelA8 {
				bpp = 1
			}
			for y := 0; y < s.Height; y++ {
				for x := 0; x < s.Width; x++ {
					o := rotatedOffset(rotation, s.Width, s.Height, s.Stride, bpp, x, y)
					p := y*plain.Stride + x*bpp
					if !bytes.Equal(s.Pixels[o:o+bpp], plain.Pixels[p:p+bpp]) {
						t.Fatalf("format %d rotation %d pixel %d,%d", format, rotation, x, y)
					}
				}
			}
		}
	}
}

func TestRotationBufferDamageAndSwitch(t *testing.T) {
	pixels := make([]byte, 3*5*2)
	s := NewRotatedSurfaceBuffer(3, 5, PixelRGB565, Rotation90, pixels)
	if s.Width != 5 || s.Height != 3 || s.Stride != 6 || &s.Pixels[0] != &pixels[0] {
		t.Fatal("native storage not reused")
	}
	s.ResetDirty()
	s.FillRect(R(1, 0, 2, 2), White)
	dirty, ok := s.DirtyRectAt(0)
	if !ok || dirty != R(1, 0, 2, 2) || s.NativeRect(dirty) != R(0, 2, 2, 2) {
		t.Fatal("damage mapping", dirty, s.NativeRect(dirty))
	}
	before := append([]byte(nil), pixels...)
	for i := 0; i < 12; i++ {
		if !s.SetRotation(Rotation(i%4)) || &s.Pixels[0] != &pixels[0] || !bytes.Equal(before, pixels) {
			t.Fatal("switch moved pixels")
		}
		if s.NativeWidth() != 3 || s.NativeHeight() != 5 {
			t.Fatal("native extent changed")
		}
	}
	if s.SetRotation(4) {
		t.Fatal("invalid orientation accepted")
	}
	s.Resize(8, 6)
	if s.Width != 8 || s.Height != 6 || s.Rotation() != Rotation270 || s.Stride != 12 {
		t.Fatal("resize lost orientation")
	}
}

func TestRotatedCopiesAndImageUpdates(t *testing.T) {
	for rotation := Rotation0; rotation <= Rotation270; rotation++ {
		s := NewRotatedSurface(8, 10, PixelRGBA8, rotation)
		data := make([]byte, s.Width*s.Height*4)
		for i := range data {
			data[i] = byte(i * 17)
		}
		s.UpdateImage(R(0, 0, Scalar(s.Width), Scalar(s.Height)), data)
		plain := NewImage(s.Width, s.Height, data)
		for _, delta := range [][2]int{{0, 1}, {0, -1}, {1, 0}, {-1, 0}, {1, 1}, {-1, -1}} {
			dx, dy, sx, sy := 1, 1, 1+delta[0], 1+delta[1]
			w, h := s.Width-3, s.Height-3
			s.CopyPixels(dx, dy, sx, sy, w, h)
			plain.CopyPixels(dx, dy, sx, sy, w, h)
			for y := 0; y < s.Height; y++ {
				for x := 0; x < s.Width; x++ {
					a, b := s.PixelOffset(x, y), plain.PixelOffset(x, y)
					if !bytes.Equal(s.Pixels[a:a+4], plain.Pixels[b:b+4]) {
						t.Fatalf("overlap rotation %d delta %v", rotation, delta)
					}
				}
			}
		}
		out := NewSurface(s.Width, s.Height)
		out.SetBlendMode(BlendCopy)
		out.DrawImage(s, R(0, 0, Scalar(s.Width), Scalar(s.Height)), R(0, 0, Scalar(s.Width), Scalar(s.Height)), SamplingNearest, White)
		if !bytes.Equal(out.Pixels, plain.Pixels) {
			t.Fatal("rotated image sampling")
		}
	}
}
