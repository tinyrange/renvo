// Package sync provides basic synchronization primitives for Renvo programs.
//
// The Renvo implementation cooperates with the installed goroutine handler;
// it does not introduce parallel execution. As in Go's sync package, values
// containing these types must not be copied after first use.
package sync

// Locker represents an object that can be locked and unlocked.
type Locker interface {
	Lock()
	Unlock()
}
