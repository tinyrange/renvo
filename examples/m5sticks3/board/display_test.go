package board

import (
	"testing"
	"unsafe"
)

func TestRGB565(t *testing.T) {
	tests := []struct {
		red, green, blue byte
		want             uint16
	}{
		{0, 0, 0, 0x0000},
		{255, 255, 255, 0xffff},
		{255, 0, 0, 0xf800},
		{0, 255, 0, 0x07e0},
		{0, 0, 255, 0x001f},
	}
	for _, test := range tests {
		if got := rgb565(test.red, test.green, test.blue); got != test.want {
			t.Fatalf("rgb565(%d, %d, %d) = %#04x, want %#04x", test.red, test.green, test.blue, got, test.want)
		}
	}
}

func TestDMABufferIsWordAligned(t *testing.T) {
	var buffer dmaBuffer
	if size := unsafe.Sizeof(buffer); size < DisplayWidth*6 {
		t.Fatalf("DMA buffer size = %d, want at least %d", size, DisplayWidth*6)
	}
	if uintptr(unsafe.Pointer(&buffer[0]))&3 != 0 {
		t.Fatalf("DMA buffer address %p is not word aligned", &buffer[0])
	}
}

func TestDMADescriptorMatchesESP32S3ABI(t *testing.T) {
	var descriptor dmaDescriptor
	if size := unsafe.Sizeof(descriptor); size != 12 {
		t.Fatalf("DMA descriptor size = %d, want 12", size)
	}
}
