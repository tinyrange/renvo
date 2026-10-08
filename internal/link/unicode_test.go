package link

import "testing"

func TestUnicodeSourceASCIIAlignmentAndBounds(t *testing.T) {
	// Exercise every alignment and tail, including non-ASCII bytes immediately
	// outside the selected slice. Every byte position must be examined.
	for offset := 0; offset < 16; offset++ {
		for size := 0; size < 48; size++ {
			storage := make([]byte, offset+size+16)
			for i := range storage {
				storage[i] = 255
			}
			src := storage[offset : offset+size]
			for i := range src {
				src[i] = byte(i % 128)
			}
			if !unicodeSourceASCII(src) {
				t.Fatalf("ASCII rejected: offset=%d size=%d", offset, size)
			}
			for i := range src {
				saved := src[i]
				src[i] = 128
				if unicodeSourceASCII(src) {
					t.Fatalf("high byte missed: offset=%d size=%d index=%d", offset, size, i)
				}
				src[i] = saved
			}
		}
	}
}
