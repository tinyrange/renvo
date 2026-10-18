//go:build browser

package engine

import (
	"renvo.dev/internal/arena"
	emu "renvo.dev/internal/rfe/runtime"
)

func (e *Engine) prefillBrowserLinks(c CPU, first *compiledBlock, memory NativeMemory, context *emu.MemoryContext, stamp CodeStamp, published bool) bool {
	// A new root publication (including after every context clear) can seed
	// its bounded successor group. New native promotions or newly observed
	// dynamic edges can expose targets unavailable to the previous prefill.
	// Descriptor collisions may still cause a normal miss; optional prefill
	// is never admission proof.
	if !published && !first.prefillDirty && first.prefillGeneration == e.Stats.Promotions {
		return true
	}
	if !e.prefillLinks(c, first, memory, context, stamp) {
		return false
	}
	first.prefillGeneration = e.Stats.Promotions
	first.prefillDirty = false
	return true
}

func (e *Engine) observeBrowserLink(context *emu.MemoryContext, target uint64) {
	// Only scalar hints in preowned block metadata survive this call.
	low, high := arena.Mark(), arena.PersistMark()
	e.observeLink(context.LinkedSourcePC(), target)
	e.endTranslation(low, high)
}
