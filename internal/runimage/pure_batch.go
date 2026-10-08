//go:build !renvo

package runimage

import (
	"fmt"
	"unsafe"
)

// CallBatch serializes a bounded sequence of pure block calls with one lock.
// The trusted selector observes the number of successfully completed calls,
// including a final notification at the call limit. It must return promptly and
// must not recursively call any method on this arena. No execution capability
// escapes the lock; each selected offset and state size is checked as in Call.
// Installation and Close wait until the whole batch (including its selector)
// has returned. The caller separately bounds any work performed by the selector.
func (a *CodeArena) CallBatch(state []uint64, limit int, next func(completed int) (entry int, more bool, err error)) (completed int, err error) {
	return a.callBatch(state, nil, limit, next)
}

// CallBatchContext supplies one trusted, GC-rooted host context to each call;
// all entry, state-shape and serialization checks are identical to CallBatch.
func (a *CodeArena) CallBatchContext(state []uint64, context unsafe.Pointer, limit int, next func(int) (int, bool, error)) (int, error) {
	if context == nil {
		return 0, fmt.Errorf("missing native memory context")
	}
	return a.callBatch(state, context, limit, next)
}
func (a *CodeArena) callBatch(state []uint64, context unsafe.Pointer, limit int, next func(int) (int, bool, error)) (completed int, err error) {
	if limit < 1 || limit > 256 || next == nil {
		return 0, fmt.Errorf("invalid native call batch")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.base == 0 || a.broken || len(state) == 0 {
		return 0, fmt.Errorf("invalid native call batch")
	}
	for {
		entry, more, err := next(completed)
		if err != nil || !more {
			return completed, err
		}
		if completed == limit {
			return completed, fmt.Errorf("native call batch limit exhausted")
		}
		if err := a.callLocked(entry, state, context); err != nil {
			return completed, err
		}
		completed++
	}
}
