//go:build !renvo

package engine

import (
	emu "renvo.dev/internal/rfe/runtime"
	"testing"
)

// A transient short caller budget must not permanently downgrade the checked
// descriptor (or its private target proof) used by later native sessions.
func TestShortTailPreservesWarmedRegionPublication(t *testing.T) {
	n, err := emu.NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(2, 1); err != nil {
		t.Fatal(err)
	}
	var leaf emu.Builder
	leaf.Store(0, leaf.Binary(emu.Add, leaf.Load(0), leaf.Constant(3)))
	leaf.Store(1, leaf.Constant(2))
	ops := leaf.Finish(2)
	entry, err := n.Compile(ops, 2)
	if err != nil {
		t.Fatal(err)
	}
	var region emu.Builder
	region.Store(0, region.Binary(emu.Add, region.Load(0), region.Constant(9)))
	region.Store(1, region.Constant(8))
	region.Checkpoint(2, 3)
	region.LoopContinue(region.Constant(0))
	regionEntry, err := n.CompileLoop(region.FinishMemory(2), 2, 3)
	if err != nil {
		t.Fatal(err)
	}
	memory := new(genericContextMemory)
	stamp := CodeStamp{Identity: new(CodeIdentity), Epoch: 1}
	memory.context.ClaimLinks(n)
	memory.context.PublishLink(0, regionEntry, 3, [17]uint8{})
	if err = n.PrepareTargetLink(0, regionEntry, 2, 3); err != nil {
		t.Fatal(err)
	}
	memory.context.LookupLink(0).Reserved = ^uint64(0)
	before := *memory.context.LookupLink(0)
	first := &compiledBlock{native: true, attempted: true, region: true, regionAttempted: true, entry: entry, regionEntry: regionEntry, regionInstructions: 3, ops: ops, instructions: []Instruction{{PC: 0, Length: 2, Bits: 0x301, Flow: Successors{Known: true, Next: 2}}}}
	cfg := DefaultEngineConfig()
	cfg.IRThreshold = 1
	cfg.MaxNativeInstructions = 64
	e := Engine{arch: stubArchitecture(), config: cfg, native: n, linkReady: true, blocks: map[uint64]*compiledBlock{0: first}, linkContexts: map[*emu.MemoryContext]CodeStamp{&memory.context: stamp}}
	c := stubCPU{memory: memory}
	if err = e.runLinked(&c, 1, first, memory, &memory.context, stamp, 1); err != nil {
		t.Fatal(err)
	}
	if c.state != [2]uint64{3, 2} || c.retired != 1 || *memory.context.LookupLink(0) != before {
		t.Fatal("short tail changed warmed publication", c.state, c.retired, memory.context.LookupLink(0), before)
	}
	c.state[1] = 0
	if err = e.runLinked(&c, 64, first, memory, &memory.context, stamp, 1); err != nil {
		t.Fatal(err)
	}
	if c.state != [2]uint64{12, 8} || c.retired != 4 || e.Stats.Linked != 3 || *memory.context.LookupLink(0) != before {
		t.Fatal("region did not survive short tail", c.state, c.retired, e.Stats, memory.context.LookupLink(0))
	}
}
