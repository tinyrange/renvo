//go:build !browser

package systemclock

import "time"

func New(frequency uint64) *Clock {
	start := time.Now()
	return WithSource(frequency, func() time.Duration { return time.Since(start) }, time.Sleep)
}

// Preserve the hosted time.Duration API over the portable nanosecond core.
func WithSource(frequency uint64, now func() time.Duration, sleep func(time.Duration)) *Clock {
	if now == nil || sleep == nil {
		panic("invalid system clock source or frequency")
	}
	return WithNanosecondSource(frequency, func() int64 { return int64(now()) }, func(ns int64) { sleep(time.Duration(ns)) })
}
func (c *Clock) PausedDuration() time.Duration { return time.Duration(c.pausedNanoseconds()) }
