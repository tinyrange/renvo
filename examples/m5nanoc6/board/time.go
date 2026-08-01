package board

const (
	systemTimerOperation  = uintptr(0x6000a004)
	systemTimerValueLow   = uintptr(0x6000a044)
	systemTimerValid      = uint32(1 << 29)
	systemTimerUpdate     = uint32(1 << 30)
	systemTimerTicksPerUS = uint32(16)
)

// TimerTicks returns the low 32 bits of the ESP32-C6's always-running 16 MHz
// system timer. Unsigned subtraction makes intervals shorter than the roughly
// 268-second wraparound safe across that wrap.
func TimerTicks() uint32 {
	store32(systemTimerOperation, systemTimerUpdate)
	for load32(systemTimerOperation)&systemTimerValid == 0 {
	}
	return load32(systemTimerValueLow)
}

func DelayMicroseconds(microseconds uint32) {
	started := TimerTicks()
	duration := microseconds * systemTimerTicksPerUS
	for TimerTicks()-started < duration {
	}
}

// DelayUntil waits until microseconds have elapsed since a TimerTicks snapshot.
// If the deadline has already passed it returns immediately.
func DelayUntil(started uint32, microseconds uint32) {
	duration := microseconds * systemTimerTicksPerUS
	for TimerTicks()-started < duration {
	}
}

func DelayMilliseconds(milliseconds uint32) {
	for milliseconds != 0 {
		DelayMicroseconds(1000)
		milliseconds--
	}
}
