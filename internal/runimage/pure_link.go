//go:build !renvo

package runimage

import (
	"fmt"
	"runtime"
	"unsafe"
)

// Dispatcher entries are never admissible as leaf link targets.
func (a *CodeArena) InstallLinkedDispatcher(code []byte, words int) (int, error) {
	if words < 1 || words > 256 {
		return 0, fmt.Errorf("invalid dispatcher state size")
	}
	return a.install(code, uint16(words)|49152)
}

// CallLinked validates the dispatcher and lends a read-only arena view under
// serialization. One cleanup covers both the view and lock, including errors.
// The specialized path avoids the generic leaf/context ABI branch per quantum.
func (a *CodeArena) CallLinked(entry int, state []uint64, context unsafe.Pointer, view *[4]uint64) error {
	a.mu.Lock()
	defer func() {
		if view != nil {
			*view = [4]uint64{}
		}
		a.mu.Unlock()
	}()
	if a.base == 0 || a.broken || context == nil || view == nil || entry < 0 || entry&15 != 0 || entry>>4 >= len(a.entries) || len(state) == 0 {
		return fmt.Errorf("invalid native dispatcher entry")
	}
	words := a.entries[entry>>4].Words
	if words&49152 != 49152 || words&8192 != 0 || words == 65535 || int(words&16383) != len(state) {
		return fmt.Errorf("invalid native dispatcher entry")
	}
	*view = a.linkedView
	top := (uintptr(unsafe.Pointer(&a.stack[len(a.stack)-1])) + 1) &^ 15
	(*LinkedContextABI)(context).AdmissionEpoch = 0
	callContext(a.base+uintptr(entry), uintptr(unsafe.Pointer(&state[0])), uintptr(context), top)
	runtime.KeepAlive(context)
	runtime.KeepAlive(state)
	runtime.KeepAlive(a.stack)
	runtime.KeepAlive(a.entries)
	return nil
}

// AdmitLinkedBlock validates a leaf once under arena serialization. A key proves
// the offset was installed, its family is admissible, and its exact state/body
// shape. Each native selection still checks the requested count and state size.
// It is separate from the public link table: callers cannot forge admissions.
func (a *CodeArena) AdmitLinkedBlock(entry, words, instructions int) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.base == 0 || a.broken || entry < 0 || entry&15 != 0 || entry >= a.used || entry>>4 >= len(a.entries) || words < 1 || words > 256 || instructions < 1 || instructions > 256 {
		return fmt.Errorf("invalid linked block admission")
	}
	record := &a.entries[entry>>4]
	metadata := record.Words
	if metadata == 0 || metadata == 65535 || metadata&16384 != 0 || int(metadata&511) != words || metadata&8192 != 0 && int(record.Instructions) != instructions || metadata&8192 == 0 && instructions > 16 {
		return fmt.Errorf("invalid linked block admission")
	}
	key := (metadata & 40960) | uint16(words)
	if record.Linked != 0 && (record.Linked != key || int(record.AdmittedCount) != instructions) {
		return fmt.Errorf("linked block admission changed")
	}
	record.Linked, record.AdmittedCount = key, uint32(instructions)
	return nil
}

// InstallSharedBlock installs a standalone compatibility entry and a checked
// internal body in the same immutable image. Only native linking uses Body;
// public Call/CallContext can enter only the installed standalone start.
func (a *CodeArena) InstallSharedBlock(code []byte, words int, memory bool, body int) (int, error) {
	if words < 1 || words > 256 || body <= 0 || body >= len(code) {
		return 0, fmt.Errorf("invalid native shared block")
	}
	metadata := uint16(words)
	if memory {
		metadata |= 32768
	}
	return a.installWithBody(code, metadata, body)
}

// LinkedCall is an opaque immutable dispatcher admission, not a raw callable
// address. It remains valid across append-only installations. Every execution
// still serializes the arena and rejects closed/broken owners and wrong shapes.
type LinkedCall struct {
	arena *CodeArena
	entry uintptr
	top   uintptr
	words int
}

func (a *CodeArena) PrepareLinkedCall(entry, words int) (*LinkedCall, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.base == 0 || a.broken || entry < 0 || entry&15 != 0 || entry>>4 >= len(a.entries) || words < 1 || words > 256 {
		return nil, fmt.Errorf("invalid prepared native dispatcher")
	}
	metadata := a.entries[entry>>4].Words
	if metadata&49152 != 49152 || metadata&8192 != 0 || metadata == 65535 || int(metadata&16383) != words {
		return nil, fmt.Errorf("invalid prepared native dispatcher")
	}
	top := (uintptr(unsafe.Pointer(&a.stack[len(a.stack)-1])) + 1) &^ 15
	return &LinkedCall{arena: a, entry: a.base + uintptr(entry), top: top, words: words}, nil
}

func (c *LinkedCall) Call(state []uint64, context unsafe.Pointer, view *[4]uint64) error {
	return c.CallWithTargets(state, context, view, nil)
}

func (c *LinkedCall) CallWithTargets(state []uint64, context unsafe.Pointer, view *[4]uint64, targets *uint64) error {
	if targets != nil {
		*targets = 0
	}
	if c == nil || c.arena == nil {
		if view != nil {
			*view = [4]uint64{}
		}
		return fmt.Errorf("invalid prepared native dispatcher")
	}
	a := c.arena
	a.mu.Lock()
	defer func() {
		if view != nil {
			*view = [4]uint64{}
		}
		if targets != nil {
			*targets = 0
		}
		a.mu.Unlock()
	}()
	if a.base == 0 || a.broken || len(state) != c.words || context == nil || view == nil {
		return fmt.Errorf("invalid prepared native dispatcher call")
	}
	*view = a.linkedView
	if targets != nil {
		*targets = uint64(uintptr(unsafe.Pointer(&a.targets[0])))
	}
	(*LinkedContextABI)(context).AdmissionEpoch = 0
	callContext(c.entry, uintptr(unsafe.Pointer(&state[0])), uintptr(context), c.top)
	runtime.KeepAlive(context)
	runtime.KeepAlive(state)
	runtime.KeepAlive(a.stack)
	runtime.KeepAlive(a.entries)
	return nil
}

// CallBatch lends the same arena view across a bounded number of distinct
// dispatcher calls. The trusted host selector runs between calls and at the
// final boundary; it must not reenter the arena. Native budgets are set and
// checked by the runtime, not enlarged by this serialization helper.
func (c *LinkedCall) CallBatch(state []uint64, context unsafe.Pointer, view *[4]uint64, limit int, next func(int) (bool, error)) (completed int, err error) {
	return c.CallBatchWithTargets(state, context, view, nil, limit, next)
}

func (c *LinkedCall) CallBatchWithTargets(state []uint64, context unsafe.Pointer, view *[4]uint64, targets *uint64, limit int, next func(int) (bool, error)) (completed int, err error) {
	if targets != nil {
		*targets = 0
	}
	if c == nil || c.arena == nil || limit < 1 || limit > 16 || next == nil {
		if view != nil {
			*view = [4]uint64{}
		}
		return 0, fmt.Errorf("invalid prepared dispatcher batch")
	}
	a := c.arena
	a.mu.Lock()
	defer func() {
		if view != nil {
			*view = [4]uint64{}
		}
		if targets != nil {
			*targets = 0
		}
		a.mu.Unlock()
	}()
	if a.base == 0 || a.broken || len(state) != c.words || context == nil || view == nil {
		return 0, fmt.Errorf("invalid prepared dispatcher batch call")
	}
	*view = a.linkedView
	if targets != nil {
		*targets = uint64(uintptr(unsafe.Pointer(&a.targets[0])))
	}
	for {
		more, err := next(completed)
		if err != nil || !more {
			return completed, err
		}
		if completed == limit {
			return completed, fmt.Errorf("prepared dispatcher batch limit exhausted")
		}
		(*LinkedContextABI)(context).AdmissionEpoch = 0
		callContext(c.entry, uintptr(unsafe.Pointer(&state[0])), uintptr(context), c.top)
		runtime.KeepAlive(context)
		runtime.KeepAlive(state)
		runtime.KeepAlive(a.stack)
		runtime.KeepAlive(a.entries)
		completed++
	}
}

// linkedTarget binds a full PC, public arena offset and body length to a
// validated immutable body. BodyFlags contains an arena-relative offset in its
// low word and ABI-family bits in its high word. It never retains a raw entry.
type linkedTarget struct {
	PC, Entry, Instructions, BodyFlags uint64
	CheckedSession                     uint64    // public descriptor checked only in this non-aliasing session
	DescriptorOffset                   uint64    // checked public way, relative to the current descriptor base
	_                                  [2]uint64 // power-of-two stride
}

// PrepareTarget is cold work, separate from execution. Replacing an indexed
// proof cannot race a borrowed native view because both hold the arena lock.
func (c *LinkedCall) PrepareTarget(pc uint64, entry, instructions int) error {
	if c == nil || c.arena == nil || pc&3 != 0 || instructions < 1 || instructions > 256 {
		return fmt.Errorf("invalid prepared target")
	}
	a := c.arena
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.base == 0 || a.broken || entry < 0 || entry&15 != 0 || entry >= a.used || entry>>4 >= len(a.entries) {
		return fmt.Errorf("invalid prepared target owner or entry")
	}
	r := a.entries[entry>>4]
	key := uint16(c.words)
	if r.Linked == 0 || r.Linked&511 != key || int(r.AdmittedCount) != instructions || int(r.Words&511) != c.words || r.Words&16384 != 0 || r.Words == 65535 {
		return fmt.Errorf("invalid prepared target admission")
	}
	if r.Words&8192 != 0 && int(r.Instructions) != instructions {
		return fmt.Errorf("invalid prepared loop target")
	}
	body := uint64(entry) + uint64(r.Body)
	if body >= uint64(a.used) || body >= uint64(a.size) {
		return fmt.Errorf("invalid prepared target body")
	}
	flags := uint64(r.Words & 40960)
	if a.transfers[entry] {
		flags |= 1
	}
	base, slot := ((pc>>2)^(pc>>12))&255, uint64(0)
	found := false
	for way := uint64(0); way < 4; way++ {
		i := base + way*256
		if a.targets[i].PC == pc && a.targets[i].Instructions != 0 {
			slot, found = i, true
			break
		}
	}
	if !found {
		for way := uint64(0); way < 4; way++ {
			i := base + way*256
			if a.targets[i].Instructions == 0 {
				slot, found = i, true
				break
			}
		}
	}
	if !found {
		slot = base + (a.targetVictim&3)*256
		a.targetVictim++
	}
	a.targets[slot] = linkedTarget{PC: pc, Entry: uint64(entry), Instructions: uint64(instructions), BodyFlags: body | flags<<32}
	return nil
}
