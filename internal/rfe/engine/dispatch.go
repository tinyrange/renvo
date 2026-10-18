package engine

// Clear both lookup structures together. A colliding direct-cache entry may
// refer to another PC and must remain intact. Native storage stays append-only.
func (e *Engine) discard(pc uint64) {
	old := e.blocks[pc]
	if old := e.blocks[pc]; old != nil && old.cacheSlot >= 0 && old.cacheSlot < len(e.cacheOrder) && e.cacheOrder[old.cacheSlot].block == old {
		e.cacheOrder[old.cacheSlot] = dispatchEntry{}
	}
	delete(e.blocks, pc)
	prefill := &e.prefillLookup[((pc>>2)^(pc>>12))&1023]
	if prefill.pc == pc {
		*prefill = dispatchEntry{}
	}
	entry := &e.dispatch[(pc>>2)&1023]
	if entry.pc == pc {
		*entry = dispatchEntry{}
	}
	if old != nil {
		e.dropTranslation(old)
		e.releaseInstructions(old)
	}
	// Native descriptors contain immutable entry/PC integers, not pointers
	// to this Go metadata. Callers of discard stop using the rejected block.
	if old != nil && len(e.blockPool) < cap(e.blockPool) {
		e.blockPool = append(e.blockPool, old)
	}
}
