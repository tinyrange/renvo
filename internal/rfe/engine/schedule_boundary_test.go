package engine

import "testing"

func TestRegionDiscoveryCannotInlineSchedulingBoundary(t *testing.T) {
	stamp := CodeStamp{Identity: new(CodeIdentity), Epoch: 1}
	memory := &regionDiscoveryMemory{stamp: stamp}
	cpu := stubCPU{memory: memory}
	first := &compiledBlock{native: true, instructions: []Instruction{{PC: 0, Length: 2, Flow: Successors{Known: true, Next: 2}}}, context: stamp, dependencies: []codeDependency{{0, stamp}}}
	boundary := &compiledBlock{native: true, instructions: []Instruction{{PC: 2, Length: 2, ScheduleBoundary: true, Flow: Successors{Known: true, Next: 0}}}, context: stamp, dependencies: []codeDependency{{0, stamp}}}
	e := Engine{arch: stubArchitecture(), config: DefaultEngineConfig(), blocks: map[uint64]*compiledBlock{2: boundary}}
	if len(e.regionParts(&cpu, first)) != 0 {
		t.Fatal("trace crossed scheduling boundary")
	}
	cpu.state[1] = 2
	if len(e.regionParts(&cpu, boundary)) != 0 {
		t.Fatal("scheduling boundary entered looping region")
	}
}
