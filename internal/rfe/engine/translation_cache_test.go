package engine

import (
	emu "renvo.dev/internal/rfe/runtime"
	"testing"
)

func cacheTestOps(value uint64, size int) []emu.Op {
	ops := make([]emu.Op, size)
	for i := range ops {
		ops[i] = emu.Op{Kind: emu.Const, Imm: value + uint64(i)}
	}
	ops[size-1] = emu.Op{Kind: emu.StoreState, A: 0, Imm: 0}
	return ops
}
func TestTranslationCacheWrapRevokesSlicesButNotCompiledOrPreparedCode(t *testing.T) {
	c := translationCache{storage: make([]emu.Op, 20), owners: make([]*compiledBlock, 20), unit: 1}
	var a, b, d compiledBlock
	var stored bool
	a.ops, stored = c.store(&a, cacheTestOps(41, 8))
	if !stored {
		t.Fatal("first store")
	}
	prepared, err := emu.Prepare(a.ops, 1)
	if err != nil {
		t.Fatal(err)
	}
	n, err := emu.NewNative(1 << 20)
	if err != nil {
		t.Skip(err)
	}
	defer n.Close()
	entry, err := n.Compile(a.ops, 1)
	if err != nil {
		t.Fatal(err)
	}
	b.ops, _ = c.store(&b, cacheTestOps(73, 8))
	d.irScore = 2
	d.ops, _ = c.store(&d, cacheTestOps(99, 8)) // wraps after evicting a
	if a.ops != nil || a.irSlot != 0 || b.ops == nil || d.ops == nil {
		t.Fatal("wrap failed to revoke exactly the overwritten slice")
	}
	for _, test := range []struct {
		block *compiledBlock
		want  uint64
	}{{&b, 73}, {&d, 99}} {
		state := []uint64{0}
		if err := emu.Interpret(test.block.ops, state); err != nil || state[0] != test.want {
			t.Fatal("surviving cached semantics changed", state, err)
		}
	}
	state := []uint64{0}
	if err := n.Call(entry, state); err != nil || state[0] != 41 {
		t.Fatal("cache reuse altered native entry", state, err)
	}
	state[0] = 0
	var ir emu.IRMachine
	if err := ir.Run(prepared, state); err != nil || state[0] != 41 {
		t.Fatal("cache reuse altered prepared copy", state, err)
	}
}

func TestTranslationCacheChurnAndReusedBlockOwners(t *testing.T) {
	c := translationCache{storage: make([]emu.Op, 37), owners: make([]*compiledBlock, 37), unit: 1}
	var blocks [8]compiledBlock
	var expected [8]uint64
	for round := 0; round < 400; round++ {
		i := round % len(blocks)
		c.forget(&blocks[i]) // discard then reuse the exact decoded-block object
		blocks[i] = compiledBlock{irScore: uint8(round/8 + 1)}
		expected[i] = uint64(round+1)<<40 | uint64(round)
		var ok bool
		blocks[i].ops, ok = c.store(&blocks[i], cacheTestOps(expected[i], 2+(round*7)%19))
		if !ok {
			state := []uint64{0}
			if err := emu.Interpret(blocks[i].ops, state); err != nil || state[0] != expected[i] {
				t.Fatal("borrowed fallback changed semantics", round, state, err)
			}
			blocks[i].ops = nil // synchronous consumer completed
		}
		if c.used > len(c.storage) || c.count > len(c.owners) || sumCacheScores(c.scores[:]) != c.used {
			t.Fatal("bounded cache failed", round)
		}
		for j := range blocks {
			if blocks[j].irSlot == 0 {
				if blocks[j].ops != nil {
					t.Fatal("evicted slice remained live")
				}
				continue
			}
			state := []uint64{0}
			if err := emu.Interpret(blocks[j].ops, state); err != nil || state[0] != expected[j] {
				t.Fatal("stale owner or overlapping live storage", round, j, state, expected[j], err)
			}
		}
	}
	// Oversized translations use the original synchronous fallback and cannot
	// evict or modify any resident translation.
	before := c.evictions
	var oversized compiledBlock
	ops := cacheTestOps(5, 38)
	out, stored := c.store(&oversized, ops)
	if stored || &out[0] != &ops[0] || c.evictions != before || oversized.irSlot != 0 {
		t.Fatal("oversized fallback mutated cache")
	}
}

func TestTranslationCachePreservesRecompiledHelperOverColdLeaves(t *testing.T) {
	c := translationCache{storage: make([]emu.Op, 20), owners: make([]*compiledBlock, 20), unit: 1}
	var hot, cold, next compiledBlock
	hot.ops, _ = c.store(&hot, cacheTestOps(7, 8))
	cold.ops, _ = c.store(&cold, cacheTestOps(9, 8))
	c.recordCompile(&hot) // actual reuse when recompiling a retained helper
	next.irScore = 2
	next.ops, _ = c.store(&next, cacheTestOps(11, 8))
	if hot.ops == nil || cold.ops != nil || next.ops == nil {
		t.Fatal("hot helper displaced by cold compilation stream")
	}
	state := []uint64{0}
	if err := emu.Interpret(hot.ops, state); err != nil || state[0] != 7 {
		t.Fatal("hot helper corrupted", state, err)
	}
}

func sumCacheScores(scores []int) int {
	n := 0
	for _, v := range scores {
		n += v
	}
	return n
}

func TestTranslationCacheFragmentationFallsBackWithoutEvictingProtectedIR(t *testing.T) {
	c := translationCache{storage: make([]emu.Op, 256), owners: make([]*compiledBlock, 256), unit: 1}
	var blocks [64]compiledBlock
	for i := range blocks {
		blocks[i].irScore = 2
		blocks[i].ops, _ = c.store(&blocks[i], cacheTestOps(uint64(i), 4))
	}
	for i := 1; i < len(blocks); i += 2 {
		c.forget(&blocks[i])
	}
	var next compiledBlock
	input := cacheTestOps(99, 5) // aggregate free capacity exists, but no contiguous eligible extent
	out, stored := c.store(&next, input)
	if stored || &out[0] != &input[0] || c.evictions != 0 || c.used != 128 || sumCacheScores(c.scores[:]) != 128 {
		t.Fatal("fragmentation fallback changed protected storage")
	}
	for i := 0; i < len(blocks); i += 2 {
		state := []uint64{1000}
		if err := emu.Interpret(blocks[i].ops, state); err != nil || state[0] != uint64(i) {
			t.Fatal("protected IR changed", i, state, err)
		}
	}
}
func TestTranslationCacheMultiUnitPaddingAndSaturatedReuse(t *testing.T) {
	c := translationCache{storage: make([]emu.Op, 128), owners: make([]*compiledBlock, 8), unit: 16}
	var blocks [5]compiledBlock
	for round := 0; round < 400; round++ {
		i := round % len(blocks)
		c.forget(&blocks[i])
		blocks[i] = compiledBlock{irScore: uint8(round/5 + 1)}
		ops := cacheTestOps(uint64(round)<<40, 2+round%35)
		out, stored := c.store(&blocks[i], ops)
		if stored {
			blocks[i].ops = out
		}
		if c.used > len(c.owners) || c.used != sumCacheScores(c.scores[:]) {
			t.Fatal("padding escaped unit accounting", round, c.used)
		}
		for j := range blocks {
			if blocks[j].irSlot != 0 && blocks[j].ops[len(blocks[j].ops)-1].Kind != emu.StoreState {
				t.Fatal("multi-unit live slice overlapped", round, j)
			}
		}
	}
	for i := range blocks {
		c.forget(&blocks[i])
	}
	var hot compiledBlock
	hot.ops, _ = c.store(&hot, cacheTestOps(7, 8))
	for i := 0; i < 300; i++ {
		c.recordCompile(&hot)
	}
	if hot.irScore != 255 || c.scores[255] != hot.irUnits || c.used != sumCacheScores(c.scores[:]) {
		t.Fatal("reuse counter wrapped or corrupted ownership")
	}
}
