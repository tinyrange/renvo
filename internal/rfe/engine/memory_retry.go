package engine

import emu "renvo.dev/internal/rfe/runtime"

// This is a retry hint, never an access or code proof. Only ordinary fully
// writable RAM warrants another dispatch; watched stores/devices keep Step.
func refillReady(context *emu.MemoryContext) bool {
	number := context.Address >> 12
	p := context.PageSlot(number)
	return p.Number == number && p.Data != nil && p.Permissions == 3 && context.Address&4095 <= 4080
}

func (e *Engine) retryMemory(c CPU, remaining uint64) error {
	e.memoryRetry = true
	// Re-enter normal lookup/admission after refill: its architectural walk
	// can change mapping epochs, code identity, permission or the root slot.
	// PC is already the committed precise suffix. Never replay its prefix or
	// extend the original remaining instruction budget. A second exit must
	// interpret, so read-modify-write/second-page misses cannot spin.
	err := e.Run(c, remaining)
	e.memoryRetry = false
	return err
}
