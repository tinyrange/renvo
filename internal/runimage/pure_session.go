//go:build !renvo

package runimage

import (
	"fmt"
	"renvo.dev/internal/rfenativebridge"
	"runtime"
	"unsafe"
)

// SessionInstructionLimit bounds uninterrupted guest work, not the total Run
// ceiling. Every public session returns at/before this bound; the caller can
// observe cancellation or perform host work between sessions. Larger budgets
// never use the old goroutine-stack assembly boundary.
const SessionInstructionLimit = 65536

const (
	sessionImageWords      = unsafe.Sizeof(LinkedContextABI{}) / 8
	sessionPrefixWords     = unsafe.Offsetof(LinkedContextABI{}.Blocks) / 8
	sessionCodeViewWord    = unsafe.Offsetof(LinkedContextABI{}.CodeView) / 8
	sessionEpochWord       = unsafe.Offsetof(LinkedContextABI{}.AdmissionEpoch) / 8
	sessionExitsWord       = unsafe.Offsetof(LinkedContextABI{}.LoopExits) / 8
	sessionIterationsWord  = unsafe.Offsetof(LinkedContextABI{}.LoopIterations) / 8
	sessionTargetsWord     = unsafe.Offsetof(LinkedContextABI{}.PreparedTargets) / 8
	sessionDescriptorsWord = unsafe.Offsetof(LinkedContextABI{}.DescriptorBase) / 8
	sessionRemainingWord   = unsafe.Offsetof(LinkedContextABI{}.Remaining) / 8
	sessionTotalWord       = unsafe.Offsetof(LinkedContextABI{}.Total) / 8
	sessionMemoryWord      = unsafe.Offsetof(LinkedContextABI{}.MemoryTotal) / 8
	sessionDirectWord      = unsafe.Offsetof(LinkedContextABI{}.DirectGuest) / 8
)

func ForeignSessionsAvailable() bool { return rfenativebridge.Available }

// CallSession executes one exact-budget slice through a cgo boundary. The
// pointer-free staging allocations exclude enclosing Go owner graphs. All memory
// reachable by native code is explicitly rooted and pinned for the duration.
// Native effects that could change the context's pointer graph mid-call use
// the conservative original <=64-instruction boundary instead.
func (c *LinkedCall) CallSession(state []uint64, m *LinkedContextABI, budget uint64) (err error) {
	clearContextBorrow(m)
	if c == nil || c.arena == nil || m == nil || len(state) != c.words || budget == 0 || budget > SessionInstructionLimit {
		return fmt.Errorf("invalid native execution session")
	}
	a := c.arena
	a.mu.Lock()
	defer func() {
		clearContextBorrow(m)
		a.mu.Unlock()
		runtime.KeepAlive(m)
		runtime.KeepAlive(state)
	}()
	if a.base == 0 || a.broken {
		return fmt.Errorf("invalid native execution owner")
	}
	m.Remaining, m.Total, m.MemoryTotal, m.Status, m.Retired = budget, 0, 0, 0, 0
	m.LoopExits, m.LoopIterations = 0, 0
	actual := budget
	if !rfenativebridge.Available || sessionNeedsCompatibility(state, m) {
		if actual > 64 {
			actual = 64
		}
		m.Remaining = actual
		a.borrowLinked(&m.CodeView, &m.PreparedTargets)
		callContext(c.entry, uintptr(unsafe.Pointer(&state[0])), uintptr(unsafe.Pointer(m)), c.top)
	} else {
		var pins runtime.Pinner
		defer pins.Unpin()
		if a.sessionImage == nil {
			a.sessionImage, a.sessionState = new([sessionImageWords]uint64), new([256]uint64)
		}
		// The public descriptors cannot change through guest RAM on this path.
		// Each invocation has a unique token, including across different contexts.
		a.sessionEpoch++
		if a.sessionEpoch == 0 {
			for i := range a.targets {
				a.targets[i].CheckedSession = 0
			}
			a.sessionEpoch = 1
		}
		image := a.sessionImage
		image[sessionEpochWord] = a.sessionEpoch
		nativeState := a.sessionState[:len(state)]
		copy(nativeState, state)
		// Only the small memory/budget prefix is copied. Descriptors remain
		// observable in their authoritative public table on every selection.
		copy(image[:sessionPrefixWords], unsafe.Slice((*uint64)(unsafe.Pointer(m)), sessionPrefixWords))
		image[sessionExitsWord], image[sessionIterationsWord] = 0, 0
		image[sessionDirectWord], image[sessionDirectWord+1], image[sessionDirectWord+2] = m.DirectGuest, m.DirectHost, m.DirectSize
		defer func() {
			clear(image[:sessionPrefixWords])
			clear(image[sessionDirectWord : sessionDirectWord+3])
			image[sessionTargetsWord], image[sessionDescriptorsWord], image[sessionEpochWord] = 0, 0, 0
		}() // integer addresses must not outlive their pins
		pins.Pin(m) // native reads only its pointer-free Blocks field
		pins.Pin(image)
		pins.Pin(a.sessionState)
		pins.Pin(&a.stack[0])
		pins.Pin(&a.entries[0])
		pins.Pin(&a.targets[0])
		if m.Clock != nil {
			pins.Pin(m.Clock)
		}
		for _, page := range m.Pages {
			if page.Data != nil {
				pins.Pin(page.Data)
			}
			if page.Epoch != nil {
				pins.Pin(page.Epoch)
			}
		}
		copy(image[sessionCodeViewWord:sessionPrefixWords], a.linkedView[:])
		image[sessionTargetsWord] = uint64(uintptr(unsafe.Pointer(&a.targets[0])))
		image[sessionDescriptorsWord] = uint64(uintptr(unsafe.Pointer(&m.Blocks[0])))
		if len(a.faults) != 0 {
			pins.Pin(&a.faults[0])
		}
		rfenativebridge.CallFaults(c.entry, unsafe.Pointer(&nativeState[0]), unsafe.Pointer(image), c.top, a.faults)
		copy(state, nativeState)
		m.Retired, m.Status, m.Address = image[0], image[1], image[2]
		m.Remaining, m.Total, m.MemoryTotal = image[sessionRemainingWord], image[sessionTotalWord], image[sessionMemoryWord]
		m.LoopExits, m.LoopIterations = image[sessionExitsWord], image[sessionIterationsWord]
	}
	if (m.Status > 2 && m.Status != 4) || m.Total > actual || m.Remaining != actual-m.Total || m.MemoryTotal > m.Total {
		return fmt.Errorf("invalid native execution session progress")
	}
	m.Remaining = budget - m.Total
	return nil
}

func sessionOverlap(ptr unsafe.Pointer, bytes uintptr, base uintptr, length uintptr) bool {
	p := uintptr(ptr)
	return p != 0 && (p >= base && p-base < length || p < base && base-p < bytes)
}

func sessionNeedsCompatibility(state []uint64, m *LinkedContextABI) bool {
	// The verified runtime layout appends its private owner pointer. Exclude
	// writes through guest RAM to that owner as well as the public pointer graph.
	base, size := uintptr(unsafe.Pointer(m)), unsafe.Sizeof(*m)+unsafe.Sizeof(uintptr(0))
	if sessionOverlap(unsafe.Pointer(&state[0]), uintptr(len(state))*8, base, size) || sessionOverlap(unsafe.Pointer(m.Clock), 8, base, size) {
		return true
	}
	stateBase, stateSize := uintptr(unsafe.Pointer(&state[0])), uintptr(len(state))*8
	if sessionOverlap(unsafe.Pointer(m.Clock), 8, stateBase, stateSize) {
		return true
	}
	for _, page := range m.Pages {
		if sessionOverlap(unsafe.Pointer(page.Data), 4096, stateBase, stateSize) || sessionOverlap(unsafe.Pointer(page.Epoch), 8, stateBase, stateSize) {
			return true
		}
		if sessionOverlap(unsafe.Pointer(page.Data), 4096, base, size) || sessionOverlap(unsafe.Pointer(page.Epoch), 8, base, size) {
			return true
		}
	}
	return false
}
