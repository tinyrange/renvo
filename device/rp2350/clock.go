package rp2350

import "renvo.dev/device/mmio"

const timerRawLow = uintptr(0x400b0028)
const timer0Tick = uintptr(0x40108018)

// SystemTimer is the RP2350 one-microsecond monotonic counter.
type SystemTimer struct{}

var timerResetReleased bool

func startSystemTimer() {
	// Select the 12 MHz crystal for clk_ref before deriving a one-microsecond
	// TIMER0 tick. These are the RP2350 portions of the vendor clock bring-up.
	mmio.Store32(0x40048000, 0xaa0)
	mmio.Store32(0x4004800c, 47)
	mmio.Store32(0x4004a000, 0xfab000)
	for mmio.Load32(0x40048004)&(1<<31) == 0 {
	}
	mmio.Store32(0x4001303c, 1)
	for mmio.Load32(0x40010044)&1 == 0 {
	}
	mmio.Store32(0x40013030, 3)
	for mmio.Load32(0x40010038)&1 == 0 {
	}
	releaseReset(timer0Reset)
	mmio.Store32(timer0Tick+4, 12)
	mmio.Store32(timer0Tick, 1)
	for mmio.Load32(timer0Tick)&2 == 0 {
	}
	mmio.Store32(0x400b002c, 0)
}

// Ticks returns the low 32 bits of the monotonic hardware counter.
func (SystemTimer) Ticks() uint32 {
	if !timerResetReleased {
		startSystemTimer()
		timerResetReleased = true
	}
	return mmio.Load32(timerRawLow)
}

// TicksPerSecond returns the timer's fixed one-megahertz frequency.
func (SystemTimer) TicksPerSecond() uint32 { return 1000000 }
