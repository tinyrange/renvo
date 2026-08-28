package rp2

import "renvo.dev/device/mmio"

const (
	rp2040WatchdogTick = uintptr(0x4005802c)
	rp2350Timer0Tick   = uintptr(0x40108018)
)

// SystemTimer is the one-microsecond monotonic timer common to RP2 chips.
type SystemTimer struct{}

var timerResetReleased bool

func startSystemTimer() {
	// The ROM clock state does not include the one-microsecond tick generator.
	// ConfigureUSBClock also selects the 12 MHz crystal as clk_ref, giving both
	// families a known divisor for their timer tick.
	ConfigureUSBClock()
	releaseReset(rp2040TimerReset, rp2350Timer0Reset)
	if isRP2350() {
		mmio.Store32(rp2350Timer0Tick+4, 12)
		mmio.Store32(rp2350Timer0Tick, 1)
		for mmio.Load32(rp2350Timer0Tick)&2 == 0 {
		}
		// Delays must continue while a debug probe has either core halted.
		mmio.Store32(0x400b002c, 0)
	} else {
		mmio.Store32(rp2040WatchdogTick, 12|1<<9)
		for mmio.Load32(rp2040WatchdogTick)&(1<<10) == 0 {
		}
	}
}

// Ticks returns the low 32 bits of the monotonic hardware counter.
func (SystemTimer) Ticks() uint32 {
	if !timerResetReleased {
		startSystemTimer()
		timerResetReleased = true
	}
	if isRP2350() {
		return mmio.Load32(0x400b0028)
	}
	return mmio.Load32(0x40054028)
}

// TicksPerSecond returns the timer's fixed one-megahertz frequency.
func (SystemTimer) TicksPerSecond() uint32 { return 1000000 }
