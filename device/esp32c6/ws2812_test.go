package esp32c6

import "testing"

func TestRMTRegisterAddresses(t *testing.T) {
	if rmtConfig != 0x60006010 {
		t.Fatalf("RMT channel zero config = %#x", rmtConfig)
	}
	if rmtMemory != 0x60006400 {
		t.Fatalf("RMT channel zero memory = %#x", rmtMemory)
	}
	if rmtTransmitLimit != 0x60006058 {
		t.Fatalf("RMT channel zero transmit limit = %#x", rmtTransmitLimit)
	}
	if rmtMemoryWords != 192 {
		t.Fatalf("RMT memory = %d words, want 192", rmtMemoryWords)
	}
}
