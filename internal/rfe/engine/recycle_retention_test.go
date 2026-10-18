package engine

import (
	"encoding/binary"
	emu "renvo.dev/internal/rfe/runtime"
	"testing"
)

// Like system memory, this bus records code writes only while executable pages
// are watched. ReclaimCode is quiescent and drops watches, not page versions.
type recycleWatchMemory struct {
	versionedStub
	watched  [2]bool
	reclaims int
	remapped *CodeIdentity
}

func (m *recycleWatchMemory) CodeContext() CodeStamp {
	stamp := m.versionedStub.CodeContext()
	if m.remapped != nil {
		stamp.Identity = m.remapped
	}
	return stamp
}

func (m *recycleWatchMemory) CodeVersion(at uint64) (CodeStamp, error) {
	stamp, err := m.versionedStub.CodeVersion(at)
	if err == nil {
		m.watched[at/4096] = true
		if m.remapped != nil {
			stamp.Identity = m.remapped
		}
	}
	return stamp, err
}

func (m *recycleWatchMemory) Write(at uint64, size int, value uint64) error {
	if err := m.stubMemory.Write(at, size, value); err != nil {
		return err
	}
	first, last := at/4096, (at+uint64(size)-1)/4096
	if m.watched[first] || m.watched[last] {
		m.epoch++
		if m.watched[first] {
			m.pages[first] = m.epoch
		}
		if m.watched[last] {
			m.pages[last] = m.epoch
		}
	}
	return nil
}

func (m *recycleWatchMemory) ReclaimCode() {
	m.watched = [2]bool{}
	m.epoch++
	m.reclaims++
}

type recycleTranslationCounts struct{ fetch, lower int }

func recycleTestArchitecture(counts *recycleTranslationCounts, checked bool) Architecture {
	arch := stubArchitecture()
	fetch, lower := arch.Fetch, arch.Lower
	arch.Fetch = func(memory Memory, pc uint64) (Instruction, error) {
		counts.fetch++
		ins, err := fetch(memory, pc)
		// A checked effect with no RAM access, like an architectural barrier.
		ins.Memory = checked
		return ins, err
	}
	arch.Lower = func(b *emu.Builder, ins Instruction, completed int) bool {
		counts.lower++
		if checked {
			b.Guard(b.Constant(1))
		}
		return lower(b, ins, completed)
	}
	return arch
}

func newRecycleRetentionEngine(t *testing.T, arch Architecture, threshold uint64) *Engine {
	t.Helper()
	config := DefaultEngineConfig()
	config.RecycleNative, config.EvictCold = true, true
	config.MaxBlocks, config.MaxInstructions = 256, 1
	config.IRThreshold, config.NativeThreshold = 1, threshold
	config.RegionThreshold = ^uint64(0)
	config.NativeBytes = 64 << 10
	e, err := New(config, arch)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { e.Close() })
	if !e.NativeAvailable() {
		t.Skip("native host unavailable")
	}
	return e
}

func forceRecycle(t *testing.T, e *Engine, c *stubCPU) {
	t.Helper()
	// Exercise the ordinary exhaustion branch without compiling unrelated code.
	e.native.Bytes = e.config.NativeBytes * 7 / 8
	if err := e.recycle(c); err != nil {
		t.Fatal(err)
	}
	if e.Stats.ArenaResets == 0 || e.Stats.NativeBytes > e.config.NativeBytes {
		t.Fatal("recycling failed or enlarged the native budget", e.Stats)
	}
}

func TestRecycleRetainsLeafIRAndRewatchesCode(t *testing.T) {
	for _, checked := range []bool{false, true} {
		name := "pure"
		if checked {
			name = "checked"
		}
		t.Run(name, func(t *testing.T) {
			memory := new(recycleWatchMemory)
			memory.Write(0, 2, 0x101)
			counts := new(recycleTranslationCounts)
			e := newRecycleRetentionEngine(t, recycleTestArchitecture(counts, checked), 1)
			c := stubCPU{memory: memory}
			for i := 0; i < 2; i++ {
				c.state[1] = 0
				if err := e.RunQuanta(&c, 1); err != nil {
					t.Fatal(err)
				}
			}
			block, owner := e.blocks[0], e.native
			if block == nil || !block.native || memory.context.LookupLink(0) == nil || !memory.watched[0] {
				t.Fatal("test did not establish a watched, linked native leaf")
			}
			ops, visits, before, blocks := &block.ops[0], block.visits, *counts, e.Stats.Blocks
			forceRecycle(t, e, &c)
			if e.native == owner || e.blocks[0] != block || &block.ops[0] != ops || block.visits != visits || *counts != before {
				t.Fatal("recycling discarded or retranslated valid decoded leaf IR")
			}
			if block.native || block.attempted || block.entry != 0 || block.region || e.linkReady || memory.context.LookupLink(0) != nil {
				t.Fatal("recycling retained an executable entry or link")
			}
			if !memory.watched[0] || memory.reclaims != 1 {
				t.Fatal("code was not rewatched before returning to the guest")
			}
			c.state[1] = 0
			if err := e.RunQuanta(&c, 1); err != nil {
				t.Fatal(err)
			}
			if *counts != before || e.Stats.Blocks != blocks || !block.native || c.state != [2]uint64{3, 2} || c.retired != 3 {
				t.Fatal("retained leaf was decoded/lowered again or executed incorrectly", *counts, before, c.state, c.retired)
			}
			// This write would be invisible without immediate dependency rewatch.
			memory.Write(0, 2, 0x901)
			c.state[1] = 0
			if err := e.RunQuanta(&c, 1); err != nil {
				t.Fatal(err)
			}
			if c.state != [2]uint64{12, 2} || c.retired != 4 || e.Stats.Invalidations == 0 {
				t.Fatal("post-recycle source write ran stale IR", c.state, c.retired)
			}
		})
	}
}

func TestRecycleRejectsStaleVersionAndRemappedRoot(t *testing.T) {
	for _, remap := range []bool{false, true} {
		name := "source_write"
		if remap {
			name = "root_remap"
		}
		t.Run(name, func(t *testing.T) {
			memory := new(recycleWatchMemory)
			memory.Write(0, 2, 0x101)
			e := newRecycleRetentionEngine(t, stubArchitecture(), 1)
			c := stubCPU{memory: memory}
			if err := e.RunQuanta(&c, 1); err != nil {
				t.Fatal(err)
			}
			if remap {
				// A different root can reuse identical numeric page generations.
				memory.remapped = new(CodeIdentity)
				memory.stubMemory.Write(0, 2, 0x901)
			} else {
				memory.Write(0, 2, 0x901)
			}
			forceRecycle(t, e, &c)
			if e.CachedBlocks() != 0 || e.Stats.Invalidations != 1 {
				t.Fatal("stale source survived recycle validation")
			}
			c.state[1] = 0
			if err := e.RunQuanta(&c, 1); err != nil {
				t.Fatal(err)
			}
			if c.state != [2]uint64{10, 2} || c.retired != 2 {
				t.Fatal("stale remapped or modified source executed", c.state)
			}
		})
	}
}

func TestRecycleUnversionedBytesAndPreparedIR(t *testing.T) {
	memory := new(stubMemory)
	binary.LittleEndian.PutUint16(memory.bytes[:], 0x101)
	counts := new(recycleTranslationCounts)
	e := newRecycleRetentionEngine(t, recycleTestArchitecture(counts, false), 8)
	c := stubCPU{memory: memory}
	if err := e.RunQuanta(&c, 1); err != nil {
		t.Fatal(err)
	}
	block := e.blocks[0]
	if block == nil || block.prepared == nil || block.native {
		t.Fatal("test did not establish an evaluator leaf")
	}
	prepared, before := block.prepared, *counts
	forceRecycle(t, e, &c)
	if e.blocks[0] != block || block.prepared != prepared || counts.fetch != before.fetch+1 || counts.lower != before.lower {
		t.Fatal("unversioned recycle failed exact validation or discarded evaluator IR")
	}
	c.state[1] = 0
	if err := e.RunQuanta(&c, 1); err != nil {
		t.Fatal(err)
	}
	if c.state != [2]uint64{2, 2} || e.Stats.IR != 2 || counts.lower != before.lower {
		t.Fatal("retained evaluator changed execution", c.state)
	}
	memory.Write(0, 2, 0x901)
	forceRecycle(t, e, &c)
	if e.CachedBlocks() != 0 || e.Stats.Invalidations != 1 {
		t.Fatal("unversioned source change survived recycle")
	}
}

func TestRecycleInvalidatesMoreThanOneRemovalBatch(t *testing.T) {
	memory := new(recycleWatchMemory)
	for i := 0; i < 130; i++ {
		memory.Write(uint64(i*2), 2, 0x101)
	}
	e := newRecycleRetentionEngine(t, stubArchitecture(), 1000)
	c := stubCPU{memory: memory}
	for i := 0; i < 130; i++ {
		if err := e.RunQuanta(&c, 1); err != nil {
			t.Fatal(err)
		}
	}
	if e.CachedBlocks() != 130 {
		t.Fatal("test did not establish multiple invalidation batches")
	}
	memory.Write(0, 2, 0x901) // All cached leaves depend on this code page.
	forceRecycle(t, e, &c)
	if e.CachedBlocks() != 0 || e.Stats.Invalidations != 130 {
		t.Fatal("recycle skipped invalid blocks while removing cache entries", e.CachedBlocks(), e.Stats.Invalidations)
	}
}
