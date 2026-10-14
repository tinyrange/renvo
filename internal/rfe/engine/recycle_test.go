package engine

import (
	"encoding/binary"
	"testing"
)

func TestQuiescentArenaRecyclingKeepsByteCapAndSemantics(t *testing.T) {
	memory := new(stubMemory)
	for i := 0; i < 200; i++ {
		binary.LittleEndian.PutUint16(memory.bytes[i*2:], 0x101)
	}
	config := DefaultEngineConfig()
	config.RecycleNative, config.EvictCold = true, true
	config.MaxBlocks, config.MaxInstructions = 8, 1
	config.IRThreshold, config.NativeThreshold = 1, 1
	config.NativeBytes = 4096
	e, err := New(config, stubArchitecture())
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	if !e.NativeAvailable() {
		t.Skip("native host unavailable")
	}
	c := stubCPU{memory: memory}
	for i := 0; i < 200; i++ {
		c.state[1] = uint64(i * 2)
		if err = e.RunQuanta(&c, 1); err != nil {
			t.Fatal(err)
		}
		if c.state[0] != uint64(i+1) || c.retired != uint64(i+1) || e.Stats.NativeBytes > config.NativeBytes || e.CachedBlocks() > config.MaxBlocks {
			t.Fatal("recycling changed progress or expanded resource limits")
		}
	}
	if e.Stats.ArenaResets == 0 || e.Stats.Native != 200 {
		t.Fatalf("arena exhaustion permanently disabled JIT: %+v", e.Stats)
	}
}
