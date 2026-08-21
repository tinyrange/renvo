package main

import (
	"context"
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
	deadline := time.Now().Add(100 * time.Millisecond)
	parent, cancelParent := context.WithDeadline(context.Background(), deadline)
	child, cancelChild := context.WithDeadline(parent, deadline.Add(time.Second))
	inherited, ok := child.Deadline()
	if !ok || !inherited.Equal(deadline) {
		print("FAIL\n")
		return
	}
	cancelChild()
	cancelParent()
	manual, cancelManual := context.WithTimeout(context.Background(), time.Hour)
	cancelManual()
	if manual.Err() != context.Canceled {
		print("FAIL\n")
		return
	}
	handler.Drain()
	print("PASS\n")
}
