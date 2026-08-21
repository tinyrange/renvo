//go:build !renvo

package context

import (
	stdcontext "context"
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

var Canceled = stdcontext.Canceled
var DeadlineExceeded = stdcontext.DeadlineExceeded

func Background() Context { return stdcontext.Background() }
func TODO() Context       { return stdcontext.TODO() }
func WithCancel(parent Context) (Context, CancelFunc) {
	ctx, cancel := stdcontext.WithCancel(parent)
	return ctx, CancelFunc(cancel)
}
func WithCancelCause(parent Context) (Context, CancelCauseFunc) {
	ctx, cancel := stdcontext.WithCancelCause(parent)
	return ctx, CancelCauseFunc(cancel)
}
func WithDeadline(parent Context, deadline time.Time) (Context, CancelFunc) {
	ctx, cancel := stdcontext.WithDeadline(parent, deadline)
	return ctx, CancelFunc(cancel)
}
func WithTimeout(parent Context, timeout time.Duration) (Context, CancelFunc) {
	ctx, cancel := stdcontext.WithTimeout(parent, timeout)
	return ctx, CancelFunc(cancel)
}
func WithValue(parent Context, key, value any) Context {
	return stdcontext.WithValue(parent, key, value)
}
func Cause(ctx Context) error { return stdcontext.Cause(ctx) }
