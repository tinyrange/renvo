//go:build !renvo

package engine_test

import (
	"fmt"
	"renvo.dev/internal/rfe/engine"
	emu "renvo.dev/internal/rfe/runtime"
	"renvo.dev/internal/rfe/systemmemory"
	"testing"
)

type proofCPU struct {
	state   [2]uint64
	retired uint64
	memory  *systemmemory.RAM
}

func (c *proofCPU) Registers() []uint64                           { return c.state[:] }
func (c *proofCPU) Retirement() *uint64                           { return &c.retired }
func (c *proofCPU) MemoryBus() engine.Memory                      { return c.memory }
func (c *proofCPU) Step() error                                   { return fmt.Errorf("unexpected architectural fallback") }
func (c *proofCPU) ExecuteInstruction(i engine.Instruction) error { return c.Step() }
func TestDerivedRegionRevocationKeepsUnmodifiedBaseNative(t *testing.T) {
	if !emu.NativeSessionsAvailable() {
		t.Skip("foreign sessions unavailable")
	}
	data := make([]byte, 8192)
	data[0] = 1
	data[4096] = 2
	m := systemmemory.New(data, func() bool { return true }, func(a uint64, x bool) (uint64, bool, error) { return a, true, nil })
	m.EnableOwnedMemory()
	m.EnableCodeProofs()
	m.ReadGuest = func(a uint64, n int, x bool) (uint64, error) { return uint64(data[a]), nil }
	m.WriteGuest = func(a uint64, n int, v uint64) error { m.BeforeWrite(a, n); data[a] = byte(v); return nil }
	arch := engine.Architecture{Name: "proof-region", StateWords: 2, PC: 1, Alignment: 1}
	arch.Fetch = func(bus engine.Memory, pc uint64) (engine.Instruction, error) {
		value, err := bus.Read(pc, 1, true)
		next := uint64(4096)
		if pc == 4096 {
			next = 0
		}
		return engine.Instruction{PC: pc, Length: 1, Bits: value, Terminal: true, Flow: engine.Successors{Known: true, Next: next}}, err
	}
	arch.Lower = func(b *emu.Builder, i engine.Instruction, completed int) bool {
		b.Store(0, b.Binary(emu.Add, b.Load(0), b.Constant(i.Bits)))
		b.Store(1, b.Constant(i.Flow.Next))
		return true
	}
	cfg := engine.DefaultEngineConfig()
	cfg.DynamicCode = true
	cfg.NaturalBlocks = true
	cfg.MaxBlocks = 0
	cfg.IRThreshold = 1
	cfg.NativeThreshold = 1
	cfg.MaxNativeInstructions = 4096
	e, err := engine.New(cfg, arch)
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	c := &proofCPU{memory: m}
	if err = e.Step(c, 1); err != nil {
		t.Fatal(err)
	}
	if err = e.Step(c, 1); err != nil {
		t.Fatal(err)
	}
	if err = e.Run(c, 20); err != nil {
		t.Fatal(err)
	}
	if c.state != [2]uint64{33, 0} || e.Stats.Regions == 0 {
		t.Fatal("region did not execute", c.state, e.Stats)
	}
	promotions := e.Stats.Promotions
	m.BeforeWrite(4096, 1)
	data[4096] = 3
	if m.NativeContext().LookupLink(0) != nil {
		t.Fatal("derived native publication survived changed child")
	}
	if err = e.Step(c, 1); err != nil {
		t.Fatal(err)
	}
	if c.state != [2]uint64{34, 4096} || e.Stats.Promotions != promotions || e.Stats.Invalidations != 0 {
		t.Fatal("unmodified base was discarded/recompiled", c.state, e.Stats)
	}
}
