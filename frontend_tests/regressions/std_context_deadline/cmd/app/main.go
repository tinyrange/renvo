package main

import (
	"context"
	runtime "renvo.dev/x/runtime"
	"renvo.dev/x/runtime/serial"
	"time"
)

func main() {
	runtime.EnableGoroutines(serial.New())
	if !runtime.StackSupported() {
		print("PASS\n")
		return
	}
	timed, cancelTimed := context.WithTimeout(context.Background(), 2*time.Millisecond)
	select {
	case <-timed.Done():
	}
	if timed.Err() != context.DeadlineExceeded || context.Cause(timed) != context.DeadlineExceeded {
		print("FAIL\n")
		return
	}
	cancelTimed()
	print("PASS\n")
}
