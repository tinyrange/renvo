//go:build renvo && staragent_minimal

package context

import (
	"errors"
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
var DeadlineExceeded = errors.New("context deadline exceeded")

type emptyCtx int

func (emptyCtx) Deadline() (time.Time, bool) { return time.Time{}, false }
func (emptyCtx) Done() <-chan struct{}       { return nil }
func (emptyCtx) Err() error                  { return nil }
func (emptyCtx) Value(any) any               { return nil }

func Background() Context { return emptyCtx(1) }
func TODO() Context       { return emptyCtx(2) }
