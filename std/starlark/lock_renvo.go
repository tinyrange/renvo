//go:build renvo

package starlark

// Renvo's current runtime serializes goroutines, so Thread does not need a
// blocking lock. Keeping the same lock surface lets the interpreter source be
// shared with host Go while avoiding channel-backed synchronization on-device.
type threadLock struct{}

func (*threadLock) Lock()    {}
func (*threadLock) Unlock()  {}
func (*threadLock) RLock()   {}
func (*threadLock) RUnlock() {}
