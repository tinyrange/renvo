package main

import (
	"sync"

	runtime "renvo.dev/x/runtime"
	"renvo.dev/x/runtime/serial"
)

func main() {
	handler := serial.New()
	runtime.EnableGoroutines(handler)
	if !runtime.StackSupported() {
		print("PASS\n")
		return
	}

	var mutex sync.Mutex
	held := make(chan int)
	release := make(chan int)
	done := make(chan int)
	shared := 0
	go func() {
		mutex.Lock()
		shared = 1
		held <- 1
		<-release
		shared = 2
		mutex.Unlock()
		done <- 1
	}()
	<-held
	go func() {
		mutex.Lock()
		shared = shared + 1
		mutex.Unlock()
		done <- 1
	}()
	release <- 1
	<-done
	<-done
	if shared != 3 || !mutex.TryLock() || mutex.TryLock() {
		print("FAIL mutex\n")
		return
	}
	mutex.Unlock()

	var readWrite sync.RWMutex
	writerHeld := make(chan int)
	writerRelease := make(chan int)
	readerDone := make(chan int)
	observed := 0
	go func() {
		readWrite.Lock()
		shared = 7
		writerHeld <- 1
		<-writerRelease
		readWrite.Unlock()
	}()
	<-writerHeld
	go func() {
		readWrite.RLock()
		observed = shared
		readWrite.RUnlock()
		readerDone <- 1
	}()
	writerRelease <- 1
	<-readerDone
	if observed != 7 || !readWrite.TryRLock() || !readWrite.TryRLock() || readWrite.TryLock() {
		print("FAIL rwmutex\n")
		return
	}
	readWrite.RUnlock()
	readWrite.RUnlock()

	var group sync.WaitGroup
	var sumMutex sync.Mutex
	sum := 0
	for i := 1; i <= 6; i++ {
		value := i
		group.Go(func() {
			sumMutex.Lock()
			sum += value
			sumMutex.Unlock()
		})
	}
	group.Wait()
	if sum != 21 {
		print("FAIL waitgroup\n")
		return
	}

	var once sync.Once
	onceCalls := 0
	for i := 0; i < 5; i++ {
		group.Go(func() {
			once.Do(func() { onceCalls++ })
		})
	}
	group.Wait()
	if onceCalls != 1 {
		print("FAIL once\n")
		return
	}

	var conditionMutex sync.Mutex
	condition := sync.NewCond(&conditionMutex)
	waiting := make(chan int)
	conditionReady := false
	conditionObserved := false
	group.Go(func() {
		conditionMutex.Lock()
		waiting <- 1
		for !conditionReady {
			condition.Wait()
		}
		conditionObserved = true
		conditionMutex.Unlock()
	})
	<-waiting
	conditionMutex.Lock()
	conditionReady = true
	condition.Signal()
	conditionMutex.Unlock()
	group.Wait()
	if !conditionObserved {
		print("FAIL cond\n")
		return
	}

	var broadcastMutex sync.Mutex
	broadcast := sync.NewCond(&broadcastMutex)
	arrived := make(chan int)
	broadcastReady := false
	broadcastCount := 0
	for i := 0; i < 3; i++ {
		group.Go(func() {
			broadcastMutex.Lock()
			arrived <- 1
			for !broadcastReady {
				broadcast.Wait()
			}
			broadcastCount++
			broadcastMutex.Unlock()
		})
	}
	<-arrived
	<-arrived
	<-arrived
	broadcastMutex.Lock()
	broadcastReady = true
	broadcast.Broadcast()
	broadcastMutex.Unlock()
	group.Wait()
	if broadcastCount != 3 {
		print("FAIL broadcast\n")
		return
	}

	print("PASS\n")
}
