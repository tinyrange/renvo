//go:build !renvo

package engine

import (
	"reflect"
	emu "renvo.dev/internal/rfe/runtime"
	"renvo.dev/internal/rfenativebridge"
	"testing"
)

func TestPureTriangleRegionExactPathsAndBounds(t *testing.T) {
	if !emu.NativeVariablePureLoopsAvailable() {
		t.Skip("variable lowering unavailable")
	}
	n, err := emu.NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(2, 1); err != nil {
		t.Fatal(err)
	}
	stamp := CodeStamp{Identity: new(CodeIdentity), Epoch: 1}
	memory := &regionDiscoveryMemory{stamp: stamp}
	c := stubCPU{memory: memory}
	makeBlock := func(pc uint64, flow Successors) *compiledBlock {
		return &compiledBlock{native: true, context: stamp, dependencies: []codeDependency{{pc, stamp}}, instructions: []Instruction{{PC: pc, Length: 2, Terminal: true, Flow: flow}}}
	}
	first := makeBlock(0, Successors{Known: true, Conditional: true, Next: 2, Alternate: 4})
	arm := makeBlock(2, Successors{Known: true, Next: 4})
	join := makeBlock(4, Successors{Known: true, Next: 0})
	arch := stubArchitecture()
	arch.Lower = func(b *emu.Builder, ins Instruction, completed int) bool {
		switch ins.PC {
		case 0:
			b.Store(1, b.Choose(b.Binary(emu.And, b.Load(0), b.Constant(1)), b.Constant(2), b.Constant(4)))
		case 2:
			b.Store(0, b.Binary(emu.Xor, b.Load(0), b.Constant(256)))
			b.Store(1, b.Constant(4))
		case 4:
			b.Store(0, b.Binary(emu.Add, b.Load(0), b.Constant(1)))
			b.Store(1, b.Constant(0))
		}
		return true
	}
	cfg := DefaultEngineConfig()
	cfg.MaxRegionInstructions = 2
	e := Engine{arch: arch, config: cfg, native: n, blocks: map[uint64]*compiledBlock{0: first, 2: arm, 4: join}}
	if e.preparePureDiamondRegion(&c, first) {
		t.Fatal("exceeded static region cap")
	}
	e.config.MaxRegionInstructions = 3
	arm.instructions[0].Memory = true
	if e.preparePureDiamondRegion(&c, first) {
		t.Fatal("speculated a memory arm")
	}
	arm.instructions[0].Memory = false
	saved := arm.instructions[0].Flow
	arm.instructions[0].Flow.Next = 8
	if e.preparePureDiamondRegion(&c, first) {
		t.Fatal("invented a join")
	}
	arm.instructions[0].Flow = saved
	if !e.preparePureDiamondRegion(&c, first) || !first.regionDiamond || first.regionInstructions != 3 || e.Stats.CompileFailures != 0 {
		t.Fatal("pure triangle not formed", first, e.Stats)
	}
	if len(first.dependencies) != 3 {
		t.Fatal("lost constituent code dependencies", first.dependencies)
	}
	m := new(emu.MemoryContext)
	m.ClaimLinks(n)
	m.PublishLink(0, first.regionEntry, 3, [17]uint8{})
	for start := uint64(0); start < 4; start++ {
		for budget := uint64(1); budget <= 64; budget++ {
			state := []uint64{start, 0}
			want, retired, loops := start, uint64(0), uint64(0)
			for retired+3 <= budget {
				retired += 2
				loops++
				if want&1 != 0 {
					want ^= 256
					retired++
				}
				want++
			}
			if err = n.CallLinked(state, m, budget); err != nil || state[0] != want || state[1] != 0 || m.Total != retired || m.Remaining != budget-retired || m.Status != 0 || m.LoopIterations != loops {
				t.Fatal("triangle scalar paths", start, budget, err, state, want, m.Total, retired, m.Remaining, m.Status, m.LoopIterations, loops)
			}
		}
	}
	// Inconsistent decoder metadata cannot refine an arbitrary lowered PC.
	arch = e.arch
	lower := e.arch.Lower
	e.arch.Lower = func(b *emu.Builder, ins Instruction, completed int) bool {
		if !lower(b, ins, completed) {
			return false
		}
		if ins.PC == 0 {
			b.Store(1, b.Constant(8))
		}
		return true
	}
	if e.preparePureDiamondRegion(&c, first) {
		t.Fatal("unproven lowered successor admitted")
	}
	e.arch = arch
	// Cold discovery often spans the join, instead of ending the optional
	// block there. Its exact pre-join prefix must still be recognized once.
	arm.instructions = append(arm.instructions, join.instructions...)
	if !e.preparePureDiamondRegion(&c, first) {
		t.Fatal("overlapping discovered join was not recognized")
	}
}

// The entry is the loop's join branch, not its arithmetic head. Its exit must
// retire only the entry instruction, despite speculative evaluation of pure
// continuation expressions. The scalar model counts every chosen instruction.
func TestPureJoinEntryPreciseExitAndOptionalPaths(t *testing.T) {
	if !emu.NativeVariablePureLoopsAvailable() {
		t.Skip("variable lowering unavailable")
	}
	n, err := emu.NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	const words = 5
	if err = n.PrepareLinks(words, 4); err != nil {
		t.Fatal(err)
	}
	stamp := CodeStamp{Identity: new(CodeIdentity), Epoch: 1}
	memory := &regionDiscoveryMemory{stamp: stamp}
	c := stubCPU{memory: memory}
	makeBlock := func(pc uint64, flow Successors) *compiledBlock {
		return &compiledBlock{native: true, context: stamp, dependencies: []codeDependency{{pc, stamp}}, instructions: []Instruction{{PC: pc, Length: 2, Terminal: true, Flow: flow}}}
	}
	common := makeBlock(0, Successors{Known: true, Conditional: true, Next: 2, Alternate: 4})
	arm := makeBlock(2, Successors{Known: true, Next: 4})
	first := makeBlock(4, Successors{Known: true, Conditional: true, Next: 0, Alternate: 6})
	arch := stubArchitecture()
	arch.StateWords, arch.PC = words, 4
	arch.Lower = func(b *emu.Builder, ins Instruction, completed int) bool {
		switch ins.PC {
		case 4:
			b.Store(4, b.Choose(b.Binary(emu.Less, b.Load(0), b.Load(3)), b.Constant(0), b.Constant(6)))
		case 0:
			b.Store(0, b.Binary(emu.Add, b.Load(0), b.Constant(1)))
			b.Store(2, b.Binary(emu.And, b.Load(0), b.Constant(1)))
			b.Store(4, b.Choose(b.Load(2), b.Constant(2), b.Constant(4)))
		case 2:
			b.Store(1, b.Binary(emu.Add, b.Load(1), b.Load(0)))
			b.Store(4, b.Constant(4))
		}
		return true
	}
	cfg := DefaultEngineConfig()
	cfg.MaxRegionInstructions = 3
	e := Engine{arch: arch, config: cfg, native: n, blocks: map[uint64]*compiledBlock{0: common, 2: arm, 4: first}}
	c.state[1] = 4 // stubCPU has its own two-word state; expose the target PC below.
	cpu := &pureJoinCPU{stubCPU: c, state: []uint64{0, 0, 0, 0, 4}}
	if !e.preparePureDiamondRegion(cpu, first) || !first.regionDiamond || first.regionInstructions != 3 || len(first.dependencies) != 3 {
		t.Fatal("hot join not fused", first, e.Stats)
	}
	m := new(emu.MemoryContext)
	epoch := uint64(1)
	m.Clock = &epoch
	m.ClaimLinks(n)
	m.PublishLink(4, first.regionEntry, 3, [17]uint8{})
	if err = n.PrepareTargetLink(4, first.regionEntry, words, 3); err != nil {
		t.Fatal(err)
	}
	backing := make([]byte, 4096)
	for _, start := range []uint64{0, 1, 2, 7} {
		for _, limit := range []uint64{0, 1, 3, 8, 100} {
			for budget := uint64(1); budget <= 64; budget++ {
				state := []uint64{start, 19, 0xface, limit, 4}
				want := append([]uint64(nil), state...)
				var retired, loops uint64
				for retired+3 <= budget && want[4] == 4 {
					retired++
					if want[0] >= want[3] {
						want[4] = 6
						break
					}
					loops++
					want[0]++
					want[2] = want[0] & 1
					retired++
					if want[2] != 0 {
						want[1] += want[0]
						retired++
					}
				}
				for _, foreign := range []bool{false, true} {
					copy(state, []uint64{start, 19, 0xface, limit, 4})
					if foreign {
						hit, e := n.RunLinkedBackingSessionUntil(state, m, budget, backing, rfenativebridge.MonotonicNS()+1000000000)
						err = e
						if hit {
							t.Fatal("unexpected expiry")
						}
					} else {
						err = n.CallLinked(state, m, budget)
					}
					if err != nil || !reflect.DeepEqual(state, want) || m.Total != retired || m.Remaining != budget-retired || m.Status != 0 || m.LoopIterations != loops {
						t.Fatal("join scalar", foreign, start, limit, budget, err, state, want, m.Total, retired, m.LoopIterations, loops)
					}
				}
			}
		}
	}
	// Cold discovery may overlap the join. Only its optional prefix executes.
	arm.instructions = append(arm.instructions, first.instructions...)
	if !e.preparePureDiamondRegion(cpu, first) {
		t.Fatal("overlapping hot join rejected")
	}
	arm.instructions[0].ScheduleBoundary = true
	if e.preparePureDiamondRegion(cpu, first) {
		t.Fatal("speculated scheduling boundary")
	}
	arm.instructions[0].ScheduleBoundary = false
	lower := e.arch.Lower
	e.arch.Lower = func(b *emu.Builder, ins Instruction, completed int) bool {
		if !lower(b, ins, completed) {
			return false
		}
		if ins.PC == 4 {
			b.Store(4, b.Constant(10))
		}
		return true
	}
	if e.preparePureDiamondRegion(cpu, first) {
		t.Fatal("decoder-only branch proof admitted")
	}
}

type pureJoinCPU struct {
	stubCPU
	state []uint64
}

func (c *pureJoinCPU) Registers() []uint64 { return c.state }
