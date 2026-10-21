//go:build !renvo

package runimage

import (
	"fmt"
	"unsafe"
)

// EnsureStack is quiescent owner work, never a guest-controlled pointer. The
// compiler reports the complete frame requirement including dispatcher depth.
func (a *CodeArena) EnsureStack(bytes int) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.base == 0 || bytes < 0 {
		return fmt.Errorf("invalid native stack reservation")
	}
	if bytes > len(a.stack) {
		a.releaseOwnedPinsLocked()
		a.stack = make([]byte, bytes)
		top := (uintptr(unsafe.Pointer(&a.stack[len(a.stack)-1])) + 1) &^ 15
		for _, call := range a.prepared {
			call.top = top
		}
	}
	return nil
}
