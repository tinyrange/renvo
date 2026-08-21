package main

import (
	"context"
	runtime "renvo.dev/x/runtime"
	"renvo.dev/x/runtime/serial"
)

func main() {
	runtime.EnableGoroutines(serial.New())
	if !runtime.StackSupported() {
		print("PASS\n")
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if ctx.Err() != context.Canceled {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}
