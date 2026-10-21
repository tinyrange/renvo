//go:build !renvo

package engine

import "testing"

func TestRegionAdmissionKeepsColdNativeLeavesAndSpecializesHotLoop(t *testing.T) {
	m := new(ownedIndirectMemory)
	m.Write(0, 2, 3) // direct self branch
	cfg := DefaultEngineConfig()
	cfg.IRThreshold, cfg.NativeThreshold, cfg.RegionThreshold = 1, 1, 12
	cfg.MaxInstructions, cfg.MaxNativeInstructions = 1, 4096
	e, err := New(cfg, stubArchitecture())
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	if !e.NativeAvailable() {
		t.Skip("native unavailable")
	}
	c := &stubCPU{memory: m}
	for i := 0; i < 12; i++ {
		before := c.retired
		if err = e.Run(c, 4096); err != nil {
			t.Fatal(err)
		}
		if c.retired != before+1 || c.state[1] != 0 {
			t.Fatal("cold native leaf changed semantics", c.retired, c.state)
		}
		if e.Stats.NativeSessions != 0 || e.Stats.Regions != 0 {
			t.Fatal("cold code paid for a session/second compilation", e.Stats)
		}
	}
	if e.Stats.Native == 0 {
		t.Fatal("admission suppressed ordinary native compilation")
	}
	before := c.retired
	if err = e.Run(c, 4096); err != nil {
		t.Fatal(err)
	}
	if c.retired <= before+1 || c.retired > before+4096 || e.Stats.Regions == 0 {
		t.Fatal("hot loop never specialized", c.retired-before, e.Stats)
	}
}
