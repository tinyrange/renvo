package engine

// admit is bounded second-chance clock replacement. Cold sequential code must
// not evict the hot working set, nor require a full map scan per new instruction.
// No guest execution occurs during this at-most-three-sweeps search.
func (e *Engine) admit(pc uint64, block *compiledBlock) {
	for {
		slot := e.cacheNext
		e.cacheNext = (slot + 1) % len(e.cacheOrder)
		old := e.cacheOrder[slot]
		if old.block != nil && e.blocks[old.pc] == old.block {
			if old.block.recent != 0 {
				old.block.recent--
				continue
			}
			e.discard(old.pc)
			e.Stats.Evictions++
		}
		block.cacheSlot = slot
		e.cacheOrder[slot] = dispatchEntry{pc, block}
		return
	}
}
