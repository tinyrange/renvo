package engine

import "testing"

type regionDiscoveryMemory struct {
	stubMemory
	stamp CodeStamp
}

func (m *regionDiscoveryMemory) CodeContext() CodeStamp                { return m.stamp }
func (m *regionDiscoveryMemory) CodeVersion(uint64) (CodeStamp, error) { return m.stamp, nil }

func TestRegionDiscoveryAlternativeCycleAndAcyclicBounds(t *testing.T) {
	stamp := CodeStamp{Identity: new(CodeIdentity), Epoch: 1}
	memory := &regionDiscoveryMemory{stamp: stamp}
	c := stubCPU{memory: memory}
	makeBlock := func(pc, next uint64, alternate uint64, conditional, known bool) *compiledBlock {
		return &compiledBlock{native: true, instructions: []Instruction{{PC: pc, Length: 2, Flow: Successors{Next: next, Alternate: alternate, Conditional: conditional, Known: known}}}, context: stamp, dependencies: []codeDependency{{0, stamp}}}
	}
	e := Engine{arch: stubArchitecture(), config: DefaultEngineConfig(), blocks: map[uint64]*compiledBlock{}}
	// The first (backward) edge enters a different cycle. The alternative edge
	// returns to this root and must not be hidden by the failed first search.
	c.state[1] = 16
	first := makeBlock(16, 0, 20, true, true)        // B.NE 0
	e.blocks[0] = makeBlock(0, 0, 0, false, true)    // B 0
	e.blocks[20] = makeBlock(20, 16, 0, false, true) // B 16
	parts := e.regionParts(&c, first)
	if len(parts) != 2 || parts[0].pc != 16 || parts[1].pc != 20 {
		t.Fatalf("alternative cycle lost: %+v", parts)
	}
	// A noncyclic path is still useful, but it must stay inside the static cap.
	c.state[1] = 0
	first = makeBlock(0, 4, 0, false, true)
	e.blocks = map[uint64]*compiledBlock{4: makeBlock(4, 8, 0, false, true), 8: makeBlock(8, 0, 0, false, false)}
	parts = e.regionParts(&c, first)
	if len(parts) != 3 || parts[2].pc != 8 {
		t.Fatalf("acyclic path missing: %+v", parts)
	}
	e.config.MaxRegionInstructions = 2
	parts = e.regionParts(&c, first)
	if len(parts) != 2 || parts[1].pc != 4 {
		t.Fatalf("static trace bound violated: %+v", parts)
	}
	e.blocks[4].context.Epoch++
	if len(e.regionParts(&c, first)) != 0 {
		t.Fatal("cross-context candidate admitted")
	}
	e.blocks[4].context = stamp
	e.blocks[4].dependencies = nil
	if len(e.regionParts(&c, first)) != 0 {
		t.Fatal("unvalidated dependency admitted")
	}
	e.blocks[4].dependencies = []codeDependency{{0, stamp}}
	e.blocks[4].native = false
	if len(e.regionParts(&c, first)) != 0 {
		t.Fatal("uncompiled candidate admitted")
	}
}
