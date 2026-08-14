package esp32s3

import "testing"

func TestRMTRegisterAddresses(t *testing.T) {
	if rmtConfig != 0x60016020 {
		t.Fatalf("RMT channel zero config = %#x", rmtConfig)
	}
	if rmtMemory != 0x60016800 {
		t.Fatalf("RMT channel zero memory = %#x", rmtMemory)
	}
}

func TestRMTSymbols(t *testing.T) {
	if got, want := rmtSymbol(false), uint32(3|1<<15|9<<16); got != want {
		t.Fatalf("zero symbol = %#x, want %#x", got, want)
	}
	if got, want := rmtSymbol(true), uint32(9|1<<15|3<<16); got != want {
		t.Fatalf("one symbol = %#x, want %#x", got, want)
	}
}
