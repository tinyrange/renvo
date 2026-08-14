package esp32p4

import "renvo.dev/device/mmio"

const (
	systemTimerOperation = uintptr(0x500e2004)
	systemTimerValueLow  = uintptr(0x500e2044)
	systemTimerValid     = uint32(1 << 29)
	systemTimerUpdate    = uint32(1 << 30)
)

// SystemTimer is the ESP32-P4 16 MHz monotonic counter.
type SystemTimer struct{}

// Ticks returns the low 32 bits of the monotonic hardware counter.
func (*SystemTimer) Ticks() uint32 {
	mmio.Store32(systemTimerOperation, systemTimerUpdate)
	for mmio.Load32(systemTimerOperation)&systemTimerValid == 0 {
	}
	return mmio.Load32(systemTimerValueLow)
}

// TicksPerSecond returns the fixed 16 MHz counter frequency.
func (*SystemTimer) TicksPerSecond() uint32 { return 16000000 }

// DelayMicroseconds waits using the monotonic system timer.
func (timer *SystemTimer) DelayMicroseconds(microseconds uint32) {
	started := timer.Ticks()
	for timer.Ticks()-started < microseconds*16 {
	}
}

// DelayMilliseconds waits without overflowing the microsecond conversion.
func (timer *SystemTimer) DelayMilliseconds(milliseconds uint32) {
	for milliseconds != 0 {
		timer.DelayMicroseconds(1000)
		milliseconds--
	}
}
