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
	parent, cancelParent := context.WithCancel(context.Background())
	child, cancelChild := context.WithCancel(parent)
	go cancelParent()
	select {
	case <-child.Done():
	}
	if child.Err() != context.Canceled || parent.Err() != context.Canceled {
		print("FAIL\n")
		return
	}
	cancelChild()
	print("PASS\n")
}
