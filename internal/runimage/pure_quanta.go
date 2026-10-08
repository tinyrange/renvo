//go:build !renvo

package runimage

import (
	"fmt"
	"renvo.dev/internal/rfeabi"
	"runtime"
	"unsafe"
)

// Host views share the runtime's authoritative typed ABI prefix.
type LinkedContextABI = rfeabi.Context
type LinkedPageABI = rfeabi.Page
type LinkedDescriptorABI = rfeabi.Descriptor

// CallQuanta schedules only separate <=64-instruction dispatcher calls. Unlike
// CallBatch it inlines the fixed runtime selector/accounting protocol in Go.
// Every callContext invocation returns to this Go loop, where progress is
// validated before another budget is installed. The lock does not enlarge a
// native call or permit code/mapping mutation between calls.
func (c *LinkedCall) CallQuanta(state []uint64, m *LinkedContextABI, pcSlot, limit int, remaining, generation uint64) (completed int, err error) {
	clearContextBorrow(m)
	if c == nil || c.arena == nil || m == nil || len(state) != c.words || pcSlot < 0 || pcSlot >= len(state) || limit < 1 || limit > 16 || remaining == 0 {
		return 0, fmt.Errorf("invalid prepared dispatcher quanta")
	}
	a := c.arena
	initial := remaining
	var total, memory, exits, iterations uint64
	a.mu.Lock()
	defer func() {
		m.Total, m.MemoryTotal, m.LoopExits, m.LoopIterations = total, memory, exits, iterations
		m.Remaining = initial - total
		clearContextBorrow(m)
		a.mu.Unlock()
		runtime.KeepAlive(m)
		runtime.KeepAlive(state)
		runtime.KeepAlive(a.stack)
		runtime.KeepAlive(a.entries)
	}()
	if a.base == 0 || a.broken {
		return 0, fmt.Errorf("invalid prepared dispatcher owner")
	}
	m.Total, m.MemoryTotal, m.Status, m.Retired = 0, 0, 0, 0
	m.LoopExits, m.LoopIterations = 0, 0
	a.borrowLinked(&m.CodeView, &m.PreparedTargets)
	for completed < limit {
		budget := remaining
		if budget > 64 {
			budget = 64
		}
		// Generated dispatchers already add to these cumulative counters.
		// Preserve them across separate calls; only budget/leaf progress reset.
		m.Remaining, m.Retired = budget, 0
		m.AdmissionEpoch = 0
		callContext(c.entry, uintptr(unsafe.Pointer(&state[0])), uintptr(unsafe.Pointer(m)), c.top)
		// This is a real Go boundary after one, and only one, native quantum.
		completed++
		if (m.Status > 2 && m.Status != 4) || m.Total < total || m.Total-total > budget || m.Remaining != budget-(m.Total-total) || m.MemoryTotal < memory || m.MemoryTotal-memory > m.Total-total {
			return completed, fmt.Errorf("invalid native linked progress")
		}
		progress := m.Total - total
		total, memory = m.Total, m.MemoryTotal
		exits, iterations = m.LoopExits, m.LoopIterations
		remaining -= progress
		if completed == limit || remaining == 0 || m.Status != 0 || progress == 0 {
			break
		}
		pc := state[pcSlot]
		var link *LinkedDescriptorABI
		base := ((pc >> 2) ^ (pc >> 12)) & 255
		for way := uint64(0); way < 4; way++ {
			candidate := &m.Blocks[base+way*256]
			if candidate.PC == pc && candidate.Instructions != 0 {
				link = candidate
				break
			}
		}
		if link == nil {
			return completed, nil
		}
		if link.PC != pc || link.Instructions == 0 || link.Instructions > remaining || link.Reserved != generation && link.Reserved != ^uint64(0) {
			break
		}
	}
	return completed, nil
}
