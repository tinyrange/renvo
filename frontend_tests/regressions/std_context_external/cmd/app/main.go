package main

import (
	"context"
	runtime "renvo.dev/x/runtime"
	"renvo.dev/x/runtime/serial"
	"time"
)

type externalContext struct {
	done chan struct{}
	err  error
}

func newExternalContext() *externalContext {
	return &externalContext{done: make(chan struct{})}
}
func (c *externalContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (c *externalContext) Done() <-chan struct{}       { return c.done }
func (c *externalContext) Err() error                  { return c.err }
func (c *externalContext) Value(any) any               { return nil }
func (c *externalContext) cancel() {
	c.err = context.Canceled
	close(c.done)
}

func main() {
	runtime.EnableGoroutines(serial.New())
	if !runtime.StackSupported() {
		print("PASS\n")
		return
	}
	parent := newExternalContext()
	child, cancelChild := context.WithCancel(parent)
	go parent.cancel()
	select {
	case <-child.Done():
	}
	if child.Err() != context.Canceled || context.Cause(child) != context.Canceled {
		print("FAIL\n")
		return
	}
	cancelChild()
	print("PASS\n")
}
