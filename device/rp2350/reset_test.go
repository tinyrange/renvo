package rp2350

import "testing"

func TestPeripheralResetMappings(t *testing.T) {
	if resetsBase != 0x40020000 || ioBank0Reset != 1<<6 ||
		padsBank0Reset != 1<<9 || timer0Reset != 1<<23 {
		t.Fatal("unexpected RP2350 reset mapping")
	}
}

func TestSystemTimerTickMapping(t *testing.T) {
	if timerRawLow != 0x400b0028 || timer0Tick != 0x40108018 {
		t.Fatal("unexpected RP2350 timer tick mapping")
	}
}
