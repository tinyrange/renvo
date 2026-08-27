package rp2

import "renvo.dev/device/mmio"

// SystemTimer is the one-microsecond monotonic timer common to RP2 chips.
type SystemTimer struct{}

// Ticks returns the low 32 bits of the monotonic hardware counter.
func (SystemTimer) Ticks() uint32 {
	if isRP2350() {
		return mmio.Load32(0x400b0028)
	}
	return mmio.Load32(0x40054028)
}

// TicksPerSecond returns the timer's fixed one-megahertz frequency.
func (SystemTimer) TicksPerSecond() uint32 { return 1000000 }
