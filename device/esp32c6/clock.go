package esp32c6

import "renvo.dev/device/mmio"

const (
	systemTimerOperation = uintptr(0x6000a004)
	systemTimerValueLow  = uintptr(0x6000a044)
	systemTimerValid     = uint32(1 << 29)
	systemTimerUpdate    = uint32(1 << 30)
)

// SystemTimer is the ESP32-C6 16 MHz monotonic counter.
type SystemTimer struct{}

func (SystemTimer) Ticks() uint32 {
	mmio.Store32(systemTimerOperation, systemTimerUpdate)
	for mmio.Load32(systemTimerOperation)&systemTimerValid == 0 {
	}
	return mmio.Load32(systemTimerValueLow)
}

func (SystemTimer) TicksPerSecond() uint32 { return 16000000 }

// DelayMicroseconds waits using wrap-safe unsigned subtraction.
func (timer *SystemTimer) DelayMicroseconds(microseconds uint32) {
	started := timer.Ticks()
	duration := microseconds * 16
	for timer.Ticks()-started < duration {
	}
}

// DelayMilliseconds waits without overflowing a single conversion.
func (timer *SystemTimer) DelayMilliseconds(milliseconds uint32) {
	for milliseconds != 0 {
		timer.DelayMicroseconds(1000)
		milliseconds--
	}
}

// DelayUntil waits until microseconds have elapsed from started.
func (timer *SystemTimer) DelayUntil(started, microseconds uint32) {
	duration := microseconds * 16
	for timer.Ticks()-started < duration {
	}
}
