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
	if rmtClockReset != 0x60006070 {
		t.Fatalf("RMT channel zero clock reset = %#x", rmtClockReset)
	}
	if rmtInterruptMap != 0x600100c4 {
		t.Fatalf("RMT interrupt map = %#x", rmtInterruptMap)
	}
	if plicInterruptEnable != 0x20001000 || plicInterruptOnePriority != 0x20001014 || plicInterruptThreshold != 0x20001090 {
		t.Fatalf("PLIC registers = %#x/%#x/%#x", plicInterruptEnable, plicInterruptOnePriority, plicInterruptThreshold)
	}
	if interruptRefillMailbox != 0x4087ff00 {
		t.Fatalf("interrupt refill mailbox = %#x", interruptRefillMailbox)
	}
	if rmtMemoryBlocks != 1<<16 {
		t.Fatalf("RMT channel zero memory blocks = %#x, want %#x", rmtMemoryBlocks, uint32(1<<16))
	}
	if rmtMemoryWords != 48 {
		t.Fatalf("RMT memory = %d words, want 48", rmtMemoryWords)
	}
	if got, want := rmtClock80MHz|rmtSourceEnable|rmtClockDenominator, uint32(0x00500040); got != want {
		t.Fatalf("RMT source clock = %#x, want %#x", got, want)
	}
	if got, want := rmtIdleLow|rmtDivider10MHz|rmtMemoryBlocks|rmtWrap|rmtCarrierEffective|rmtCarrierOutputHigh, uint32(0x00510850); got != want {
		t.Fatalf("RMT channel config = %#x, want %#x", got, want)
	}
	if got, want := rmtLoopAutoStop|uint32(rmtMemoryWords/2), uint32(0x00200018); got != want {
		t.Fatalf("RMT transmit limit = %#x, want %#x", got, want)
	}
}
