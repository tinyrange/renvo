//go:build browser

package systemclock

// Realtime is deliberately not selected by the interpreter-first browser host.
// Never pretend that instruction retirement is an elapsed host-time source.
func New(frequency uint64) *Clock {
	panic("browser realtime clock requires an explicit host nanosecond source")
}
func (c *Clock) PausedDuration() int64 { return c.pausedNanoseconds() }
