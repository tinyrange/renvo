//go:build renvo && browser

package runtime

import (
	"renvo.dev/internal/arena"
	"unsafe"
)

// A synchronous dispatcher alone owns this capability. The callback cannot
// select a context, enter an arena, walk page tables or publish code.
var browserRefillOwner *Native

func (n *Native) BeginFastRefill(m *MemoryContext, refill func(*MemoryContext) bool) bool {
	if n == nil || n.device == nil || browserRefillOwner != nil || n.callContext != nil || m == nil || m.linkOwner != n || m.Clock == nil || !n.codeVersionedContexts[m] || refill == nil {
		return false
	}
	if len(n.refillStack) != 40960 {
		return false
	}
	n.refillContext, n.refill, n.refillClock = m, refill, m.Clock
	n.refillLeft = 64
	browserRefillOwner = n
	return true
}
func (n *Native) EndFastRefill() {
	if browserRefillOwner != n {
		return
	}
	n.refillContext.wasmServiceSP, n.refillContext.wasmServiceFP = 0, 0
	n.refillContext, n.refill, n.refillClock = nil, nil, nil
	browserRefillOwner = nil
}

// BrowserRefill is reached through the raw exported direct ABI, using ONLY
// the separate preallocated stacks above. Never call browser_step recursively.
func BrowserRefill() bool {
	n := browserRefillOwner
	if n == nil || n.refillRunning || n.device == nil || n.refillLeft == 0 {
		return false
	}
	m := n.refillContext
	if m == nil || n.callContext != m || m.linkOwner != n || m.Clock != n.refillClock || m.Status != 1 || m.Remaining == 0 {
		return false
	}
	n.refillLeft--
	n.refillRunning = true
	low, high := arena.Mark(), arena.PersistMark()
	ok := n.refill(m)
	arena.Rewind(low)
	arena.PersistReset(high)
	n.refillRunning = false
	return ok
}

// Public/copyable contexts never select an AOT stack. Publish the private
// stack coordinates afresh immediately before the actual synchronous call,
// and only for the exact live capability. Even a forged nonzero SP/FP on an
// ordinary checked CallMemory must be overwritten BEFORE a raw call is possible.
func (n *Native) prepareFastRefill(m *MemoryContext) {
	if m == nil {
		return
	}
	m.wasmServiceSP, m.wasmServiceFP = 0, 0
	if browserRefillOwner != n || n.refillContext != m || m.linkOwner != n || m.Clock != n.refillClock || !n.codeVersionedContexts[m] || len(n.refillStack) != 40960 {
		return
	}
	// Disjoint 64 KiB expression and 256 KiB frame stacks, owned by this arena.
	m.wasmServiceSP = uint32(uintptr(unsafe.Pointer(&n.refillStack[0])))
	m.wasmServiceFP = uint32(uintptr(unsafe.Pointer(&n.refillStack[len(n.refillStack)-1])))
}
