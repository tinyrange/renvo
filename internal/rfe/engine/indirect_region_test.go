//go:build !renvo

package engine

import (
	emu "renvo.dev/internal/rfe/runtime"
	"testing"
)

type ownedIndirectMemory struct{ versionedStub }

func (*ownedIndirectMemory) NativeCodeVersions() bool { return true }

type indirectCPU struct{ stubCPU }

func (c *indirectCPU) Step() error {
	ins, err := stubFetch(c.memory, c.state[1])
	if err != nil {
		return err
	}
	return c.ExecuteInstruction(ins)
}
func (c *indirectCPU) ExecuteInstruction(ins Instruction) error {
	if ins.Bits&255 == 3 {
		c.state[1] = c.state[0]
		c.retired++
		return nil
	}
	return c.stubCPU.ExecuteInstruction(ins)
}

func TestIndirectOneShotRegionKeepsDynamicTargetBudgetAndExecuteChecks(t *testing.T) {
	memory := new(ownedIndirectMemory)
	if err := memory.Write(0, 2, 3); err != nil {
		t.Fatal(err)
	}
	arch := stubArchitecture()
	fetch, lower := arch.Fetch, arch.Lower
	arch.Fetch = func(m Memory, pc uint64) (Instruction, error) {
		ins, err := fetch(m, pc)
		if err == nil && ins.Bits&255 == 3 {
			ins.Flow = Successors{}
			ins.Terminal = true
		}
		return ins, err
	}
	arch.Lower = func(b *emu.Builder, ins Instruction, completed int) bool {
		if ins.Bits&255 == 3 {
			b.Store(1, b.Load(0))
			return true
		}
		return lower(b, ins, completed)
	}
	cfg := DefaultEngineConfig()
	cfg.IRThreshold, cfg.NativeThreshold = 1, 1
	cfg.MaxInstructions, cfg.MaxNativeInstructions = 1, 4096
	e, err := New(cfg, arch)
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	if !e.NativeAvailable() {
		t.Skip("native unavailable")
	}
	c := &indirectCPU{stubCPU: stubCPU{memory: memory}}
	if err := e.RunQuanta(c, 1); err != nil {
		t.Fatal(err)
	}
	for _, budget := range []uint64{1, 2, 63, 64, 65, 4096, 65537} {
		before := c.retired
		if err := e.RunQuanta(c, budget); err != nil {
			t.Fatal(err)
		}
		want := budget
		if want > 4096 {
			want = 4096
		}
		if c.retired-before != want || c.state[1] != 0 {
			t.Fatal("indirect self target lost bounded progress", budget, c.retired-before, c.state)
		}
	}
	if !e.blocks[0].region || e.blocks[0].regionInstructions != 1 || e.Stats.LoopIterations == 0 {
		t.Fatal("indirect leaf never specialized", e.Stats)
	}
	// A warmed indirect target is never guessed or cached as an architectural
	// successor: change the live register and keep the original full target.
	c.state[0] = 4096
	before := c.retired
	if err := e.RunQuanta(c, 64); err != nil {
		t.Fatal(err)
	}
	if c.retired != before+1 || c.state[1] != 4096 {
		t.Fatal("dynamic target replaced by root")
	}
	memory.denySecond = true
	memory.epoch++
	before = c.retired
	if err := e.RunQuanta(c, 64); err == nil || c.retired != before || c.state[1] != 4096 {
		t.Fatal("indirect target bypassed execute permission")
	}
	// Mutation through the physical code owner invalidates the old branch proof.
	c.state[1] = 0
	if err := memory.Write(0, 2, 0x101); err != nil {
		t.Fatal(err)
	}
	if err := e.RunQuanta(c, 1); err != nil {
		t.Fatal(err)
	}
	if c.state[0] != 4097 || c.state[1] != 2 || c.retired != before+1 {
		t.Fatal("indirect region survived code mutation", c.state)
	}
}
