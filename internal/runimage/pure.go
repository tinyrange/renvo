//go:build !renvo

package runimage

import (
	"fmt"
	"runtime"
	"sync"
	"unsafe"
)

// CodeArena owns bounded native code storage for short, call-free state
// transformations. Installation, calls, and Close are serialized. Entries are
// offsets, so callers cannot accidentally invoke an arbitrary host address.
type CodeArena struct {
	mu         sync.Mutex
	base       uintptr
	size, used int
	entries    map[int]bool
	stack      []byte
	broken     bool
}

func NewCodeArena(size int) (*CodeArena, error) {
	if size < 4096 || size > 64<<20 {
		return nil, fmt.Errorf("invalid code arena size")
	}
	size = (size + 16383) &^ 16383
	base, err := pureMap(size)
	if err != nil {
		return nil, err
	}
	return &CodeArena{base: base, size: size, entries: map[int]bool{}, stack: make([]byte, 64<<10)}, nil
}
func (a *CodeArena) Install(code []byte) (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	at := (a.used + 15) &^ 15
	if a.base == 0 || a.broken || len(code) == 0 || len(code) > a.size-at {
		return 0, fmt.Errorf("native code arena full or closed")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := pureWritable(a.base, a.size, true); err != nil {
		return 0, err
	}
	copy(unsafe.Slice((*byte)(unsafe.Pointer(a.base+uintptr(at))), len(code)), code)
	if err := pureSeal(a.base, a.size, at, len(code)); err != nil {
		a.broken = true
		return 0, err
	}
	a.used = at + len(code)
	a.entries[at] = true
	return at, nil
}
func (a *CodeArena) Call(entry int, state []uint64) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.base == 0 || a.broken || !a.entries[entry] || len(state) == 0 {
		return fmt.Errorf("invalid native block entry")
	}
	top := (uintptr(unsafe.Pointer(&a.stack[len(a.stack)-1])) + 1) &^ 15
	callPure(a.base+uintptr(entry), uintptr(unsafe.Pointer(&state[0])), top)
	runtime.KeepAlive(state)
	runtime.KeepAlive(a.stack)
	return nil
}
func (a *CodeArena) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.base == 0 {
		return nil
	}
	err := pureUnmap(a.base, a.size)
	if err == nil {
		a.base = 0
		a.entries = nil
		a.stack = nil
	}
	return err
}
