package engine

// recycle runs only at the outer Go scheduling boundary, never inside an arena
// callback. Clear all rooted links before closing an append-only arena. The
// replacement has exactly the original byte cap. Browser leaves may transfer
// only after old-owner revocation and fresh dependency validation.
func (e *Engine) recycle(c CPU) error {
	if !e.config.RecycleNative || e.native == nil || e.native.Bytes < e.config.NativeBytes*7/8 && !nativeEntryBudgetNeedsRecycle(e.native) {
		return nil
	}
	if clock := pauseTranslationClock(c); clock != nil {
		defer resumeTranslationClock(clock)
	}
	if err := e.validateCPU(c); err != nil {
		return err
	}
	if memory, ok := c.MemoryBus().(NativeMemory); ok {
		if context := memory.NativeContext(); context != nil {
			context.ClearLinks()
		}
	}
	clear(e.prefillLookup[:])
	old := e.native
	e.native = nil
	e.rememberNativeRecycle(old)
	// Decoded instructions and leaf IR do not contain executable addresses.
	// Keep this bounded cache and its heat, but revoke every old-owner execution
	// route before any selected code is validated for transfer.
	for _, block := range e.blocks {
		block.native, block.attempted, block.entry = false, false, 0
		block.regionAttempted, block.region, block.regionGeneration = false, false, 0
		resetRegionForRecycle(block)
		block.context = CodeStamp{}
	}
	clear(e.regionPartsScratch[:])
	clear(e.regionBestScratch[:])
	clear(e.regionCycleScratch[:])
	clear(e.regionResultScratch[:])
	e.linkReady = false
	e.linkContext = CodeStamp{}
	clear(e.linkContexts)
	if err := beginNativeRecycle(old); err != nil {
		return err
	}
	if memory, ok := c.MemoryBus().(interface{ ReclaimCode() }); ok {
		memory.ReclaimCode()
	}
	// ReclaimCode releases the previous arena's executable watches. Restore
	// watches for retained dependencies in this same quiescent interval, before
	// any guest write could otherwise change code without updating its stamp.
	// Changed mappings, code versions, or unversioned bytes discard their IR.
	e.rewatchDecoded(c)
	e.Stats.NativeBytes = 0
	n, err := e.recycleNative(old, e.config.NativeBytes)
	if err != nil {
		e.Stats.CompileFailures++
		for _, block := range e.blocks {
			if !block.effects && block.prepared == nil {
				// Pure native leaves never needed an evaluator program. With no
				// replacement arena, let the ordinary path prepare one lazily.
				block.ops = nil
			}
		}
		return nil
	}
	e.native = n
	e.Stats.ArenaResets++
	e.Stats.NativeBytes = n.Bytes
	if e.config.ProfileNative {
		_ = n.EnablePerfMap(e.config.ProfileNativeMonotonic)
		e.Stats.ProfileErrors += n.SymbolErrors
	}
	return nil
}

func (e *Engine) rewatchDecoded(c CPU) {
	for _, block := range e.blocks {
		valid := false
		if len(block.dependencies) != 0 {
			valid = e.validVersions(c, block, CodeStamp{})
		} else if len(block.instructions) != 0 {
			valid = e.validBytes(c, block.instructions[0].PC, block)
		}
		// Temporary invalid marker; all marked blocks are removed below before
		// a normal promotion can observe attempted. Valid leaf IR is unchanged.
		block.attempted = !valid
	}
	// Do not delete while ranging the dense map representation. A fixed batch
	// avoids allocating another MaxBlocks-sized cache during every recycle.
	for {
		var invalid [64]uint64
		count := 0
		for pc, block := range e.blocks {
			if block.attempted {
				invalid[count] = pc
				count++
				if count == len(invalid) {
					break
				}
			}
		}
		if count == 0 {
			return
		}
		for _, pc := range invalid[:count] {
			e.discard(pc)
			e.Stats.Invalidations++
		}
	}
}
