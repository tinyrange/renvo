package esprmt

import "testing"

func TestSymbols(t *testing.T) {
	if got, want := symbol(false), uint32(3|1<<15|9<<16); got != want {
		t.Fatalf("zero symbol = %#x, want %#x", got, want)
	}
	if got, want := symbol(true), uint32(9|1<<15|3<<16); got != want {
		t.Fatalf("one symbol = %#x, want %#x", got, want)
	}
}

func TestEncoding(t *testing.T) {
	symbols := encode([]byte{0x81, 0x42}, nil)
	if len(symbols) != 17 || symbols[16] != 0 {
		t.Fatalf("encoded length/terminator = %d/%#x, want 17/0", len(symbols), symbols[len(symbols)-1])
	}
	for byteIndex, value := range []byte{0x81, 0x42} {
		for bit := 0; bit < 8; bit++ {
			want := symbol(value&(uint8(1)<<(7-bit)) != 0)
			if got := symbols[byteIndex*8+bit]; got != want {
				t.Fatalf("byte %d bit %d = %#x, want %#x", byteIndex, bit, got, want)
			}
		}
	}
}

func TestEncodingReusesCapacity(t *testing.T) {
	buffer := make([]uint32, 0, 17)
	symbols := encode([]byte{1, 2}, buffer)
	if len(symbols) != 17 || cap(symbols) != cap(buffer) {
		t.Fatalf("reused symbols have len/cap %d/%d, want 17/%d", len(symbols), cap(symbols), cap(buffer))
	}
}
