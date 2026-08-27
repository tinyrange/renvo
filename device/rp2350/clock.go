package rp2350

import "renvo.dev/device/mmio"

const timerRawLow = uintptr(0x400b0028)

// SystemTimer is the RP2350 one-microsecond monotonic counter.
type SystemTimer struct{}

// Ticks returns the low 32 bits of the monotonic hardware counter.
func (SystemTimer) Ticks() uint32 { return mmio.Load32(timerRawLow) }

// TicksPerSecond returns the timer's fixed one-megahertz frequency.
func (SystemTimer) TicksPerSecond() uint32 { return 1000000 }
