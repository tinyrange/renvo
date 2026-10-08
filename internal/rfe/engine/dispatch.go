package engine

// Clear both lookup structures together. A colliding direct-cache entry may
// refer to another PC and must remain intact. Native storage stays append-only.
func (e *Engine) discard(pc uint64) {
	delete(e.blocks, pc)
	entry := &e.dispatch[(pc>>2)&1023]
	if entry.pc == pc {
		*entry = dispatchEntry{}
	}
}
