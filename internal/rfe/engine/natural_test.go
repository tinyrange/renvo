//go:build !renvo

package engine

import (
	emu "renvo.dev/internal/rfe/runtime"
	"testing"
)

func TestNaturalBlockAndRegionWidthsExceedLegacyLimits(t *testing.T) {
	if !emu.NativeSessionsAvailable() {
		t.Skip("wide foreign sessions unavailable on this host")
	}
	m := new(ownedIndirectMemory)
	for i := 0; i < 300; i++ {
		m.Write(uint64(i*2), 2, 0x101)
	}
	m.Write(600, 2, 3)
	cfg := DefaultEngineConfig()
	cfg.NaturalBlocks = true
	cfg.MaxInstructions = 0
	cfg.MaxRegionInstructions = 0
	cfg.IRThreshold, cfg.NativeThreshold = 1, 1
	cfg.MaxNativeInstructions = 4096
	e, err := New(cfg, stubArchitecture())
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	c := &stubCPU{memory: m}
	if err = e.Step(c, 301); err != nil {
		t.Fatal(err)
	}
	if c.state != [2]uint64{300, 0} || c.retired != 301 || e.Stats.Native != 301 || len(e.blocks[0].instructions) != 301 {
		t.Fatal("wide block was split or lost progress", c.state, c.retired, e.Stats)
	}
	if err = e.Run(c, 903); err != nil {
		t.Fatal(err)
	}
	if c.state[0] != 1200 || c.retired != 1204 {
		t.Fatal("wide dispatch lost precise budget", c.state, c.retired)
	}
	// Independently exercise fusion of more than 256 separate resident blocks.
	e.blocks = make(map[uint64]*compiledBlock)
	clear(e.dispatch[:])
	stamp := m.CodeContext()
	for i := 0; i < 300; i++ {
		pc, next := uint64(i*2), uint64((i+1)%300*2)
		e.blocks[pc] = &compiledBlock{native: true, instructions: []Instruction{{PC: pc, Length: 2, Bits: 0x101, Flow: Successors{Known: true, Next: next}}}, context: stamp, dependencies: []codeDependency{{0, stamp}}, visits: 100}
	}
	c.state = [2]uint64{}
	first := e.blocks[0]
	e.prepareRegion(c, first)
	if !first.region || first.regionInstructions != 300 {
		t.Fatal("resident trace retained region/search ceiling", first.regionInstructions, e.Stats)
	}
	before := c.retired
	if err = e.runLinked(c, 900, first, m, m.NativeContext(), stamp, 1); err != nil {
		t.Fatal(err)
	}
	if c.retired-before != 900 || c.state != [2]uint64{900, 0} {
		t.Fatal("wide fused trace mismatch", c.state, c.retired-before)
	}
}

func TestAdaptiveColdOnePassAndActualHotReuse(t *testing.T) {
	m := new(ownedIndirectMemory)
	for i := 0; i < 1000; i++ {
		m.Write(uint64(i*2), 2, 0x101)
	}
	m.Write(2000, 2, 3)
	cfg := DefaultEngineConfig()
	cfg.Adaptive = true
	cfg.DynamicCode = true
	cfg.MaxBlocks = 0
	cfg.IRThreshold, cfg.NativeThreshold, cfg.RegionThreshold = 0, 0, 0
	cfg.MaxInstructions = 1
	cfg.MaxNativeInstructions = 4096
	e, err := New(cfg, stubArchitecture())
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	c := &stubCPU{memory: m}
	for i := 0; i < 1000; i++ {
		if err = e.Run(c, 1); err != nil {
			t.Fatal(err)
		}
	}
	if e.Stats.Promotions != 0 || e.CachedBlocks() != 0 {
		t.Fatal("cold one-pass code compiled", e.Stats)
	}
	c.state[1] = 2000
	for i := 0; i < 10000 && e.Stats.Native == 0; i++ {
		if err = e.Run(c, 4096); err != nil {
			t.Fatal(err)
		}
	}
	if e.Stats.Native == 0 || e.Stats.CompileFailures != 0 {
		t.Fatal("real reuse did not amortize compilation", e.Stats)
	}
}

func TestAdaptiveLinksProfitableOneShotSuccessorsWithoutLoopPolicy(t *testing.T) {
	m := new(ownedIndirectMemory)
	m.Write(0, 2, 0x403)
	m.Write(4, 2, 0x803)
	arch := stubArchitecture()
	fetch := arch.Fetch
	arch.Fetch = func(m Memory, pc uint64) (Instruction, error) {
		i, err := fetch(m, pc)
		i.Flow.Known = false
		return i, err
	}
	cfg := DefaultEngineConfig()
	cfg.Adaptive = true
	cfg.DynamicCode = true
	cfg.NaturalBlocks = true
	cfg.MaxBlocks = 0
	cfg.IRThreshold, cfg.NativeThreshold, cfg.RegionThreshold = 0, 0, 0
	cfg.MaxNativeInstructions = 4096
	e, err := New(cfg, arch)
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	// Test publication policy with a controlled learned cost, rather than
	// depending on a compiler wall-time outlier or this host's timer resolution.
	e.payback.interpretedNS, e.payback.interpreted = 1000000, 1
	c := &stubCPU{memory: m}
	for i := 0; i < 10000; i++ {
		c.state[1] = 0
		if err = e.Step(c, 1); err != nil {
			t.Fatal(err)
		}
		c.state[1] = 4
		if err = e.Step(c, 1); err != nil {
			t.Fatal(err)
		}
		if e.blocks[0] != nil && e.blocks[4] != nil && e.blocks[0].native && e.blocks[4].native {
			break
		}
	}
	if e.blocks[0] == nil || !e.blocks[0].native || e.blocks[4] == nil || !e.blocks[4].native {
		t.Fatal("reused one-shot leaves not native")
	}
	c.state[1] = 0
	before := c.retired
	sessions := e.Stats.NativeSessions
	if err = e.Run(c, 2); err != nil {
		t.Fatal(err)
	}
	if c.state[1] != 8 || c.retired-before != 2 || e.Stats.NativeSessions == sessions {
		t.Fatal("one-shot successor was excluded from linking", c.state, c.retired-before, e.Stats)
	}
}
