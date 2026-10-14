package engine

import (
	"encoding/binary"
	"testing"
)

func TestEvictedWorkingSetMustReestablishDiscoveryHeat(t *testing.T) {
	memory := new(stubMemory)
	binary.LittleEndian.PutUint16(memory.bytes[:], 0x101)
	binary.LittleEndian.PutUint16(memory.bytes[8:], 0x201)
	config := DefaultEngineConfig()
	config.Mode, config.EvictCold = "ir", true
	config.MaxBlocks, config.MaxInstructions = 1, 1
	config.DiscoveryThreshold = 4
	config.IRThreshold, config.NativeThreshold = 1, 1
	e, err := New(config, stubArchitecture())
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	c := stubCPU{memory: memory}
	// Two hot instructions exceed a one-block cache. Each eviction must
	// demand fresh observations; old saturated counters cannot decode every miss.
	for phase := 0; phase < 4; phase++ {
		for i := 0; i < 4; i++ {
			c.state[1] = uint64((phase % 2) * 8)
			if err = e.Step(&c, 1); err != nil {
				t.Fatal(err)
			}
		}
	}
	if c.state[0] != 24 || c.retired != 16 || e.Stats.Blocks != 4 || e.Stats.Evictions != 3 || e.CachedBlocks() != 1 {
		t.Fatalf("evicted heat caused churn or incorrect execution: %+v state=%v", e.Stats, c.state)
	}
}
