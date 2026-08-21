//go:build renvo && !staragent_minimal

package context

import (
	"errors"
	runtime "renvo.dev/x/runtime"
	"time"
)

type Context interface {
	Deadline() (time.Time, bool)
	Done() <-chan struct{}
	Err() error
	Value(key any) any
}
type CancelFunc func()
type CancelCauseFunc func(error)

var Canceled = errors.New("context canceled")
var DeadlineExceeded error = deadlineExceededError{}

type deadlineExceededError struct{}

func (deadlineExceededError) Error() string   { return "context deadline exceeded" }
func (deadlineExceededError) Timeout() bool   { return true }
func (deadlineExceededError) Temporary() bool { return true }

type emptyCtx int

func (emptyCtx) Deadline() (time.Time, bool) { return time.Time{}, false }
func (emptyCtx) Done() <-chan struct{}       { return nil }
func (emptyCtx) Err() error                  { return nil }
func (emptyCtx) Value(any) any               { return nil }
func Background() Context                    { return emptyCtx(1) }
func TODO() Context                          { return emptyCtx(2) }

func makeDoneChannel() chan struct{} {
	done := make(chan struct{})
	return done
}

type cancelCtx struct {
	parent   Context
	done     chan struct{}
	children []*cancelCtx
	err      error
	cause    error
	deadline time.Time
	hasLimit bool
	timer    runtime.Timer
}

func WithCancel(parent Context) (Context, CancelFunc) {
	c := newCancel(parent)
	return c, func() { c.cancel(Canceled, Canceled) }
}
func WithCancelCause(parent Context) (Context, CancelCauseFunc) {
	c := newCancel(parent)
	return c, func(cause error) {
		if cause == nil {
			cause = Canceled
		}
		c.cancel(Canceled, cause)
	}
}
func newCancel(parent Context) *cancelCtx {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	c := new(cancelCtx)
	c.parent = parent
	c.done = makeDoneChannel()
	if p := findCancel(parent); p != nil {
		if p.err != nil {
			c.cancel(p.err, p.cause)
		} else {
			p.children = append(p.children, c)
		}
	} else if err := parent.Err(); err != nil {
		c.cancel(err, Cause(parent))
	} else if parent.Done() != nil {
		go waitParent(parent, c)
	}
	return c
}
func (c *cancelCtx) Deadline() (time.Time, bool) {
	if c.hasLimit {
		return c.deadline, true
	}
	return c.parent.Deadline()
}
func (c *cancelCtx) Done() <-chan struct{} { return c.done }
func (c *cancelCtx) Err() error            { return c.err }
func (c *cancelCtx) Value(key any) any     { return c.parent.Value(key) }
func (c *cancelCtx) cancel(err, cause error) {
	if c.err != nil {
		return
	}
	c.err = err
	c.cause = cause
	timer := c.timer
	c.timer = 0
	if timer != 0 {
		timer.Stop()
	}
	done := c.done
	close(done)
	children := c.children
	c.children = nil
	for _, child := range children {
		child.cancel(err, cause)
	}
	removeChild(c.parent, c)
}
func waitParent(parent Context, child *cancelCtx) {
	select {
	case <-parent.Done():
		err := parent.Err()
		if err == nil {
			err = Canceled
		}
		cause := Cause(parent)
		if cause == nil {
			cause = err
		}
		child.cancel(err, cause)
	case <-child.Done():
	}
}

func WithDeadline(parent Context, deadline time.Time) (Context, CancelFunc) {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	if inherited, ok := parent.Deadline(); ok && inherited.Sub(deadline) <= 0 {
		return WithCancel(parent)
	}
	c := newCancel(parent)
	c.deadline = deadline
	c.hasLimit = true
	if c.err == nil {
		delay := deadline.Sub(time.Now())
		if delay <= 0 {
			c.cancel(DeadlineExceeded, DeadlineExceeded)
		} else {
			c.timer = runtime.NewTimer(int64(delay))
			go waitDeadline(c)
		}
	}
	return c, func() { c.cancel(Canceled, Canceled) }
}

func WithTimeout(parent Context, timeout time.Duration) (Context, CancelFunc) {
	return WithDeadline(parent, time.Now().Add(timeout))
}

func waitDeadline(c *cancelCtx) {
	timer := c.timer
	if timer != 0 && timer.Wait() {
		c.cancel(DeadlineExceeded, DeadlineExceeded)
	}
}
func removeChild(parent Context, child *cancelCtx) {
	p := findCancel(parent)
	if p == nil {
		return
	}
	for i, candidate := range p.children {
		if candidate == child {
			copy(p.children[i:], p.children[i+1:])
			p.children = p.children[:len(p.children)-1]
			return
		}
	}
}
func findCancel(ctx Context) *cancelCtx {
	for {
		switch c := ctx.(type) {
		case *cancelCtx:
			return c
		case *valueCtx:
			ctx = c.parent
		default:
			return nil
		}
	}
}
func Cause(ctx Context) error {
	if ctx == nil {
		panic("cannot find cause of nil Context")
	}
	if c := findCancel(ctx); c != nil {
		return c.cause
	}
	return ctx.Err()
}

type valueCtx struct {
	parent Context
	key    any
	value  any
}

func WithValue(parent Context, key, value any) Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	if key == nil {
		panic("nil key")
	}
	return &valueCtx{parent: parent, key: key, value: value}
}
func (c *valueCtx) Deadline() (time.Time, bool) { return c.parent.Deadline() }
func (c *valueCtx) Done() <-chan struct{}       { return c.parent.Done() }
func (c *valueCtx) Err() error                  { return c.parent.Err() }
func (c *valueCtx) Value(key any) any {
	if c.key == key {
		return c.value
	}
	return c.parent.Value(key)
}
