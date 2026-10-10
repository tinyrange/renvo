//go:build !renvo

package engine

import (
	emu "renvo.dev/internal/rfe/runtime"
	"testing"
)

func TestRegionUpgradeAfterLateNeighborPromotion(t *testing.T) {
	n, err := emu.NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(2, 1); err != nil {
		t.Fatal(err)
	}
	stamp := CodeStamp{Identity: new(CodeIdentity), Epoch: 1}
	c := stubCPU{memory: &regionDiscoveryMemory{stamp: stamp}}
	block := func(pc, next, add uint64) *compiledBlock {
		ins := Instruction{PC: pc, Length: 2, Bits: add<<8 | 1, Flow: Successors{Known: true, Next: next}}
		if next == 0 {
			ins.Terminal, ins.Bits = true, 3
		}
		return &compiledBlock{native: true, instructions: []Instruction{ins}, context: stamp, dependencies: []codeDependency{{0, stamp}}}
	}
	first, second, third := block(0, 2, 3), block(2, 4, 5), block(4, 0, 0)
	e := Engine{arch: stubArchitecture(), config: DefaultEngineConfig(), native: n, blocks: map[uint64]*compiledBlock{0: first, 2: second}}
	e.config.MaxRegionInstructions = 3
	e.Stats.Promotions = 2
	e.prepareRegion(&c, first)
	if !first.region || first.regionInstructions != 2 {
		t.Fatal("initial partial region", first.regionInstructions, e.Stats)
	}
	oldEntry, oldBytes := first.regionEntry, n.Bytes
	e.blocks[4] = third
	e.prepareRegion(&c, first)
	if first.regionEntry != oldEntry || n.Bytes != oldBytes {
		t.Fatal("retried unchanged promotion generation")
	}
	e.Stats.Promotions++
	e.prepareRegion(&c, first)
	if first.regionInstructions != 3 || first.regionEntry == oldEntry || e.Stats.Regions != 2 {
		t.Fatal("late neighbor did not upgrade region", first.regionInstructions, e.Stats)
	}
	grownBytes := n.Bytes
	e.prepareRegion(&c, first)
	e.Stats.Promotions++
	e.prepareRegion(&c, first)
	if n.Bytes != grownBytes || e.Stats.Regions != 2 {
		t.Fatal("duplicate region compilation", n.Bytes, grownBytes)
	}
	for _, entry := range []int{oldEntry, first.regionEntry} {
		for budget := uint64(1); budget <= 17; budget++ {
			m := new(emu.MemoryContext)
			m.ClaimLinks(n)
			count := 3
			if entry == oldEntry {
				count = 2
			}
			m.PublishLink(0, entry, count, [17]uint8{})
			state := []uint64{11, 0}
			if err = n.CallLinked(state, m, budget); err != nil {
				t.Fatal(err)
			}
			iterations := budget / uint64(count)
			if entry == oldEntry && iterations > 1 {
				iterations = 1
			}
			pc := uint64(0)
			if entry == oldEntry && iterations != 0 {
				pc = 4
			}
			if state[0] != 11+iterations*8 || state[1] != pc || m.Total != iterations*uint64(count) || m.Remaining != budget-m.Total {
				t.Fatal("old/new immutable region accounting", entry, budget, state, m.Total, m.Remaining)
			}
		}
	}
}
