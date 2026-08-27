package rp2

import "testing"

func TestPeripheralResetMappings(t *testing.T) {
	if rp2040Resets != 0x4000c000 ||
		rp2040IOBank0Reset != 1<<5 || rp2040PadsBank0Reset != 1<<8 ||
		rp2040PIO0Reset != 1<<10 || rp2040TimerReset != 1<<21 {
		t.Fatal("unexpected RP2040 reset mapping")
	}
	if rp2350Resets != 0x40020000 ||
		rp2350IOBank0Reset != 1<<6 || rp2350PadsBank0Reset != 1<<9 ||
		rp2350PIO0Reset != 1<<11 || rp2350Timer0Reset != 1<<23 {
		t.Fatal("unexpected RP2350 reset mapping")
	}
}

func TestSystemTimerTickMappings(t *testing.T) {
	if rp2040WatchdogTick != 0x4005802c || rp2350Timer0Tick != 0x40108018 {
		t.Fatal("unexpected RP2 timer tick mapping")
	}
}
