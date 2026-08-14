// Package clock defines portable monotonic timing capabilities.
package clock

// Source is a monotonic tick counter with a fixed frequency.
type Source interface {
	Ticks() uint32
	TicksPerSecond() uint32
}

// Clock adds wrap-safe busy-wait helpers to a Source. Freestanding programs
// explicitly poll because Renvo does not require a scheduler.
type Clock struct {
	source Source
}

// New returns a clock backed by source.
func New(source Source) Clock { return Clock{source: source} }

// Ticks returns the current low 32 bits of the monotonic counter.
func (c *Clock) Ticks() uint32 { return c.source.Ticks() }

// Milliseconds returns the low 32 bits of the monotonic time in milliseconds.
// It is intended for short wrap-safe elapsed-time measurements.
func (c *Clock) Milliseconds() uint32 {
	return c.Ticks() / (c.source.TicksPerSecond() / 1000)
}

// DelayMicroseconds waits for at least microseconds.
func (c *Clock) DelayMicroseconds(microseconds uint32) {
	started := c.Ticks()
	ticks := microseconds * (c.source.TicksPerSecond() / 1000000)
	for c.Ticks()-started < ticks {
	}
}

// DelayMilliseconds waits without overflowing the microsecond conversion.
func (c *Clock) DelayMilliseconds(milliseconds uint32) {
	for milliseconds != 0 {
		c.DelayMicroseconds(1000)
		milliseconds--
	}
}

// DelayUntil waits until microseconds have elapsed from a prior Ticks value.
func (c *Clock) DelayUntil(started, microseconds uint32) {
	ticks := microseconds * (c.source.TicksPerSecond() / 1000000)
	for c.Ticks()-started < ticks {
	}
}
