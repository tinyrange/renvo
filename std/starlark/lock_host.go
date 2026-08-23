//go:build !renvo

package starlark

import "sync"

type threadLock struct{ lock sync.RWMutex }

func (m *threadLock) Lock()    { m.lock.Lock() }
func (m *threadLock) Unlock()  { m.lock.Unlock() }
func (m *threadLock) RLock()   { m.lock.RLock() }
func (m *threadLock) RUnlock() { m.lock.RUnlock() }
