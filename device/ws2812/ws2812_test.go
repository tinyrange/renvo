package ws2812

import "testing"

type recordingTransmitter struct {
	bytes []byte
}

func (t *recordingTransmitter) Transmit(data []byte) bool {
	t.bytes = append(t.bytes[:0], data...)
	return true
}

func TestSetPixelsUsesGRBWireOrder(t *testing.T) {
	transmitter := &recordingTransmitter{}
	strip := New(transmitter)
	if !strip.SetPixels([]RGB{
		{Red: 0x81, Green: 0x42, Blue: 0x24},
		{Red: 0x18, Green: 0x57, Blue: 0x39},
	}) {
		t.Fatal("SetPixels returned false")
	}
	want := []byte{0x42, 0x81, 0x24, 0x57, 0x18, 0x39}
	if len(transmitter.bytes) != len(want) {
		t.Fatalf("wire byte count = %d, want %d", len(transmitter.bytes), len(want))
	}
	for i := 0; i < len(want); i++ {
		if transmitter.bytes[i] != want[i] {
			t.Fatalf("wire byte %d = %#x, want %#x", i, transmitter.bytes[i], want[i])
		}
	}
}

func TestEncodingReusesCapacity(t *testing.T) {
	buffer := make([]byte, 0, 6)
	encoded := encode([]RGB{{Red: 1}, {Blue: 2}}, buffer)
	if len(encoded) != 6 || cap(encoded) != cap(buffer) {
		t.Fatalf("reused bytes have len/cap %d/%d, want 6/%d", len(encoded), cap(encoded), cap(buffer))
	}
}

func TestSetUsesSinglePixel(t *testing.T) {
	transmitter := &recordingTransmitter{}
	strip := New(transmitter)
	strip.Set(0x12, 0x34, 0x56)
	want := [3]byte{0x34, 0x12, 0x56}
	for i := 0; i < len(want); i++ {
		if transmitter.bytes[i] != want[i] {
			t.Fatalf("wire byte %d = %#x, want %#x", i, transmitter.bytes[i], want[i])
		}
	}
}
