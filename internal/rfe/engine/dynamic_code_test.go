package engine

import (
	"encoding/binary"
	"testing"
)

func TestDynamicCodeKeepsWorkingSetAcrossSmallInitialArena(t *testing.T) {
	m := new(stubMemory)
	for i := 0; i < 500; i++ {
		binary.LittleEndian.PutUint16(m.bytes[i*2:], 0x101)
	}
	cfg := DefaultEngineConfig()
	cfg.DynamicCode = true
	cfg.EvictCold = true
	cfg.CacheContexts = true
	cfg.MaxBlocks = 0
	cfg.MaxInstructions = 1
	cfg.NativeBytes = 4096
	cfg.IRThreshold = 1
	cfg.NativeThreshold = 1
	e, err := New(cfg, stubArchitecture())
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	c := &stubCPU{memory: m}
	for pass := 0; pass < 2; pass++ {
		for i := 0; i < 500; i++ {
			c.state[1] = uint64(i * 2)
			if err = e.Step(c, 1); err != nil {
				t.Fatal(err)
			}
		}
	}
	if e.CachedBlocks() != 500 || e.Stats.Promotions != 500 || e.Stats.Native != 1000 || e.Stats.ArenaResets != 0 || c.state[0] != 1000 {
		t.Fatal("live working set discarded", e.CachedBlocks(), e.Stats, c.state)
	}
}
