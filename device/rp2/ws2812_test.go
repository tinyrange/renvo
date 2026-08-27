package rp2

import "testing"

func TestWS2812PIOProgramAndTiming(t *testing.T) {
	want := [4]uint32{0x6221, 0x1123, 0x1400, 0xa442}
	if ws2812Program != want {
		t.Fatalf("WS2812 PIO program = %#v, want %#v", ws2812Program, want)
	}
	if pioClockDiv8MHz != 0x00018000 {
		t.Fatalf("WS2812 PIO clock divider = %#x, want 1.5", pioClockDiv8MHz)
	}
}

func TestWS2812PIOUsesSharedRP2Registers(t *testing.T) {
	if pio0Base != 0x50200000 || pioSM0ClockDiv != 0x502000c8 || pioSM0Pin != 0x502000dc {
		t.Fatalf("unexpected PIO0 register map: base=%#x clkdiv=%#x pinctrl=%#x", pio0Base, pioSM0ClockDiv, pioSM0Pin)
	}
}
