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
	sessionTimerWord       = unsafe.Offsetof(LinkedContextABI{}.TimerFunction) / 8
)

func ForeignSessionsAvailable() bool { return rfenativebridge.Available }

// CallSession executes one exact-budget slice through a cgo boundary. The
// pointer-free staging allocations exclude enclosing Go owner graphs. All memory
// reachable by native code is explicitly rooted and pinned for the duration.
// Native effects that could change the context's pointer graph mid-call use
// the conservative original <=64-instruction boundary instead.
func (c *LinkedCall) CallSession(state []uint64, m *LinkedContextABI, budget uint64) error {
	return c.callSession(state, m, budget, false, nil, 0, nil)
}

// CallOwnedSession permits pin reuse only for the host-owned address-space
// contract. Every call still verifies non-aliasing, compares every live pointer,
// issues a fresh admission epoch, and enforces the same scheduling slice.
func (c *LinkedCall) CallOwnedSession(state []uint64, m *LinkedContextABI, budget uint64) error {
	return c.callSession(state, m, budget, true, nil, 0, nil)
}

// CallOwnedBackingSession pins one contiguous Go byte allocation instead of
// repeatedly pinning its interior pages. The entire live page graph is checked
// against this allocation on every call; no permissions or aliases are waived.
func (c *LinkedCall) CallOwnedBackingSession(state []uint64, m *LinkedContextABI, budget uint64, backing []byte) error {
	return c.callSession(state, m, budget, true, backing, 0, nil)
}

// CallOwnedBackingSessionUntil keeps the same admitted graph resident across
// <=64-instruction timer polls. Only the host-owned, non-aliasing foreign path
// can batch. Compatibility calls retain the original scheduling boundary.
func (c *LinkedCall) CallOwnedBackingSessionUntil(state []uint64, m *LinkedContextABI, budget uint64, backing []byte, deadline uint64) (bool, error) {
	expired := false
	err := c.callSession(state, m, budget, true, backing, deadline, &expired)
	return expired, err
}
func (c *LinkedCall) callSession(state []uint64, m *LinkedContextABI, budget uint64, owned bool, backing []byte, deadline uint64, expired *bool) (err error) {
	clearContextBorrow(m)
	if c == nil || c.arena == nil || m == nil || len(state) != c.words || budget == 0 || budget > SessionInstructionLimit {
		c.ReleaseOwnedSessionPins()
		return fmt.Errorf("invalid native execution session")
	}
	if len(backing) != 0 && !sessionBackingContainsPages(m, backing) {
		c.ReleaseOwnedSessionPins()
		return fmt.Errorf("native page outside owned backing")
	}
	a := c.arena
	a.mu.Lock()
	defer func() {
		clearContextBorrow(m)
		a.mu.Unlock()
		runtime.KeepAlive(m)
		runtime.KeepAlive(state)
	}()
	if !owned {
		a.releaseOwnedPinsLocked()
	}
	if a.base == 0 || a.broken {
		a.releaseOwnedPinsLocked()
		return fmt.Errorf("invalid native execution owner")
	}
	m.Remaining, m.Total, m.MemoryTotal, m.Status, m.Retired = budget, 0, 0, 0, 0
	m.LoopExits, m.LoopIterations = 0, 0
	actual := budget
	compatible := false
	if len(backing) != 0 {
		compatible = sessionNeedsBackingCompatibility(state, m, backing)
	} else {
		compatible = sessionNeedsCompatibility(state, m)
	}
	if !rfenativebridge.Available || compatible {
		a.releaseOwnedPinsLocked()
		if actual > 64 {
			actual = 64
		}
		m.Remaining = actual
		a.borrowLinked(&m.CodeView, &m.PreparedTargets)
		callContext(c.entry, uintptr(unsafe.Pointer(&state[0])), uintptr(unsafe.Pointer(m)), c.top)
	} else {
		var pins *runtime.Pinner
		if !owned {
			pins = new(runtime.Pinner)
			defer pins.Unpin()
		}
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
		clear(image[sessionTimerWord : sessionTimerWord+3])
		if deadline != 0 && owned && len(backing) != 0 {
			image[sessionTimerWord], image[sessionTimerWord+1] = rfenativebridge.TimerFunction(), deadline
		}
		defer func() {
			clear(image[:sessionPrefixWords])
			clear(image[sessionDirectWord : sessionDirectWord+3])
			clear(image[sessionTimerWord : sessionTimerWord+3])
			image[sessionTargetsWord], image[sessionDescriptorsWord], image[sessionEpochWord] = 0, 0, 0
		}() // integer addresses must not outlive their pins
		if owned {
			pins = &a.pinOwnedSession(m, backing).pinner
		} else {
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
		}
		copy(image[sessionCodeViewWord:sessionPrefixWords], a.linkedView[:])
		image[sessionTargetsWord] = uint64(uintptr(unsafe.Pointer(&a.targets[0])))
		image[sessionDescriptorsWord] = uint64(uintptr(unsafe.Pointer(&m.Blocks[0])))
		if !owned && len(a.faults) != 0 {
			pins.Pin(&a.faults[0])
		}
		rfenativebridge.CallFaults(c.entry, unsafe.Pointer(&nativeState[0]), unsafe.Pointer(image), c.top, a.faults)
		if expired != nil {
			*expired = image[sessionTimerWord+2] != 0
		}
		copy(state, nativeState)
		m.Retired, m.Status, m.Address = image[0], image[1], image[2]
		m.Remaining, m.Total, m.MemoryTotal = image[sessionRemainingWord], image[sessionTotalWord], image[sessionMemoryWord]
		m.LoopExits, m.LoopIterations = image[sessionExitsWord], image[sessionIterationsWord]
	}
	if owned && (m.Status != 0 || m.Total == 0) {
		if a.sessionImage != nil {
			clearSessionPinBorrow(a.sessionImage)
		}
		a.releaseOwnedPinsLocked() // retain only after nonzero, fault-free progress; link misses do not change pointer authority
	}
	if (m.Status > 2 && m.Status != 4) || m.Total > actual || m.Remaining != actual-m.Total || m.MemoryTotal > m.Total {
		if a.sessionImage != nil {
			clearSessionPinBorrow(a.sessionImage)
		}
		a.releaseOwnedPinsLocked()
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

// Pointers, not page contents/permissions/tags, determine GC pin identity.
// Numeric metadata is still copied and validated independently on every call.
type ownedPinPage struct{ data, epoch unsafe.Pointer }
type ownedSessionPins struct {
	pinner  runtime.Pinner
	context *LinkedContextABI
	clock   *uint64
	entries unsafe.Pointer
	faults  unsafe.Pointer
	backing unsafe.Pointer
	pages   [64]ownedPinPage
}

func (a *CodeArena) releaseOwnedPinsLocked() {
	if a.ownedPins != nil {
		released := a.ownedPins
		released.pinner.Unpin()
		// Keep only reusable bookkeeping, never roots from the retired graph.
		released.context, released.clock = nil, nil
		released.entries, released.faults, released.backing = nil, nil, nil
		clear(released.pages[:])
		a.ownedPins = nil
		a.sparePins = released
	}
}

func (c *LinkedCall) ReleaseOwnedSessionPins() {
	if c == nil || c.arena == nil {
		return
	}
	a := c.arena
	a.mu.Lock()
	a.releaseOwnedPinsLocked()
	a.mu.Unlock()
}

func (a *CodeArena) pinOwnedSession(m *LinkedContextABI, backing []byte) *ownedSessionPins {
	backingBase := unsafe.Pointer(nil)
	if len(backing) != 0 {
		backingBase = unsafe.Pointer(&backing[0])
	}
	entries := unsafe.Pointer(&a.entries[0])
	faults := unsafe.Pointer(nil)
	if len(a.faults) != 0 {
		faults = unsafe.Pointer(&a.faults[0])
	}
	cached := a.ownedPins
	match := cached != nil && cached.context == m && cached.clock == m.Clock && cached.entries == entries && cached.faults == faults && cached.backing == backingBase
	if match {
		for i, page := range m.Pages {
			if cached.pages[i].data != unsafe.Pointer(page.Data) || cached.pages[i].epoch != unsafe.Pointer(page.Epoch) {
				match = false
				break
			}
		}
	}
	if match {
		return cached
	}
	// A live graph replacement gets fresh bookkeeping as before. Reuse only
	// storage already released on an earlier fault/zero-progress/explicit exit.
	// This avoids allocation on repeated refill faults without retaining pins
	// across any of those boundaries.
	replacedLive := cached != nil
	a.releaseOwnedPinsLocked()
	if !replacedLive && a.sparePins != nil {
		cached, a.sparePins = a.sparePins, nil
	} else {
		cached = new(ownedSessionPins)
	}
	published := false
	defer func() {
		if !published {
			cached.pinner.Unpin()
		}
	}()
	cached.context, cached.clock, cached.entries, cached.faults, cached.backing = m, m.Clock, entries, faults, backingBase
	cached.pinner.Pin(m)
	cached.pinner.Pin(a.sessionImage)
	cached.pinner.Pin(a.sessionState)
	cached.pinner.Pin(&a.stack[0])
	cached.pinner.Pin(&a.entries[0])
	cached.pinner.Pin(&a.targets[0])
	if m.Clock != nil {
		cached.pinner.Pin(m.Clock)
	}
	if backingBase != nil {
		cached.pinner.Pin(&backing[0])
	}
	for i, page := range m.Pages {
		cached.pages[i] = ownedPinPage{unsafe.Pointer(page.Data), unsafe.Pointer(page.Epoch)}
		if page.Data != nil && backingBase == nil {
			cached.pinner.Pin(page.Data)
		}
		if page.Epoch != nil && page.Epoch != m.Clock {
			cached.pinner.Pin(page.Epoch)
		}
	}
	if faults != nil {
		cached.pinner.Pin(&a.faults[0])
	}
	a.ownedPins = cached
	published = true
	return cached
}

func clearSessionPinBorrow(image *[sessionImageWords]uint64) {
	clear(image[:sessionPrefixWords])
	clear(image[sessionDirectWord : sessionDirectWord+3])
	clear(image[sessionTimerWord : sessionTimerWord+3])
	image[sessionTargetsWord], image[sessionDescriptorsWord], image[sessionEpochWord] = 0, 0, 0
}

func sessionBackingContainsPages(m *LinkedContextABI, backing []byte) bool {
	if len(backing) < 4096 {
		return false
	}
	base := uintptr(unsafe.Pointer(&backing[0]))
	for _, page := range m.Pages {
		if page.Data == nil {
			continue
		}
		p := uintptr(unsafe.Pointer(page.Data))
		if p < base || p-base > uintptr(len(backing)-4096) {
			return false
		}
	}
	return true
}

// Every page was already proven wholly inside backing before this call. Check
// the entire backing once for state/context aliases; this may conservatively
// select compatibility for an unused RAM extent, but can never miss a live page
// alias. Independent epoch pointers still need their own checks.
func sessionNeedsBackingCompatibility(state []uint64, m *LinkedContextABI, backing []byte) bool {
	base, size := uintptr(unsafe.Pointer(m)), unsafe.Sizeof(*m)+unsafe.Sizeof(uintptr(0))
	statePtr := unsafe.Pointer(&state[0])
	stateBase, stateSize := uintptr(statePtr), uintptr(len(state))*8
	backingPtr := unsafe.Pointer(&backing[0])
	backingSize := uintptr(len(backing))
	if sessionOverlap(statePtr, stateSize, base, size) ||
		sessionOverlap(unsafe.Pointer(m.Clock), 8, base, size) ||
		sessionOverlap(unsafe.Pointer(m.Clock), 8, stateBase, stateSize) ||
		sessionOverlap(backingPtr, backingSize, base, size) ||
		sessionOverlap(backingPtr, backingSize, stateBase, stateSize) {
		return true
	}
	for _, page := range m.Pages {
		if page.Epoch != m.Clock && (sessionOverlap(unsafe.Pointer(page.Epoch), 8, base, size) || sessionOverlap(unsafe.Pointer(page.Epoch), 8, stateBase, stateSize)) {
			return true
		}
	}
	return false
}
