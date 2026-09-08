package graphics

import (
	"bytes"
	"testing"
)

func TestNativeRGB565ImageScalingClippingAndRotation(t *testing.T) {
	for _, rotation := range []Rotation{Rotation0, Rotation90, Rotation180, Rotation270} {
		native := NewSurfaceFormat(4, 4, PixelRGB565)
		rgba := NewSurface(4, 4)
		for i := 0; i < 16; i++ {
			color := uint16(i*3917 + 107)
			native.Pixels[i*2], native.Pixels[i*2+1] = byte(color), byte(color>>8)
			rgba.writePixel(i%4, i/4, decodeRGB565(byte(color), byte(color>>8)))
		}
		native.SetRotation(rotation)
		rgba.SetRotation(rotation)
		for _, targetRotation := range []Rotation{Rotation0, Rotation90} {
			got := NewRotatedSurface(6, 6, PixelRGB565, targetRotation)
			want := NewRotatedSurface(6, 6, PixelRGB565, targetRotation)
			for _, s := range []*Surface{got, want} {
				s.Clear(Black)
				s.ResetDirty()
				s.PushClipRect(R(1, 1, 4, 4))
			}
			got.DrawImage(native, R(0, 0, 4, 4), R(-1, 0, 7, 6), SamplingNearest, White)
			want.DrawImage(rgba, R(0, 0, 4, 4), R(-1, 0, 7, 6), SamplingNearest, White)
			if !bytes.Equal(got.Pixels, want.Pixels) {
				t.Fatalf("native blit differs from equivalent RGBA source: rotations %d/%d", rotation, targetRotation)
			}
			dirty, ok := got.DirtyRect()
			if !ok || dirty != R(1, 1, 4, 4) {
				t.Fatalf("damage = %v, %v", dirty, ok)
			}
		}
	}
}

func TestNativeRGB565MixedFormatAlpha(t *testing.T) {
	target := NewSurfaceFormat(2, 1, PixelRGB565)
	if target.Format != PixelRGB565 || len(target.Pixels) != 4 {
		t.Fatal("native surface did not allocate packed storage")
	}
	target.Clear(RGBA(0, 0, 255, 255))
	source := NewSurface(1, 1)
	source.Clear(RGBA(255, 0, 0, 128))
	target.DrawImage(source, R(0, 0, 1, 1), R(0, 0, 1, 1), SamplingNearest, White)
	want := encodeRGB565(Color{R: 128, B: 127, A: 255})
	got := uint16(target.Pixels[0]) | uint16(target.Pixels[1])<<8
	if got != want {
		t.Fatalf("mixed-format source-over = %04x want %04x", got, want)
	}
	if target.Pixels[2] != 0x1f || target.Pixels[3] != 0 {
		t.Fatal("image changed pixel outside destination")
	}
}

func TestNativeRGB565ImageOverlappingStorage(t *testing.T) {
	// Both bytes of a source pixel must be read before either destination byte
	// is written, even when caller-provided views overlap at a byte boundary.
	storage := []byte{0x34, 0x12, 0}
	source := NewSurfaceBufferFormatPreserve(1, 1, PixelRGB565, storage[:2])
	target := NewSurfaceBufferFormatPreserve(1, 1, PixelRGB565, storage[1:])
	target.DrawImage(source, R(0, 0, 1, 1), R(0, 0, 1, 1), SamplingNearest, White)
	if storage[1] != 0x34 || storage[2] != 0x12 {
		t.Fatalf("overlapping image = %x", storage)
	}
}
