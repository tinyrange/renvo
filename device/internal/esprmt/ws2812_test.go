package esprmt

import (
	"testing"
	"unsafe"
)

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
	if len(symbols) != 18 || symbols[16] != resetSymbol || symbols[17] != 0 {
		t.Fatalf("encoded reset/EOF = %d/%#x/%#x, want 18/%#x/0", len(symbols), symbols[16], symbols[17], resetSymbol)
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
	buffer := make([]uint32, 0, 18)
	symbols := encode([]byte{1, 2}, buffer)
	if len(symbols) != 18 || cap(symbols) != cap(buffer) {
		t.Fatalf("reused symbols have len/cap %d/%d, want 18/%d", len(symbols), cap(symbols), cap(buffer))
	}
}

func TestPaddingMakesWholeRefillChunks(t *testing.T) {
	symbols := []uint32{1, 2, 3, 4, 5}
	padded := pad(symbols, 4)
	if len(padded) != 8 {
		t.Fatalf("padded length = %d, want 8", len(padded))
	}
	for i := 0; i < len(symbols); i++ {
		if padded[i] != symbols[i] {
			t.Fatalf("padded symbol %d = %d, want %d", i, padded[i], symbols[i])
		}
	}
	for i := len(symbols); i < len(padded); i++ {
		if padded[i] != 0 {
			t.Fatalf("padding symbol %d = %d, want zero", i, padded[i])
		}
	}
}

func TestFillCopiesUnrolledBurstAndRemainder(t *testing.T) {
	source := make([]uint32, 61)
	for i := 0; i < len(source); i++ {
		source[i] = uint32(i + 100)
	}
	destination := make([]uint32, 59)
	next := fill(uintptr(unsafe.Pointer(&destination[0])), len(destination), source, 2)
	if next != 61 {
		t.Fatalf("next offset = %d, want 61", next)
	}
	for i := 0; i < len(destination); i++ {
		if got, want := destination[i], source[i+2]; got != want {
			t.Fatalf("destination %d = %d, want %d", i, got, want)
		}
	}
}
