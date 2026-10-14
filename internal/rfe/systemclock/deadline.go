package systemclock

// NativeDeadline converts an oscillator target to the same host-monotonic
// counter used inside foreign sessions. Controlled/custom sources and paused
// clocks deliberately have no native deadline capability. No elapsed time is
// synthesized from instruction retirement.
func (c *Clock) NativeDeadline(target uint64) uint64 {
	if c == nil || c.nativeStart == 0 || c.pauseDepth != 0 || c.withheld != 0 {
		return 0
	}
	seconds, fraction := target/c.frequency, target%c.frequency
	if seconds > (^uint64(0)-c.nativeStart)/1000000000 {
		return 0
	}
	ns := seconds * 1000000000
	remainder := (fraction*1000000000 + c.frequency - 1) / c.frequency
	if remainder > ^uint64(0)-ns {
		return 0
	}
	ns += remainder
	if ns > ^uint64(0)-c.nativeStart {
		return 0
	}
	return c.nativeStart + ns
}
