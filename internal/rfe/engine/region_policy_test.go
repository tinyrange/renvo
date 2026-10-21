//go:build !renvo

package engine

import "testing"

func TestLoopOnlyPolicyLeavesHotUnknownSuccessorsOnOrdinaryNativePath(t *testing.T) {
	for _, loopOnly := range []bool{true, false} {
		m := new(ownedIndirectMemory)
		m.Write(0, 2, 0x403) // dynamically classified successor at 4
		arch := stubArchitecture()
		fetch := arch.Fetch
		arch.Fetch = func(memory Memory, pc uint64) (Instruction, error) {
			i, err := fetch(memory, pc)
			i.Flow.Known = false
			return i, err
		}
		cfg := DefaultEngineConfig()
		cfg.IRThreshold, cfg.NativeThreshold, cfg.RegionThreshold = 1, 1, 4
		cfg.MaxInstructions, cfg.MaxNativeInstructions = 1, 4096
		cfg.LoopRegionsOnly = loopOnly
		e, err := New(cfg, arch)
		if err != nil {
			t.Fatal(err)
		}
		if !e.NativeAvailable() {
			e.Close()
			t.Skip("native unavailable")
		}
		c := &stubCPU{memory: m}
		for n := 0; n < 16; n++ {
			c.state[1] = 0
			if err = e.Run(c, 4096); err != nil {
				e.Close()
				t.Fatal(err)
			}
			if c.state[1] != 4 || c.retired != uint64(n+1) {
				e.Close()
				t.Fatal("policy changed guest semantics", c.state, c.retired)
			}
		}
		if e.Stats.Native == 0 {
			t.Fatal("policy disabled ordinary native execution")
		}
		if loopOnly && (e.Stats.Regions != 0 || e.Stats.NativeSessions != 0) {
			t.Fatal("one-shot code paid session/compilation costs", e.Stats)
		}
		if !loopOnly && (e.Stats.Regions == 0 || e.Stats.NativeSessions == 0) {
			t.Fatal("opt-in policy changed general engine behavior", e.Stats)
		}
		e.Close()
	}
}
