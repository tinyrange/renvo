//go:build !renvo

package engine

import (
	emu "renvo.dev/internal/rfe/runtime"
	"testing"
)

func TestWideRegionUpgradeRetainsBaseForSmallerBudget(t *testing.T) {
	if !emu.NativeSessionsAvailable() {
		t.Skip("wide foreign sessions unavailable on this host")
	}
	m := new(ownedIndirectMemory)
	for i := 0; i < 300; i++ {
		m.Write(uint64(i*2), 2, 0x101)
	}
	m.Write(600, 2, 3)
	m.Write(602, 2, 3)
	arch := stubArchitecture()
	fetch := arch.Fetch
	arch.Fetch = func(m Memory, pc uint64) (Instruction, error) {
		i, err := fetch(m, pc)
		if err == nil && i.Terminal {
			target := uint64(0)
			if pc == 600 {
				target = 602
			}
			i.Flow.Next = target
			i.Bits = target<<8 | 3
		}
		return i, err
	}
	cfg := DefaultEngineConfig()
	cfg.NaturalBlocks = true
	cfg.DynamicCode = true
	cfg.MaxBlocks = 0
	cfg.IRThreshold, cfg.NativeThreshold = 1, 1
	cfg.MaxNativeInstructions = 4096
	e, err := New(cfg, arch)
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	c := &stubCPU{memory: m}
	if err = e.Step(c, 301); err != nil {
		t.Fatal(err)
	}
	if err = e.Step(c, 1); err != nil {
		t.Fatal(err)
	}
	first := e.blocks[0]
	base := first.entry
	if err = e.Run(c, 906); err != nil {
		t.Fatal(err)
	}
	if !first.region || first.regionInstructions != 302 || first.regionEntry == base {
		t.Fatal("no wide upgrade", first.regionInstructions)
	}
	if c.state != [2]uint64{1200, 0} {
		t.Fatal("fused state", c.state)
	}
	// Step drains deferred retirements, then cannot afford the fused body. The
	// immutable base entry must still be admissible, not released as an old region.
	if err = e.Step(c, 301); err != nil {
		t.Fatal("base entry retired by region upgrade", err)
	}
	if c.state != [2]uint64{1500, 602} {
		t.Fatal("smaller-budget base state", c.state)
	}
}
