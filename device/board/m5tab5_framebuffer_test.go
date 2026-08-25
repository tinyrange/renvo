//go:build m5tab5

package board

import (
	"renvo.dev/std/graphics"
	"testing"
	"unsafe"
)

func TestExternalBytesPreservesAddressAndBounds(t *testing.T) {
	const address = uintptr(0x12345678)
	const size = 4096
	var pixels []byte
	bindExternalBytes(&pixels, address, size)
	if len(pixels) != size || cap(pixels) != size {
		t.Fatalf("external bytes bounds = %d/%d, want %d/%d", len(pixels), cap(pixels), size, size)
	}
	if uintptr(unsafe.Pointer(&pixels[0])) != address {
		t.Fatalf("external bytes address = %#x, want %#x", uintptr(unsafe.Pointer(&pixels[0])), address)
	}
}

func TestDamageBandsSortAndCoalesceRows(t *testing.T) {
	surface := graphics.NewSurfaceBufferFormat(
		DisplayWidth, DisplayHeight, graphics.PixelRGB565,
		make([]byte, framebufferSize),
	)
	surface.ResetDirty()
	surface.MarkUpdated(graphics.R(20, 80, 10, 5))
	surface.MarkUpdated(graphics.R(10, 20, 10, 4))
	surface.MarkUpdated(graphics.R(40, 24, 10, 3))

	bands, count := damageBands(surface)
	if count != 2 {
		t.Fatalf("damage band count = %d, want 2", count)
	}
	if bands[0] != (rowBand{minY: 20, maxY: 27}) ||
		bands[1] != (rowBand{minY: 80, maxY: 85}) {
		t.Fatalf("damage bands = %#v %#v", bands[0], bands[1])
	}
}
