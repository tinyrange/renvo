//go:build renvo

package sync

// Mutex is a mutual exclusion lock. Its zero value is unlocked.
type Mutex struct {
	locked  bool
	waiters int
	wake    chan int
}

// Lock locks m. If the lock is already in use, the calling goroutine blocks
// until the mutex is available.
func (m *Mutex) Lock() {
	if !m.locked {
		m.locked = true
		return
	}
	if m.wake == nil {
		m.wake = make(chan int)
	}
	m.waiters++
	<-m.wake
}

// TryLock tries to lock m and reports whether it succeeded.
func (m *Mutex) TryLock() bool {
	if m.locked {
		return false
	}
	m.locked = true
	return true
}

// Unlock unlocks m. Unlocking an unlocked mutex panics.
func (m *Mutex) Unlock() {
	if !m.locked {
		panic("sync: unlock of unlocked mutex")
	}
	if m.waiters == 0 {
		m.locked = false
		return
	}
	m.waiters--
	m.wake <- 1
}

// RWMutex is a reader/writer mutual exclusion lock. The lock can be held by
// any number of readers or by one writer. Its zero value is unlocked.
type RWMutex struct {
	writer        bool
	readers       int
	readerWaiters int
	writerWaiters int
	readerWake    chan int
	writerWake    chan int
}

// Lock locks m for writing.
func (m *RWMutex) Lock() {
	if !m.writer && m.readers == 0 {
		m.writer = true
		return
	}
	if m.writerWake == nil {
		m.writerWake = make(chan int)
	}
	m.writerWaiters++
	<-m.writerWake
}

// TryLock tries to lock m for writing and reports whether it succeeded.
func (m *RWMutex) TryLock() bool {
	if m.writer || m.readers != 0 {
		return false
	}
	m.writer = true
	return true
}

// Unlock unlocks m for writing. Unlocking a mutex not held for writing
// panics.
func (m *RWMutex) Unlock() {
	if !m.writer {
		panic("sync: unlock of unlocked RWMutex")
	}
	m.writer = false
	m.wakeWaiters()
}

// RLock locks m for reading.
func (m *RWMutex) RLock() {
	if !m.writer && m.writerWaiters == 0 {
		m.readers++
		return
	}
	if m.readerWake == nil {
		m.readerWake = make(chan int)
	}
	m.readerWaiters++
	<-m.readerWake
}

// TryRLock tries to lock m for reading and reports whether it succeeded.
func (m *RWMutex) TryRLock() bool {
	if m.writer || m.writerWaiters != 0 {
		return false
	}
	m.readers++
	return true
}

// RUnlock undoes one RLock call. Calling it without a read lock panics.
func (m *RWMutex) RUnlock() {
	if m.readers == 0 {
		panic("sync: RUnlock of unlocked RWMutex")
	}
	m.readers--
	if m.readers == 0 {
		m.wakeWaiters()
	}
}

// RLocker returns a Locker interface implemented by RLock and RUnlock.
func (m *RWMutex) RLocker() Locker { return readLocker{mutex: m} }

type readLocker struct{ mutex *RWMutex }

func (r readLocker) Lock()   { r.mutex.RLock() }
func (r readLocker) Unlock() { r.mutex.RUnlock() }

func (m *RWMutex) wakeWaiters() {
	if m.writer || m.readers != 0 {
		return
	}
	if m.writerWaiters > 0 {
		m.writerWaiters--
		m.writer = true
		m.writerWake <- 1
		return
	}
	wake := m.readerWaiters
	m.readerWaiters = 0
	m.readers += wake
	for i := 0; i < wake; i++ {
		m.readerWake <- 1
	}
}

// WaitGroup waits for a collection of tasks to finish. Its zero value has no
// tasks to wait for.
type WaitGroup struct {
	count   int
	waiters int
	wake    chan int
}

// Add adds delta to the task count. A negative task count panics.
func (w *WaitGroup) Add(delta int) {
	next := w.count + delta
	if next < 0 {
		panic("sync: negative WaitGroup counter")
	}
	w.count = next
	if next != 0 || w.waiters == 0 {
		return
	}
	wake := w.waiters
	w.waiters = 0
	for i := 0; i < wake; i++ {
		w.wake <- 1
	}
}

// Done decrements the task count by one.
func (w *WaitGroup) Done() { w.Add(-1) }

// Go starts f in a new goroutine and adds it to the WaitGroup. When f returns,
// it is removed from the task count.
func (w *WaitGroup) Go(f func()) {
	w.Add(1)
	go func() {
		defer w.Done()
		f()
	}()
}

// Wait blocks until the task count becomes zero.
func (w *WaitGroup) Wait() {
	if w.count == 0 {
		return
	}
	if w.wake == nil {
		w.wake = make(chan int)
	}
	w.waiters++
	<-w.wake
}

// Once performs exactly one call to the function passed to Do. A panic from f
// still marks the Once as complete, matching Go's sync.Once.
type Once struct {
	done bool
	lock Mutex
}

// Do calls f if and only if Do has not previously been called for this Once.
func (o *Once) Do(f func()) {
	if o.done {
		return
	}
	o.lock.Lock()
	if o.done {
		o.lock.Unlock()
		return
	}
	defer func() {
		o.done = true
		o.lock.Unlock()
	}()
	f()
}

// Cond is a condition variable associated with L.
type Cond struct {
	L       Locker
	waiters []chan int
}

// NewCond returns a condition variable using l as its lock.
func NewCond(l Locker) *Cond { return &Cond{L: l} }

// Wait atomically unlocks c.L and suspends the caller. On return it locks c.L
// again. Callers should test their condition in a loop.
func (c *Cond) Wait() {
	wake := make(chan int)
	c.waiters = append(c.waiters, wake)
	c.L.Unlock()
	<-wake
	c.L.Lock()
}

// Signal wakes one goroutine waiting on c, if any.
func (c *Cond) Signal() {
	if len(c.waiters) == 0 {
		return
	}
	wakeConditionWaiter(c.waiters[0])
	copy(c.waiters, c.waiters[1:])
	c.waiters = c.waiters[:len(c.waiters)-1]
}

// Broadcast wakes every goroutine waiting on c.
func (c *Cond) Broadcast() {
	for len(c.waiters) > 0 {
		wakeConditionWaiter(c.waiters[0])
		c.waiters = c.waiters[1:]
	}
}

func wakeConditionWaiter(waiter chan int) { waiter <- 1 }
