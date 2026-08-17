//go:build !renvo

package sync

import stdsync "sync"

// Mutex is a mutual exclusion lock. Its zero value is unlocked.
type Mutex struct{ value stdsync.Mutex }

// Lock locks m, blocking until it is available.
func (m *Mutex) Lock() { m.value.Lock() }

// TryLock tries to lock m and reports whether it succeeded.
func (m *Mutex) TryLock() bool { return m.value.TryLock() }

// Unlock unlocks m.
func (m *Mutex) Unlock() { m.value.Unlock() }

// RWMutex is a reader/writer mutual exclusion lock. Its zero value is
// unlocked.
type RWMutex struct{ value stdsync.RWMutex }

// Lock locks m for writing.
func (m *RWMutex) Lock() { m.value.Lock() }

// TryLock tries to lock m for writing and reports whether it succeeded.
func (m *RWMutex) TryLock() bool { return m.value.TryLock() }

// Unlock unlocks m for writing.
func (m *RWMutex) Unlock() { m.value.Unlock() }

// RLock locks m for reading.
func (m *RWMutex) RLock() { m.value.RLock() }

// TryRLock tries to lock m for reading and reports whether it succeeded.
func (m *RWMutex) TryRLock() bool { return m.value.TryRLock() }

// RUnlock undoes one RLock call.
func (m *RWMutex) RUnlock() { m.value.RUnlock() }

// RLocker returns a Locker implemented by RLock and RUnlock.
func (m *RWMutex) RLocker() Locker {
	return m.value.RLocker()
}

// WaitGroup waits for a collection of tasks to finish. Its zero value has no
// tasks to wait for.
type WaitGroup struct{ value stdsync.WaitGroup }

// Add adds delta to the task count.
func (w *WaitGroup) Add(delta int) { w.value.Add(delta) }

// Done decrements the task count by one.
func (w *WaitGroup) Done() { w.value.Done() }

// Go starts f in a new goroutine and adds it to the WaitGroup.
func (w *WaitGroup) Go(f func()) { w.value.Go(f) }

// Wait blocks until the task count becomes zero.
func (w *WaitGroup) Wait() { w.value.Wait() }

// Once performs exactly one call to Do's function.
type Once struct{ value stdsync.Once }

// Do calls f if and only if Do has not previously been called for this Once.
func (o *Once) Do(f func()) { o.value.Do(f) }

// Cond is a condition variable associated with L.
type Cond struct {
	L     Locker
	value *stdsync.Cond
}

// NewCond returns a condition variable using l as its lock.
func NewCond(l Locker) *Cond {
	return &Cond{L: l, value: stdsync.NewCond(l)}
}

// Wait unlocks c.L, suspends the caller, and locks c.L again before returning.
func (c *Cond) Wait() { c.value.Wait() }

// Signal wakes one goroutine waiting on c, if any.
func (c *Cond) Signal() { c.value.Signal() }

// Broadcast wakes every goroutine waiting on c.
func (c *Cond) Broadcast() { c.value.Broadcast() }
