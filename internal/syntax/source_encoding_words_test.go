package syntax

import (
	"bytes"
	"testing"
	"unicode/utf8"
)

func TestSourceEncodingWordBoundaries(t *testing.T) {
	for offset := 0; offset < 16; offset++ {
		for size := 0; size < 80; size++ {
			storage := bytes.Repeat([]byte{'a'}, offset+size+16)
			src := storage[offset : offset+size]
			if !validSourceEncoding(src) {
				t.Fatalf("ASCII offset=%d size=%d", offset, size)
			}
			for at := range src {
				for _, value := range []byte{0, 0x7f, 0x80, 0xc0, 0xff} {
					src[at] = value
					want := value != 0 && utf8.Valid(src)
					if validSourceEncoding(src) != want {
						t.Fatalf("offset=%d size=%d at=%d byte=%x", offset, size, at, value)
					}
				}
				src[at] = 'a'
			}
		}
	}
}
