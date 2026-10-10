//go:build !renvo

package engine

import (
	emu "renvo.dev/internal/rfe/runtime"
	"testing"
)

type genericContextMemory struct {
	stubMemory
	context emu.MemoryContext
}

func (m *genericContextMemory) NativeContext() *emu.MemoryContext { return &m.context }
func (m *genericContextMemory) NativeRefill(uint64)               {}

func TestSpecializedArenaFallsBackForGenericContext(t *testing.T) {
	n, err := emu.NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(2, 1); err != nil {
		t.Fatal(err)
	}
	var owned emu.Builder
	owned.Store(0, owned.Binary(emu.Add, owned.Load(0), owned.Constant(1)))
	owned.Store(1, owned.Constant(2))
	owned.Checkpoint(2, 1)
	owned.LoopContinue(owned.Constant(0))
	if _, err = n.CompileCodeVersionedLoop(owned.FinishMemory(2), 2, 1, new(emu.MemoryContext)); err != nil {
		t.Fatal(err)
	}
	var plain emu.Builder
	plain.Store(0, plain.Binary(emu.Add, plain.Load(0), plain.Constant(3)))
	plain.Store(1, plain.Constant(2))
	ops := plain.Finish(2)
	entry, err := n.Compile(ops, 2)
	if err != nil {
		t.Fatal(err)
	}
	memory := new(genericContextMemory)
	c := stubCPU{memory: memory}
	first := &compiledBlock{native: true, attempted: true, regionAttempted: true, entry: entry, ops: ops, instructions: []Instruction{{PC: 0, Length: 2, Bits: 0x301, Flow: Successors{Known: true, Next: 2}}}}
	config := DefaultEngineConfig()
	config.IRThreshold = 1
	e := Engine{arch: stubArchitecture(), config: config, native: n, linkReady: true, blocks: map[uint64]*compiledBlock{0: first}}
	if err = e.runLinked(&c, 64, first, memory, &memory.context, CodeStamp{}, 1); err != nil {
		t.Fatal(err)
	}
	if c.state != [2]uint64{3, 2} || c.retired != 1 || e.Stats.Linked != 0 || e.Stats.Native != 1 {
		t.Fatal("generic leaf fallback", c.state, c.retired, e.Stats)
	}
}

func (m *genericContextMemory) CodeContext() CodeStamp                { return CodeStamp{} }
func (m *genericContextMemory) CodeVersion(uint64) (CodeStamp, error) { return CodeStamp{}, nil }
