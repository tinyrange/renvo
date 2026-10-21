package engine

// admit is bounded second-chance clock replacement. Cold sequential code must
// not evict the hot working set, nor require a full map scan per new instruction.
// No guest execution occurs during this at-most-three-sweeps search.
func (e *Engine) admit(pc uint64, block *compiledBlock) {
	if e.config.DynamicCode {
		slot := len(e.cacheOrder)
		if len(e.freeCacheSlots) != 0 {
			slot = e.freeCacheSlots[len(e.freeCacheSlots)-1]
			e.freeCacheSlots = e.freeCacheSlots[:len(e.freeCacheSlots)-1]
		} else {
			e.cacheOrder = append(e.cacheOrder, dispatchEntry{})
		}
		block.cacheSlot = slot
		e.cacheOrder[slot] = dispatchEntry{pc, block}
		return
	}
	for {
		slot := e.cacheNext
		e.cacheNext = (slot + 1) % len(e.cacheOrder)
		old := e.cacheOrder[slot]
		if old.block != nil {
			if old.block.recent != 0 {
				old.block.recent--
				continue
			}
			e.discardVariant(old.pc, old.block)
			e.Stats.Evictions++
		}
		block.cacheSlot = slot
		e.cacheOrder[slot] = dispatchEntry{pc, block}
		return
	}
}
