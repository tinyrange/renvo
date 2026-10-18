package engine

import "testing"

type regionLimitMemory struct {
	versionedStub
	contexts int
}

func (m *regionLimitMemory) CodeContext() CodeStamp {
	m.contexts++
	return m.versionedStub.CodeContext()
}

func TestFullRegionSkipsRediscoveryAfterUnrelatedPromotion(t *testing.T) {
	memory := new(regionLimitMemory)
	memory.Write(0, 2, 0x101)
	memory.Write(2, 2, 3)
	memory.Write(128, 2, 0x101)
	config := DefaultEngineConfig()
	config.MaxInstructions, config.MaxRegionInstructions = 1, 2
	config.IRThreshold, config.NativeThreshold = 1, 1
	e, err := New(config, stubArchitecture())
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	if !e.NativeAvailable() {
		t.Skip("native host unavailable")
	}
	c := stubCPU{memory: memory}
	for i := 0; i < 3; i++ {
		if err = e.RunQuanta(&c, 2); err != nil {
			t.Fatal(err)
		}
	}
	block := e.blocks[0]
	if block == nil || !block.region || block.regionInstructions != config.MaxRegionInstructions {
		t.Fatal("test did not establish a region at its fixed instruction cap")
	}
	entry, generation := block.regionEntry, block.regionGeneration
	c.state[1] = 128
	if err = e.RunQuanta(&c, 1); err != nil {
		t.Fatal(err)
	}
	if generation == e.Stats.Promotions {
		t.Fatal("test did not promote an unrelated leaf")
	}
	c.state[1] = 0
	if e.block(&c) != block {
		t.Fatal("region failed ordinary code-context validation")
	}
	checks, regions := memory.contexts, e.Stats.Regions
	e.prepareRegion(&c, block)
	if memory.contexts != checks || e.Stats.Regions != regions || block.regionEntry != entry || block.regionGeneration != e.Stats.Promotions {
		t.Fatal("full region unnecessarily searched or replaced its validated entry")
	}
	before, retired := c.state[0], c.retired
	if err = e.RunQuanta(&c, 6); err != nil {
		t.Fatal(err)
	}
	if c.state != [2]uint64{before + 3, 0} || c.retired != retired+6 {
		t.Fatal("retained region changed retirement or execution", c.state, c.retired)
	}
}
