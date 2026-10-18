//go:build !renvo

package engine

import (
	emu "renvo.dev/internal/rfe/runtime"
	"testing"
)

// Actual native sessions must not restart an already published countdown.
// A fresh publication after a context clear still initializes its normal hint.
func TestRegionDemandCountdownSurvivesMatchingNativeSessions(t *testing.T) {
	memory := new(ownedIndirectMemory)
	if err := memory.Write(0, 2, 3); err != nil {
		t.Fatal(err)
	}
	arch := stubArchitecture()
	fetch, lower := arch.Fetch, arch.Lower
	arch.Fetch = func(m Memory, pc uint64) (Instruction, error) {
		ins, err := fetch(m, pc)
		if err == nil && ins.Bits&255 == 3 {
			ins.Flow = Successors{}
			ins.Terminal = true
		}
		return ins, err
	}
	arch.Lower = func(b *emu.Builder, ins Instruction, completed int) bool {
		if ins.Bits&255 == 3 {
			b.Store(1, b.Load(0))
			return true
		}
		return lower(b, ins, completed)
	}
	cfg := DefaultEngineConfig()
	cfg.IRThreshold, cfg.NativeThreshold, cfg.RegionThreshold = 1, 1, 1
	cfg.MaxInstructions, cfg.MaxNativeInstructions = 1, 4096
	e, err := New(cfg, arch)
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	if !e.NativeAvailable() {
		t.Skip("native unavailable")
	}
	c := &indirectCPU{stubCPU: stubCPU{memory: memory}}
	for i := 0; i < 3; i++ {
		if err := e.RunQuanta(c, 64); err != nil {
			t.Fatal(err)
		}
	}
	block := e.blocks[0]
	if block == nil || !block.region {
		t.Fatal("owned region not prepared", e.Stats)
	}
	context := memory.NativeContext()
	link := context.LookupLink(0)
	if link == nil {
		t.Fatal("region not published")
	}
	entry, instructions := link.Entry, link.Instructions
	for _, hint := range []uint64{256, 17, 1} {
		link.Reserved = hint
		before := c.retired
		if err := e.RunQuanta(c, 64); err != nil {
			t.Fatal(err)
		}
		link = context.LookupLink(0)
		if c.retired-before != 64 || c.state[1] != 0 || link == nil || link.Entry != entry || link.Instructions != instructions || link.Reserved != hint {
			t.Fatal("matching session replaced hint or lost progress", hint, c.retired-before, link)
		}
	}
	context.ClearLinks()
	if err := e.RunQuanta(c, 64); err != nil {
		t.Fatal(err)
	}
	link = context.LookupLink(0)
	if link == nil || link.Entry != entry || link.Reserved != ^uint64(0) {
		t.Fatal("fresh native publication did not initialize ordinary hint", link)
	}
}
