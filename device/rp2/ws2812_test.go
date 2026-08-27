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

func TestWS2812PIOResetBits(t *testing.T) {
	if rp2040Resets != 0x4000c000 || rp2040PIO0Reset != 1<<10 {
		t.Fatalf("unexpected RP2040 PIO0 reset mapping")
	}
	if rp2350Resets != 0x40020000 || rp2350PIO0Reset != 1<<11 {
		t.Fatalf("unexpected RP2350 PIO0 reset mapping")
	}
}

func TestWS2812PIOHotReloadState(t *testing.T) {
	if pioJoinTX != 1<<30 || pioJoinRX != 1<<31 {
		t.Fatalf("PIO FIFO join bits = %#x, %#x", pioJoinTX, pioJoinRX)
	}
	if ws2812PollLimit <= 0 {
		t.Fatalf("WS2812 poll limit = %d", ws2812PollLimit)
	}
}
