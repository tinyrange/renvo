package esp32s3

import "testing"

func TestRMTRegisterAddresses(t *testing.T) {
	if rmtConfig != 0x60016020 {
		t.Fatalf("RMT channel zero config = %#x", rmtConfig)
	}
	if rmtMemory != 0x60016800 {
		t.Fatalf("RMT channel zero memory = %#x", rmtMemory)
	}
	if rmtTransmitLimit != 0x600160a0 {
		t.Fatalf("RMT channel zero transmit limit = %#x", rmtTransmitLimit)
	}
	if rmtMemoryWords != 384 {
		t.Fatalf("RMT memory = %d words, want 384", rmtMemoryWords)
	}
}

func TestRMTMemoryHoldsFifteenPixelFrame(t *testing.T) {
	if 15*24+1 > rmtMemoryWords {
		t.Fatalf("15-pixel frame does not fit in %d RMT words", rmtMemoryWords)
	}
}
