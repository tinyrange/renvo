package rp2

import "renvo.dev/device/gpio"

// LED is a concrete RP2 indicator. Keeping the pin concrete also keeps early
// boot usable before interface dispatch and the persistent arena are ready.
type LED struct {
	pin         *Pin
	activeLow   bool
	initialized bool
}

// NewLED describes an LED attached to an RP2 pin.
func NewLED(pin *Pin, activeLow bool) LED {
	return LED{pin: pin, activeLow: activeLow}
}

func (l *LED) initialize() {
	if l.initialized {
		return
	}
	l.pin.Set(l.activeLow)
	_ = l.pin.Configure(gpio.Config{Direction: gpio.Output})
	l.initialized = true
}

// Set turns the LED on or off.
func (l *LED) Set(on bool) {
	l.initialize()
	l.pin.Set(on != l.activeLow)
}

// Toggle changes the current visible state.
func (l *LED) Toggle() {
	l.initialize()
	l.pin.Set(!l.pin.Get())
}

// Clock is the concrete RP2 monotonic clock and busy-wait timer.
type Clock struct{}

// Ticks returns the current one-megahertz hardware tick count.
func (Clock) Ticks() uint32 { return SystemTimer{}.Ticks() }

// DelayMicroseconds waits for at least microseconds.
func (c Clock) DelayMicroseconds(microseconds uint32) {
	started := c.Ticks()
	for c.Ticks()-started < microseconds {
	}
}

// DelayMilliseconds waits without overflowing the microsecond conversion.
func (c Clock) DelayMilliseconds(milliseconds uint32) {
	for milliseconds != 0 {
		c.DelayMicroseconds(1000)
		milliseconds--
	}
}

// DelayUntil waits until microseconds have elapsed from a prior Ticks value.
func (c Clock) DelayUntil(started, microseconds uint32) {
	for c.Ticks()-started < microseconds {
	}
}
