package engine

import emu "renvo.dev/internal/rfe/runtime"

// recycle runs only at the outer Go scheduling boundary, never inside an arena
// callback. Clear all rooted links before closing an append-only arena. The
// replacement has exactly the original byte cap; no executable entries survive.
func (e *Engine) recycle(c CPU) error {
	if e.config.DynamicCode {
		if e.native != nil {
			for _, entry := range e.pendingReleases {
				e.native.Release(entry)
			}
		}
		e.pendingReleases = e.pendingReleases[:0]
		if e.native != nil {
			e.Stats.NativeBytes = e.native.Bytes
		}
		return nil
	}
	if !e.config.RecycleNative || e.native == nil || e.native.Bytes < e.config.NativeBytes*7/8 {
		return nil
	}
	defer pauseTranslationClock(c)()
	if err := e.validateCPU(c); err != nil {
		return err
	}
	if memory, ok := c.MemoryBus().(NativeMemory); ok {
		if context := memory.NativeContext(); context != nil {
			context.ClearLinks()
		}
	}
	old := e.native
	e.native = nil
	clear(e.variants)
	clear(e.blocks)
	clear(e.cacheOrder)
	e.cacheNext = 0
	clear(e.dispatch[:])
	e.edgeCounts = nil
	e.linkReady = false
	e.linkContext = CodeStamp{}
	e.linkContexts = nil
	if err := old.Close(); err != nil {
		return err
	}
	if memory, ok := c.MemoryBus().(interface{ ReclaimCode() }); ok {
		memory.ReclaimCode()
	}
	n, err := emu.NewNative(e.config.NativeBytes)
	if err != nil {
		e.Stats.CompileFailures++
		return nil
	}
	e.native = n
	e.Stats.ArenaResets++
	e.Stats.NativeBytes = 0
	if e.config.ProfileNative {
		_ = n.EnablePerfMap(e.config.ProfileNativeMonotonic)
		e.Stats.ProfileErrors += n.SymbolErrors
	}
	return nil
}
