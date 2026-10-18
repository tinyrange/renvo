package engine

import emu "renvo.dev/internal/rfe/runtime"

// Coalesce two fresh observations only for an already-native, current-context
// hit. Cold decoding, dependency revalidation and unsupported owners preserve
// block() and its ordinary separate observation path. No metadata is cached.
func (e *Engine) linkedSnapshot(c CPU, remaining uint64) (*compiledBlock, NativeMemory, *emu.MemoryContext) {
	memory, ok := c.MemoryBus().(NativeSnapshotMemory)
	if !ok {
		return nil, nil, nil
	}
	pc := c.Registers()[e.arch.PC]
	if pc&(e.arch.Alignment-1) != 0 {
		return nil, nil, nil
	}
	cache := &e.dispatch[(pc>>2)&1023]
	block := cache.block
	direct := block != nil && cache.pc == pc
	if !direct {
		block = e.blocks[pc]
	}
	if block == nil || !block.native || len(block.instructions) == 0 || block.instructions[0].ScheduleBoundary || len(block.dependencies) == 0 || uint64(len(block.instructions)) > remaining {
		return nil, nil, nil
	}
	stamp, context := memory.NativeCodeContext()
	e.Stats.ContextChecks++
	if context == nil || stamp.Identity == nil || stamp != block.context {
		return nil, nil, nil
	}
	block.recent = 2
	if !direct {
		*cache = dispatchEntry{pc, block}
	}
	return block, memory, context
}
