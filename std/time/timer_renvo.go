//go:build renvo

package time

import runtime "renvo.dev/x/runtime"

// Sleep pauses the current task for at least duration d. It requires a
// runtime handler; install one before the first concurrency operation.
func Sleep(d Duration) {
	runtime.NewTimer(int64(d)).Wait()
}

// NewTimer creates a one-shot timer. The timer's channel is buffered, so a
// caller which never reads it does not leak a blocked task after Stop.
func NewTimer(d Duration) *Timer {
	channel := make(chan Time, 1)
	t := &Timer{C: channel, stop: make(chan struct{}), send: channel}
	handle := runtime.NewTimer(int64(d))
	t.cancel = func() bool { return handle.Stop() }
	go func() {
		if !handle.Wait() {
			return
		}
		t.deliver(Now())
	}()
	return t
}

// After fires the current time on the returned channel after at least d.
func After(d Duration) <-chan Time { return NewTimer(d).C }
