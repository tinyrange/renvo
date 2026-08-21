//go:build !renvo

package time

// Sleep pauses the calling goroutine for at least duration d.
func Sleep(d Duration) {
	deadline := Now().Add(d)
	for !expired(deadline) {
	}
}

// NewTimer creates a one-shot timer backed by a polling goroutine.
func NewTimer(d Duration) *Timer {
	channel := make(chan Time, 1)
	t := &Timer{C: channel, stop: make(chan struct{}), send: channel}
	deadline := Now().Add(d)
	go func() {
		for !expired(deadline) {
			select {
			case <-t.stop:
				return
			default:
			}
		}
		t.deliver(Now())
	}()
	return t
}

// After fires the current time on the returned channel after at least d.
func After(d Duration) <-chan Time { return NewTimer(d).C }
