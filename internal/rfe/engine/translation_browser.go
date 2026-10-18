//go:build browser

package engine

import (
	"renvo.dev/internal/arena"
	emu "renvo.dev/internal/rfe/runtime"
	"unsafe"
)

func (e *Engine) nameEntry(entry int, pc uint64, count int, kind string) {}

func nativeEntryBudgetNeedsRecycle(n *emu.Native) bool { return n.EntryBudgetNeedsRecycle() }

// WASM installation and PreparedBlock creation consume IR synchronously.
// Admissions rejected by the bounded cache retain the original borrowed path.
// Four MiB of operation storage per engine, reused in place. Unlike a
// cumulative allocation allowance, cold early-boot entries cannot permanently
// prevent later hot translations from entering the cache. No per-store
// allocation or GC assumption is needed once these fixed arrays exist.
const browserRetainedIRLimit = 4 << 20

func (e *Engine) publishTranslation(block *compiledBlock, ops []emu.Op) []emu.Op {
	e.Stats.LeafTranslations++
	if !e.config.RetainBrowserIR {
		return ops
	}
	if e.leafIR.storage == nil {
		e.leafIR.storage = make([]emu.Op, browserRetainedIRLimit/int(unsafe.Sizeof(emu.Op{})))
		e.leafIR.unit = 16
		e.leafIR.owners = make([]*compiledBlock, len(e.leafIR.storage)/e.leafIR.unit)
		e.Stats.IRCacheAllocated = len(e.leafIR.storage) * int(unsafe.Sizeof(emu.Op{}))
	}
	// Cache data and owner arrays were allocated before this scope. Only
	// slice-return/call temporaries may allocate during fixed-storage updates;
	// no such temporary is part of a retained IR value or owner reference.
	low, high := arena.Mark(), arena.PersistMark()
	e.leafIR.recordCompile(block)
	out, stored := e.leafIR.store(block, ops)
	if stored {
		e.Stats.IRCacheStores++
	} else {
		e.Stats.IRCacheBypass++
	}
	e.Stats.IRCacheEvictions = e.leafIR.evictions
	block.ops = out // publish into preowned metadata before reclaiming call scratch
	e.endTranslation(low, high)
	return block.ops
}
func (e *Engine) releaseTranslation(block *compiledBlock) {
	if block.irSlot == 0 {
		block.ops = nil
	}
}
func (e *Engine) dropTranslation(block *compiledBlock) {
	if block.irSlot == 0 {
		return
	}
	e.Stats.IRCacheDiscarded++
	low, high := arena.Mark(), arena.PersistMark()
	e.leafIR.forget(block)
	e.endTranslation(low, high)
}

// Lowering only writes its supplied builder. No builder maps, closures, or
// intermediate values survive: validated scalar IR is copied to preowned
// bounded storage before both allocator cursors are restored.
func (e *Engine) beginTranslation(progress bool) (*emu.Builder, int, int) {
	low, high := arena.Mark(), arena.PersistMark()
	b := &emu.Builder{}
	b.Reset(progress)
	return b, low, high
}
func (e *Engine) endTranslation(low, high int) {
	arena.Rewind(low)
	arena.PersistReset(high)
}
func (e *Engine) finishTranslation(b *emu.Builder, memory bool, low, high int) []emu.Op {
	var ops []emu.Op
	if memory {
		ops = b.FinishMemory(e.arch.StateWords)
	} else {
		ops = b.Finish(e.arch.StateWords)
	}
	if len(ops) > len(e.translationOps) {
		// Oversized IR will be rejected by the ordinary validator. Keep it live
		// rather than truncating or returning reclaimed backing storage.
		return ops
	}
	count := copy(e.translationOps[:], ops)
	e.endTranslation(low, high)
	return e.translationOps[:count]
}

func (e *Engine) reuseTranslation(block *compiledBlock) {
	e.Stats.LeafIRReuses++
	low, high := arena.Mark(), arena.PersistMark()
	e.leafIR.recordCompile(block)
	e.endTranslation(low, high)
}
