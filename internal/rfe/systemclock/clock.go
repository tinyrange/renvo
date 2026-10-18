// Package systemclock supplies the monotonic oscillator for hosted system emulators.
// Retirement is an execution/resource counter, never a wall-clock timebase.
package systemclock

// Clock is owned by the machine execution thread. Counter values have the
// device's advertised frequency, independent of instruction tier or throughput.
type Clock struct {
	frequency  uint64
	now        func() int64
	sleep      func(int64)
	last       uint64
	pauseDepth int
	pauseStart int64
	withheld   int64
}

// WithNanosecondSource is the portable oscillator core. Browser hosts can
// provide monotonic nanoseconds and an appropriate waiter without importing
// a desktop time package. The interpreter-only browser machine uses nil Clock.
func WithNanosecondSource(frequency uint64, now func() int64, sleep func(int64)) *Clock {
	if frequency == 0 || frequency > 1000000000 || now == nil || sleep == nil {
		panic("invalid system clock source or frequency")
	}
	return &Clock{frequency: frequency, now: now, sleep: sleep}
}

func (c *Clock) Read() uint64 {
	elapsed := c.now()
	if c.pauseDepth != 0 {
		elapsed = c.pauseStart
	}
	elapsed -= c.withheld
	if elapsed > 0 {
		ns := uint64(elapsed)
		// Divide first: multiplying a long uptime by the frequency overflows.
		ticks := ns/1000000000*c.frequency + ns%1000000000*c.frequency/1000000000
		if ticks > c.last {
			c.last = ticks
		}
	}
	return c.last
}

// WaitUntil waits for actual oscillator time; it does not fabricate elapsed
// time or retire instructions. Long waits are chunked to avoid conversion overflow.
func (c *Clock) WaitUntil(target uint64) {
	if c.pauseDepth != 0 {
		panic("idle wait with stopped translation clock")
	}
	for {
		now := c.Read()
		if now >= target {
			return
		}
		ticks := target - now
		if ticks > c.frequency {
			ticks = c.frequency
		}
		ns := (ticks*1000000000 + c.frequency - 1) / c.frequency
		c.sleep(int64(ns))
	}
}

// Pause/Resume bracket translation or arena reclamation while the vCPU is
// stopped. They do not bracket guest execution, device access or idle waiting.
// Nesting permits region discovery to invoke ordinary cold-block translation.
func (c *Clock) Pause() {
	if c.pauseDepth == 0 {
		c.pauseStart = c.now()
	}
	c.pauseDepth++
}
func (c *Clock) Resume() {
	if c.pauseDepth == 0 {
		panic("unbalanced system clock resume")
	}
	c.pauseDepth--
	if c.pauseDepth == 0 {
		if elapsed := c.now() - c.pauseStart; elapsed > 0 {
			c.withheld += elapsed
		}
	}
}

// PausedDuration reports host time excluded while the vCPU was stopped.
func (c *Clock) pausedNanoseconds() int64 {
	paused := c.withheld
	if c.pauseDepth != 0 {
		if d := c.now() - c.pauseStart; d > 0 {
			paused += d
		}
	}
	return paused
}
