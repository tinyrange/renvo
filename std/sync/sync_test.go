package sync

import "testing"

func TestMutexAndWaitGroup(t *testing.T) {
	var mutex Mutex
	var group WaitGroup
	value := 0
	for i := 0; i < 8; i++ {
		group.Go(func() {
			for j := 0; j < 100; j++ {
				mutex.Lock()
				value++
				mutex.Unlock()
			}
		})
	}
	group.Wait()
	if value != 800 {
		t.Fatalf("value = %d", value)
	}
	if !mutex.TryLock() {
		t.Fatal("TryLock failed on unlocked mutex")
	}
	if mutex.TryLock() {
		t.Fatal("TryLock succeeded twice")
	}
	mutex.Unlock()
}

func TestRWMutex(t *testing.T) {
	var mutex RWMutex
	mutex.RLock()
	if !mutex.TryRLock() {
		t.Fatal("second reader failed")
	}
	if mutex.TryLock() {
		t.Fatal("writer acquired read-locked mutex")
	}
	mutex.RUnlock()
	mutex.RUnlock()
	if !mutex.TryLock() {
		t.Fatal("writer failed")
	}
	if mutex.TryRLock() {
		t.Fatal("reader acquired write-locked mutex")
	}
	mutex.Unlock()
	locker := mutex.RLocker()
	locker.Lock()
	locker.Unlock()
}

func TestOnce(t *testing.T) {
	var once Once
	calls := 0
	once.Do(func() { calls++ })
	once.Do(func() { calls++ })
	if calls != 1 {
		t.Fatalf("calls = %d", calls)
	}
}

func TestCond(t *testing.T) {
	var mutex Mutex
	condition := NewCond(&mutex)
	ready := false
	finished := make(chan bool)
	go func() {
		mutex.Lock()
		for !ready {
			condition.Wait()
		}
		mutex.Unlock()
		finished <- true
	}()
	mutex.Lock()
	ready = true
	condition.Signal()
	mutex.Unlock()
	<-finished
}
