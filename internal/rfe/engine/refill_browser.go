//go:build browser

package engine

import (
	"renvo.dev/internal/arena"
	emu "renvo.dev/internal/rfe/runtime"
)

func (e *Engine) beginFastRefill(memory NativeMemory, context *emu.MemoryContext) bool {
	if !e.memoryRetryAllowed {
		return false
	}
	e.fastRefillLow, e.fastRefillHigh = arena.Mark(), arena.PersistMark()
	if m, ok := memory.(interface{ NativeFastRefill(*emu.MemoryContext) bool }); ok {
		if e.native.BeginFastRefill(context, m.NativeFastRefill) {
			return true
		}
	}
	e.endFastRefill(true)
	return false
}
func (e *Engine) endFastRefill(success bool) {
	e.native.EndFastRefill()
	// Only setup tuples/bound-method objects/call temporaries were allocated.
	// No MMU walk, map insertion, watch publication, or stack allocation occurs
	// here. Error objects, if any, must remain live for the caller.
	if success {
		arena.Rewind(e.fastRefillLow)
		arena.PersistReset(e.fastRefillHigh)
	}
}
