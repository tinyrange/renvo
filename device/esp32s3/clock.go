package esp32s3

import "renvo.dev/device/mmio"

const (
	systemTimerOperation = uintptr(0x60023004)
	systemTimerValueLow  = uintptr(0x60023044)
	systemTimerValid     = uint32(1 << 29)
	systemTimerUpdate    = uint32(1 << 30)
)

// SystemTimer is the ESP32-S3 16 MHz monotonic counter.
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

// DelayMicroseconds busy-waits for at least microseconds.
func (*SystemTimer) DelayMicroseconds(microseconds uint32) {
	timer := SystemTimer{}
	started := timer.Ticks()
	for timer.Ticks()-started < microseconds*16 {
	}
}

// DelayMilliseconds busy-waits for at least milliseconds.
func (timer *SystemTimer) DelayMilliseconds(milliseconds uint32) {
	for milliseconds != 0 {
		timer.DelayMicroseconds(1000)
		milliseconds--
	}
}
