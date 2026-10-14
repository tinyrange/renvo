// Package systemclock supplies the monotonic oscillator for hosted system emulators.
// Retirement is an execution/resource counter, never a wall-clock timebase.
package systemclock

import "time"

// Clock is owned by the machine execution thread. Counter values have the
// device's advertised frequency, independent of instruction tier or throughput.
type Clock struct {
	nativeStart uint64
	frequency   uint64
	now         func() time.Duration
	sleep       func(time.Duration)
	last        uint64
	pauseDepth  int
	pauseStart  time.Duration
	withheld    time.Duration
}

func New(frequency uint64) *Clock {
	if start := nativeNow(); start != 0 {
		c := WithSource(frequency, func() time.Duration {
			now := nativeNow()
			if now < start {
				return 0
			}
			return time.Duration(now - start)
		}, time.Sleep)
		c.nativeStart = start
		return c
	}
	start := time.Now()
	return WithSource(frequency, func() time.Duration { return time.Since(start) }, time.Sleep)
}

// WithSource permits a controlled monotonic source and waiter, notably for
// tests that must verify time/retirement separation without sleeping.
func WithSource(frequency uint64, now func() time.Duration, sleep func(time.Duration)) *Clock {
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
		c.sleep(time.Duration(ns))
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
func (c *Clock) PausedDuration() time.Duration {
	paused := c.withheld
	if c.pauseDepth != 0 {
		if d := c.now() - c.pauseStart; d > 0 {
			paused += d
		}
	}
	return paused
}
