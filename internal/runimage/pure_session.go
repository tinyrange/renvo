//go:build !renvo

package runimage

import (
	"fmt"
	"renvo.dev/internal/rfeabi"
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
	sessionMaskWord        = unsafe.Offsetof(LinkedContextABI{}.DescriptorMask) / 8
	sessionDescriptorsWord = unsafe.Offsetof(LinkedContextABI{}.DescriptorBase) / 8
	sessionRemainingWord   = unsafe.Offsetof(LinkedContextABI{}.Remaining) / 8
	sessionTotalWord       = unsafe.Offsetof(LinkedContextABI{}.Total) / 8
	sessionMemoryWord      = unsafe.Offsetof(LinkedContextABI{}.MemoryTotal) / 8
	sessionLinkEpochWord   = unsafe.Offsetof(LinkedContextABI{}.LinkEpoch) / 8
	sessionStopWord        = unsafe.Offsetof(LinkedContextABI{}.Stop) / 8
	sessionOwnedWord       = unsafe.Offsetof(LinkedContextABI{}.Memory) / 8
	sessionDirectWord      = unsafe.Offsetof(LinkedContextABI{}.DirectGuest) / 8
)

func ForeignSessionsAvailable() bool { return rfenativebridge.Available }

// CallSession executes one exact-budget slice through a cgo boundary. The
// pointer-free staging allocations exclude enclosing Go owner graphs. All memory
// reachable by native code is explicitly rooted and pinned for the duration.
// Native effects that could change the context's pointer graph mid-call use
// the conservative original <=64-instruction boundary instead.
func (c *LinkedCall) CallSession(state []uint64, m *LinkedContextABI, budget uint64) error {
	return c.callSession(state, m, budget, false, nil)
}

// CallOwnedSession permits pin reuse only for the host-owned address-space
// contract. Every call still verifies non-aliasing, compares every live pointer,
// issues a fresh admission epoch, and enforces the same scheduling slice.
func (c *LinkedCall) CallOwnedSession(state []uint64, m *LinkedContextABI, budget uint64) error {
	return c.callSession(state, m, budget, true, nil)
}

// CallOwnedBackingSession pins one contiguous Go byte allocation instead of
// repeatedly pinning its interior pages. The entire live page graph is checked
// against this allocation on every call; no permissions or aliases are waived.
func (c *LinkedCall) CallOwnedBackingSession(state []uint64, m *LinkedContextABI, budget uint64, backing []byte) error {
	return c.callSession(state, m, budget, true, backing)
}
func (c *LinkedCall) callSession(state []uint64, m *LinkedContextABI, budget uint64, owned bool, backing []byte) (err error) {
	clearContextBorrow(m)
	if c == nil || c.arena == nil || m == nil || len(state) != c.words || budget == 0 || budget > SessionInstructionLimit && m.Stop == nil {
		c.ReleaseOwnedSessionPins()
		return fmt.Errorf("invalid native execution session")
	}
	if len(backing) != 0 && !sessionBackingContainsPages(m, backing) {
		c.ReleaseOwnedSessionPins()
		return fmt.Errorf("native page outside owned backing")
	}
	if sessionDecodeVersionsAlias(state, m) {
		return fmt.Errorf("decode versions alias execution state or guest RAM")
	}
	if m.Links != nil && (!m.Links.Valid() || m.Links.Dynamic() && m.DescriptorMask != m.Links.Mask) {
		return fmt.Errorf("invalid native publication owner")
	}
	if m.Memory != nil && (!owned || !m.Memory.Valid()) {
		return fmt.Errorf("invalid scalable memory owner")
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
		a.releaseBackingPinsLocked()
	}
	if a.base == 0 || a.broken {
		a.releaseOwnedPinsLocked()
		return fmt.Errorf("invalid native execution owner")
	}
	m.Remaining, m.Total, m.MemoryTotal, m.Status, m.Retired = budget, 0, 0, 0, 0
	m.LoopExits, m.LoopIterations = 0, 0
	actual := budget
	compatible := false
	if m.Links != nil && sessionLinksAlias(state, m, backing) {
		compatible = true
	}
	if len(backing) != 0 {
		compatible = compatible || sessionNeedsBackingCompatibility(state, m, backing)
	} else {
		compatible = compatible || sessionNeedsCompatibility(state, m)
	}
	if !rfenativebridge.Available || compatible {
		a.releaseOwnedPinsLocked()
		a.releaseBackingPinsLocked()
		if actual > 64 {
			actual = 64
		}
		m.Remaining = actual
		a.borrowLinked(&m.CodeView, &m.PreparedTargets, m)
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
		image[sessionOwnedWord] = uint64(uintptr(unsafe.Pointer(m.Memory)))
		image[sessionStopWord] = uint64(uintptr(unsafe.Pointer(m.Stop)))
		image[sessionLinkEpochWord] = m.LinkEpoch
		image[sessionMaskWord] = m.DescriptorMask
		defer func() {
			clear(image[:sessionPrefixWords])
			clear(image[sessionDirectWord : sessionDirectWord+3])
			image[sessionOwnedWord] = 0
			image[sessionStopWord] = 0
			image[sessionTargetsWord], image[sessionDescriptorsWord], image[sessionEpochWord] = 0, 0, 0
		}() // integer addresses must not outlive their pins
		if owned {
			pins = &a.pinOwnedSession(m, backing).pinner
		} else {
			pins.Pin(m) // native reads only pointer-free descriptor storage
			if m.Links != nil {
				pins.Pin(m.Links)
				if m.Links.Dynamic() {
					pins.Pin(m.Links.Base)
				}
			}
			pins.Pin(image)
			pins.Pin(a.sessionState)
			pins.Pin(&a.stack[0])
			pins.Pin(&a.entries[0])
			pins.Pin(&a.targets[0])
			if m.Clock != nil {
				pins.Pin(m.Clock)
			}
			if m.Stop != nil {
				pins.Pin(m.Stop)
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
		image[sessionDescriptorsWord] = uint64(uintptr(unsafe.Pointer(m.DescriptorPointer())))
		if !owned && len(a.faults) != 0 {
			pins.Pin(&a.faults[0])
		}
		rfenativebridge.CallFaults(c.entry, unsafe.Pointer(&nativeState[0]), unsafe.Pointer(image), c.top, a.faults)
		copy(state, nativeState)
		m.Retired, m.Status, m.Address = image[0], image[1], image[2]
		m.Remaining, m.Total, m.MemoryTotal = image[sessionRemainingWord], image[sessionTotalWord], image[sessionMemoryWord]
		m.LoopExits, m.LoopIterations = image[sessionExitsWord], image[sessionIterationsWord]
	}
	if owned && m.Memory == nil && (m.Status != 0 || m.Total == 0) {
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
	if m.Stop != nil {
		ptr := unsafe.Pointer(m.Stop)
		if sessionOverlap(ptr, 4, base, size) || sessionOverlap(ptr, 4, uintptr(unsafe.Pointer(&state[0])), uintptr(len(state))*8) {
			return true
		}
		for _, page := range m.Pages {
			if page.Data != nil && sessionOverlap(ptr, 4, uintptr(unsafe.Pointer(page.Data)), 4096) {
				return true
			}
		}
	}
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
	linkBase                      *rfeabi.Descriptor
	links                         *rfeabi.LinkTable
	watchCount                    int
	pinner                        runtime.Pinner
	context                       *LinkedContextABI
	clock                         *uint64
	stop                          *uint32
	entries                       unsafe.Pointer
	faults                        unsafe.Pointer
	backing                       unsafe.Pointer
	pages                         [64]ownedPinPage
	memory                        *rfeabi.OwnedMemory
	memoryRevision, watchRevision uint64
}

func (a *CodeArena) releaseOwnedPinsLocked() {
	if a.ownedPins != nil {
		released := a.ownedPins
		released.pinner.Unpin()
		// Keep only reusable bookkeeping, never roots from the retired graph.
		released.context, released.clock = nil, nil
		released.links = nil
		released.linkBase = nil
		released.stop = nil
		released.entries, released.faults, released.backing = nil, nil, nil
		clear(released.pages[:])
		released.memory = nil
		released.memoryRevision = 0
		released.watchRevision = 0
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
	a.releaseBackingPinsLocked()
	a.mu.Unlock()
}

func (a *CodeArena) pinOwnedSession(m *LinkedContextABI, backing []byte) *ownedSessionPins {
	if m.Memory != nil {
		a.pinBackingLocked(m.Memory)
	}
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
	match := cached != nil && cached.context == m && cached.links == m.Links && cached.linkBase == m.DescriptorPointer() && cached.clock == m.Clock && cached.stop == m.Stop && cached.entries == entries && cached.faults == faults && cached.backing == backingBase
	if match {
		match = cached.memory == m.Memory
		if match && m.Memory != nil {
			match = cached.memoryRevision == m.Memory.Revision
		}
	}
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
	cached.stop = m.Stop
	if m.Stop != nil {
		cached.pinner.Pin(m.Stop)
	}
	cached.pinner.Pin(m)
	cached.links = m.Links
	cached.linkBase = m.DescriptorPointer()
	if m.Links != nil {
		cached.pinner.Pin(m.Links)
		if m.Links.Dynamic() {
			cached.pinner.Pin(m.Links.Base)
		}
	}
	cached.pinner.Pin(a.sessionImage)
	cached.pinner.Pin(a.sessionState)
	cached.pinner.Pin(&a.stack[0])
	cached.pinner.Pin(&a.entries[0])
	cached.pinner.Pin(&a.targets[0])
	if m.Clock != nil {
		cached.pinner.Pin(m.Clock)
	}
	if backingBase != nil && m.Memory == nil {
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
	cached.memory = m.Memory
	if m.Memory != nil {
		memory := m.Memory
		cached.memoryRevision, cached.watchRevision = memory.Revision, memory.Watches.Revision
		cached.pinner.Pin(memory)
		cached.pinner.Pin(memory.Base)
		cached.pinner.Pin(memory.Clock)

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
	image[sessionOwnedWord] = 0
	image[sessionStopWord] = 0
	image[sessionTargetsWord], image[sessionDescriptorsWord], image[sessionEpochWord] = 0, 0, 0
}

func sessionBackingContainsPages(m *LinkedContextABI, backing []byte) bool {
	if len(backing) < 4096 {
		return false
	}
	if m.Memory != nil && (!m.Memory.Valid() || m.Memory.Backing != &backing[0] || m.Memory.Bytes != uint64(len(backing))) {
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
	if m.Stop != nil && (sessionOverlap(unsafe.Pointer(m.Stop), 4, base, size) || sessionOverlap(unsafe.Pointer(m.Stop), 4, stateBase, stateSize) || sessionOverlap(unsafe.Pointer(m.Stop), 4, uintptr(backingPtr), backingSize)) {
		return true
	}
	if sessionOverlap(statePtr, stateSize, base, size) ||
		sessionOverlap(unsafe.Pointer(m.Clock), 8, base, size) ||
		sessionOverlap(unsafe.Pointer(m.Clock), 8, stateBase, stateSize) ||
		sessionOverlap(backingPtr, backingSize, base, size) ||
		sessionOverlap(backingPtr, backingSize, stateBase, stateSize) {
		return true
	}
	if m.Memory != nil {
		memory := m.Memory
		if memory.Clock != m.Clock || sessionOverlap(backingPtr, backingSize, uintptr(unsafe.Pointer(memory)), unsafe.Sizeof(*memory)) || sessionOverlap(backingPtr, backingSize, uintptr(unsafe.Pointer(memory.Base)), uintptr(memory.Mask+1)*32) || sessionOverlap(backingPtr, backingSize, uintptr(unsafe.Pointer(memory.Watches.Base)), uintptr(memory.Watches.Count)*8) || sessionOverlap(statePtr, stateSize, uintptr(unsafe.Pointer(memory)), unsafe.Sizeof(*memory)) {
			return true
		}
	}
	for _, page := range m.Pages {
		if page.Epoch != m.Clock && (sessionOverlap(unsafe.Pointer(page.Epoch), 8, base, size) || sessionOverlap(unsafe.Pointer(page.Epoch), 8, stateBase, stateSize)) {
			return true
		}
	}
	return false
}

func sessionLinksAlias(state []uint64, m *LinkedContextABI, backing []byte) bool {
	if m.Links == nil {
		return false
	}
	base, size := uintptr(unsafe.Pointer(m.Links)), unsafe.Sizeof(*m.Links)
	if m.Links.Dynamic() {
		base = uintptr(unsafe.Pointer(m.Links.Base))
		size = uintptr(len(m.Links.Entries())) * 64
	}
	if sessionOverlap(unsafe.Pointer(m.Clock), 8, base, size) || sessionOverlap(unsafe.Pointer(m.Stop), 4, base, size) {
		return true
	}
	for _, page := range m.Pages {
		if sessionOverlap(unsafe.Pointer(page.Epoch), 8, base, size) {
			return true
		}
	}
	if sessionOverlap(unsafe.Pointer(m), unsafe.Sizeof(*m), base, size) || sessionOverlap(unsafe.Pointer(&state[0]), uintptr(len(state))*8, base, size) {
		return true
	}
	if len(backing) != 0 {
		return sessionOverlap(unsafe.Pointer(&backing[0]), uintptr(len(backing)), base, size)
	}
	if m.Memory != nil {
		return sessionOverlap(unsafe.Pointer(m.Memory.Backing), uintptr(m.Memory.Bytes), base, size)
	}
	for _, page := range m.Pages {
		if page.Data != nil && sessionOverlap(unsafe.Pointer(page.Data), 4096, base, size) {
			return true
		}
	}
	return false
}

// Physical decode epochs are owner metadata, not guest-addressable memory.
// This is a constant-size span check, independent of RAM/watch page count.
func sessionDecodeVersionsAlias(state []uint64, m *LinkedContextABI) bool {
	if m.Memory == nil || m.Memory.DecodeEpochs == nil {
		return false
	}
	ptr := unsafe.Pointer(m.Memory.DecodeEpochs)
	size := uintptr(len(m.Memory.DecodeVersions())) * 8
	overlap := func(other unsafe.Pointer, n uintptr) bool { return sessionOverlap(other, n, uintptr(ptr), size) }
	if overlap(unsafe.Pointer(m.Memory.Backing), uintptr(m.Memory.Bytes)) || overlap(unsafe.Pointer(&state[0]), uintptr(len(state))*8) || overlap(unsafe.Pointer(m), unsafe.Sizeof(*m)+unsafe.Sizeof(uintptr(0))) || overlap(unsafe.Pointer(m.Clock), 8) || overlap(unsafe.Pointer(m.Stop), 4) || overlap(unsafe.Pointer(m.Memory), unsafe.Sizeof(*m.Memory)) {
		return true
	}
	if m.Links != nil && (overlap(unsafe.Pointer(m.Links), unsafe.Sizeof(*m.Links)) || m.Links.Dynamic() && overlap(unsafe.Pointer(m.Links.Base), uintptr(len(m.Links.Entries()))*64)) {
		return true
	}
	return false
}
