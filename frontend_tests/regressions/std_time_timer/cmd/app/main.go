package main

import (
	runtime "renvo.dev/x/runtime"
	"renvo.dev/x/runtime/serial"
	"time"
)

func main() {
	handler := serial.New()
	runtime.EnableGoroutines(handler)
	if !runtime.StackSupported() {
		print("PASS\n")
		return
	}
	start := time.Now()
	time.Sleep(2 * time.Millisecond)
	if time.Since(start) < time.Millisecond {
		print("FAIL\n")
		return
	}
	when := <-time.After(2 * time.Millisecond)
	if when.Sub(start) < 2*time.Millisecond {
		print("FAIL\n")
		return
	}
	timer := time.NewTimer(time.Hour)
	if !timer.Stop() {
		print("FAIL\n")
		return
	}
	select {
	case <-timer.C:
		print("FAIL\n")
		return
	default:
	}
	handler.Drain()
	print("PASS\n")
}
