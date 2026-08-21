package time

import "sync"

// Timer represents a single one-shot wakeup. Its C channel receives the
// current time when the timer expires. After a successful Stop the channel
// never fires. The zero value is not useful; create timers with NewTimer.
type Timer struct {
	C <-chan Time

	mu       sync.Mutex
	stopped  bool
	fired    bool
	stop     chan struct{}
	send     chan Time
	cancelFn func() bool
}

// Stop prevents a pending timer from firing. It reports whether the call
// stopped the timer before it expired. Stop is idempotent.
func (t *Timer) Stop() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.stopped {
		return false
	}
	t.stopped = true
	close(t.stop)
	if t.fired {
		return false
	}
	if t.cancelFn != nil {
		return t.cancelFn()
	}
	return true
}

// deliver sends value unless the timer was stopped first. It reports whether
// the value was delivered.
func (t *Timer) deliver(value Time) bool {
	t.mu.Lock()
	if t.stopped {
		t.mu.Unlock()
		return false
	}
	t.mu.Unlock()
	select {
	case <-t.stop:
		return false
	case t.send <- value:
		t.mu.Lock()
		t.fired = true
		t.mu.Unlock()
		return true
	}
}

func expired(deadline Time) bool { return Now().Sub(deadline) >= 0 }
