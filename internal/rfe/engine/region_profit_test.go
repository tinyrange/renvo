package engine

import "testing"

func TestProfiledRegionColdCycleVersusHotTrace(t *testing.T) {
	stamp := CodeStamp{Identity: new(CodeIdentity), Epoch: 1}
	c := stubCPU{memory: &regionDiscoveryMemory{stamp: stamp}}
	block := func(pc uint64, flow Successors) *compiledBlock {
		return &compiledBlock{native: true, instructions: []Instruction{{PC: pc, Length: 2, Flow: flow}}, context: stamp, dependencies: []codeDependency{{0, stamp}}}
	}
	first := block(0, Successors{Known: true, Conditional: true, Next: 4, Alternate: 8})
	e := Engine{arch: stubArchitecture(), config: DefaultEngineConfig(), blocks: map[uint64]*compiledBlock{
		4:  block(4, Successors{Known: true, Next: 0}),
		8:  block(8, Successors{Known: true, Next: 12}),
		12: block(12, Successors{}),
	}}
	for _, trial := range []struct {
		name   string
		counts [2]uint64
		want   []uint64
	}{
		{"unobserved-cycle", [2]uint64{}, []uint64{0, 4}},
		{"cold-cycle", [2]uint64{1, 99}, []uint64{0, 8, 12}},
		{"hot-cycle", [2]uint64{99, 1}, []uint64{0, 4}},
	} {
		t.Run(trial.name, func(t *testing.T) {
			e.edgeCounts = map[uint64][2]uint64{0: trial.counts}
			parts := e.regionParts(&c, first)
			if len(parts) != len(trial.want) {
				t.Fatalf("parts=%v want PCs=%v", parts, trial.want)
			}
			for i, part := range parts {
				if part.pc != trial.want[i] {
					t.Fatalf("part %d PC=%d want=%d", i, part.pc, trial.want[i])
				}
			}
		})
	}
}

func TestProfiledRegionValuesTransitionsRatherThanInstructionVolume(t *testing.T) {
	stamp := CodeStamp{Identity: new(CodeIdentity), Epoch: 1}
	c := stubCPU{memory: &regionDiscoveryMemory{stamp: stamp}}
	block := func(pc uint64, length int, flow Successors) *compiledBlock {
		instructions := make([]Instruction, length)
		instructions[length-1] = Instruction{PC: pc, Length: 2, Flow: flow}
		return &compiledBlock{native: true, instructions: instructions, context: stamp, dependencies: []codeDependency{{0, stamp}}}
	}
	first := block(0, 1, Successors{Known: true, Conditional: true, Next: 4, Alternate: 8})
	e := Engine{arch: stubArchitecture(), config: DefaultEngineConfig(), edgeCounts: map[uint64][2]uint64{0: {40, 60}}, blocks: map[uint64]*compiledBlock{
		4:  block(4, 16, Successors{}),
		8:  block(8, 1, Successors{Known: true, Next: 12}),
		12: block(12, 1, Successors{}),
	}}
	// A long leaf is not sixteen dispatches saved. The more likely path
	// removes two transitions despite containing fewer guest instructions.
	parts := e.regionParts(&c, first)
	if len(parts) != 3 || parts[1].pc != 8 || parts[2].pc != 12 {
		t.Fatal("preferred instruction volume over hot transitions", parts)
	}
	e.config.MaxRegionInstructions = 2
	parts = e.regionParts(&c, first)
	if len(parts) != 2 || parts[1].pc != 8 {
		t.Fatal("profitability bypassed the static bound", parts)
	}
	e.config.MaxRegionInstructions = 256
	e.blocks[8].context.Epoch++
	parts = e.regionParts(&c, first)
	if len(parts) != 2 || parts[1].pc != 4 {
		t.Fatal("profitability bypassed code-context validation", parts)
	}
}
