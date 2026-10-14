package engine

import (
	"encoding/binary"
	"testing"
)

func TestBoundedDecodedAdmissionAfterColdStartup(t *testing.T) {
	for _, mode := range []string{"ir", "native"} {
		memory := new(stubMemory)
		for i := 0; i < 3; i++ {
			binary.LittleEndian.PutUint16(memory.bytes[i*2:], uint16((i+1)<<8|1))
		}
		config := DefaultEngineConfig()
		config.Mode, config.EvictCold = mode, true
		config.MaxBlocks, config.MaxInstructions = 2, 1
		config.IRThreshold, config.NativeThreshold = 1, 1
		e, err := New(config, stubArchitecture())
		if err != nil {
			t.Fatal(err)
		}
		c := stubCPU{memory: memory}
		for i := 0; i < 12; i++ {
			c.state[1] = uint64(i % 3 * 2)
			if err = e.Step(&c, 1); err != nil {
				t.Fatal(err)
			}
			if e.CachedBlocks() > 2 {
				t.Fatal("decoded admission exceeded bound")
			}
		}
		if c.state[0] != 24 || c.retired != 12 || e.Stats.Evictions != 10 || e.Stats.Blocks != 12 {
			t.Fatalf("cold startup permanently denied new translations: %+v state=%v", e.Stats, c.state)
		}
		if mode == "native" && e.NativeAvailable() && e.Stats.Native != 12 {
			t.Fatal("late blocks never reached native tier")
		}
		// An evicted direct-cache slot must not resurrect an old instruction.
		binary.LittleEndian.PutUint16(memory.bytes[:], 0x701)
		c.state[1] = 0
		before := c.state[0]
		if err = e.Step(&c, 1); err != nil || c.state[0] != before+7 {
			t.Fatal("stale direct dispatch entry after eviction", err)
		}
		if e.Stats.NativeBytes > config.NativeBytes {
			t.Fatal("eviction expanded native resource limit")
		}
		e.Close()
	}
}
