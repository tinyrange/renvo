package engine

import "testing"

func TestProfiledTracePrefersHotShortPath(t *testing.T) {
	stamp := CodeStamp{Identity: new(CodeIdentity), Epoch: 1}
	c := stubCPU{memory: &regionDiscoveryMemory{stamp: stamp}}
	makeBlock := func(pc uint64, length int, flow Successors) *compiledBlock {
		ins := make([]Instruction, length)
		ins[length-1] = Instruction{PC: pc, Length: 2, Flow: flow}
		return &compiledBlock{native: true, instructions: ins, context: stamp, dependencies: []codeDependency{{0, stamp}}}
	}
	first := makeBlock(0, 1, Successors{Known: true, Conditional: true, Next: 4, Alternate: 8})
	e := Engine{arch: stubArchitecture(), config: DefaultEngineConfig(), blocks: map[uint64]*compiledBlock{
		4: makeBlock(4, 8, Successors{}), 8: makeBlock(8, 1, Successors{}),
	}}
	if parts := e.regionParts(&c, first); len(parts) != 2 || parts[1].pc != 4 {
		t.Fatal("unprofiled long path", parts)
	}
	for i := 0; i < 100; i++ {
		e.observeEdge(first.instructions[0], 8)
	}
	if parts := e.regionParts(&c, first); len(parts) != 2 || parts[1].pc != 8 {
		t.Fatal("hot short path", parts)
	}
	e.blocks[8].context.Epoch++
	if parts := e.regionParts(&c, first); len(parts) != 2 || parts[1].pc != 4 {
		t.Fatal("profile bypassed context guard", parts)
	}
}

func TestEdgeObservationsBoundedAndLegal(t *testing.T) {
	e := Engine{config: DefaultEngineConfig()}
	e.config.MaxBlocks = 1
	ins := Instruction{PC: 10, Flow: Successors{Known: true, Conditional: true, Next: 20, Alternate: 30}}
	e.observeEdge(ins, 99)
	if len(e.edgeCounts) != 0 {
		t.Fatal("illegal successor counted")
	}
	e.observeEdge(ins, 20)
	e.observeEdge(ins, 30)
	if e.edgeCounts[10] != [2]uint64{1, 1} {
		t.Fatal(e.edgeCounts)
	}
	other := ins
	other.PC = 11
	e.observeEdge(other, 20)
	if len(e.edgeCounts) != 1 {
		t.Fatal("profile table exceeded cap")
	}
	for i := 0; i < 70000; i++ {
		e.observeEdge(ins, 30)
	}
	if e.edgeCounts[10] != [2]uint64{1, 65535} {
		t.Fatal("counter saturation", e.edgeCounts)
	}
}
