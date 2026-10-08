package engine

import "testing"

func TestGuestAlignmentRemainsArchitectureOwned(t *testing.T) {
	for _, mode := range []string{"interpreter", "ir", "native"} {
		m := new(stubMemory)
		cfg := DefaultEngineConfig()
		cfg.Mode = mode
		cfg.IRThreshold = 1
		cfg.NativeThreshold = 1
		e, err := New(cfg, stubArchitecture())
		if err != nil {
			t.Fatal(err)
		}
		c := stubCPU{memory: m, state: [2]uint64{7, 1}}
		if err = e.Step(&c, 1); err == nil || c.retired != 0 || c.state != [2]uint64{7, 1} || e.CachedBlocks() != 0 {
			t.Fatal("unaligned guest executed", mode, c.state, err)
		}
		e.Close()
	}
}
