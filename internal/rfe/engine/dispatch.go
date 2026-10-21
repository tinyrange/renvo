package engine

func (e *Engine) cachedBlock(pc uint64) *compiledBlock {
	slot := &e.dispatch[e.dispatchSlot(pc)]
	if slot.pc == pc && slot.block != nil && slot.block.cacheContext == e.cacheContext {
		return slot.block
	}
	block := e.blocks[pc]
	if block != nil && block.cacheContext == e.cacheContext {
		return block
	}
	if e.variants != nil {
		return e.variants[contextPC{pc, e.cacheContext}]
	}
	return nil
}

// Clear only this translation variant. A colliding direct-cache entry may
// refer to another PC/context and must remain intact. Native code stays
// append-only; all variants compete in the same global replacement clock.
func (e *Engine) discard(pc uint64) { e.discardVariant(pc, e.cachedBlock(pc)) }
func (e *Engine) discardVariant(pc uint64, old *compiledBlock) {
	if old == nil {
		return
	}
	if old.proof != nil {
		old.proof.Close()
	}
	if old.regionProof != nil && old.regionProof != old.proof {
		old.regionProof.Close()
	}
	if e.config.DynamicCode && old.native {
		e.pendingReleases = append(e.pendingReleases, old.entry)
		if old.region && old.regionEntry != old.entry {
			e.pendingReleases = append(e.pendingReleases, old.regionEntry)
		}
	}
	if old.cacheSlot >= 0 && old.cacheSlot < len(e.cacheOrder) && e.cacheOrder[old.cacheSlot].block == old {
		e.cacheOrder[old.cacheSlot] = dispatchEntry{}
		if e.config.DynamicCode {
			e.freeCacheSlots = append(e.freeCacheSlots, old.cacheSlot)
		}
	}
	if e.blocks[pc] == old {
		delete(e.blocks, pc)
	}
	if e.variants != nil {
		delete(e.variants, contextPC{pc, old.cacheContext})
	}
	entry := &e.dispatch[e.dispatchSlot(pc)]
	if entry.pc == pc && entry.block == old {
		*entry = dispatchEntry{}
	}
}

// Hash the architecture's actual instruction unit, not an assumed four-byte
// word. Byte-aligned x86 PCs must not discard their two most useful tag bits.
func (e *Engine) dispatchSlot(pc uint64) uint64 { return ((pc >> e.dispatchShift) ^ (pc >> 12)) & 1023 }
