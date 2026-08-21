package main

import (
	"context"
	runtime "renvo.dev/x/runtime"
	"renvo.dev/x/runtime/serial"
)

var sharedCancel context.CancelFunc

func cancelShared() { sharedCancel() }

func main() {
	runtime.EnableGoroutines(serial.New())
	if !runtime.StackSupported() {
		print("PASS\n")
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	sharedCancel = cancel
	go cancelShared()
	select {
	case <-ctx.Done():
	}
	cancel()
	if ctx.Err() != context.Canceled {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}
