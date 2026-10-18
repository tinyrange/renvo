package engine

import "testing"

func TestDemandIndirectCallsRemainObservedOwnedAndBounded(t *testing.T) {
	stamp := CodeStamp{Identity: new(CodeIdentity), Epoch: 9}
	for _, mode := range []string{"mono", "unseen", "polymorphic", "cold-root", "context", "boundary", "dependency", "cap", "high-alias", "nested", "too-many-targets", "shared-direct-callee", "foreign-second"} {
		t.Run(mode, func(t *testing.T) {
			makeBlock := func(pc uint64, flow Successors) *compiledBlock {
				return &compiledBlock{native: true, context: stamp, dependencies: []codeDependency{{0, stamp}}, instructions: []Instruction{{PC: pc, Length: 4, Flow: flow}}}
			}
			root := makeBlock(0, Successors{IndirectCall: true, Continuation: 4})
			root.browserDemand = true
			root.linkSuccessorCount = 1
			root.linkSuccessors[0] = 1<<40 | 16
			target := makeBlock(1<<40|16, Successors{Return: true})
			cont := makeBlock(4, Successors{Known: true, Next: 0})
			e := Engine{config: DefaultEngineConfig(), blocks: map[uint64]*compiledBlock{0: root, 4: cont, 1<<40 | 16: target}}
			want := 3
			switch mode {
			case "unseen":
				root.linkSuccessorCount = 0
				want = 1
			case "polymorphic":
				root.linkSuccessorCount = 2
				root.linkSuccessors[1] = 32
				e.blocks[32] = makeBlock(32, Successors{Return: true})
				want = 4
			case "too-many-targets":
				root.linkSuccessorCount = 3
				want = 1
			case "shared-direct-callee":
				target.instructions[0].Flow = Successors{Call: true, Next: 32, Continuation: 20}
				e.blocks[32] = makeBlock(32, Successors{Return: true})
				e.blocks[20] = makeBlock(20, Successors{Call: true, Next: 32, Continuation: 24})
				e.blocks[24] = makeBlock(24, Successors{Known: true, Next: 4})
				want = 6
			case "foreign-second":
				root.linkSuccessorCount = 2
				root.linkSuccessors[1] = 32
				e.blocks[32] = makeBlock(32, Successors{Return: true})
				e.blocks[32].context.Epoch++
			case "cold-root":
				root.browserDemand = false
				want = 1
			case "context":
				target.context.Epoch++
				want = 2
			case "boundary":
				target.instructions[0].ScheduleBoundary = true
				want = 2
			case "dependency":
				target.dependencies = nil
				want = 2
			case "cap":
				e.config.MaxRegionInstructions = 2
				want = 2
			case "high-alias":
				root.linkSuccessors[0] = 16
				want = 2
			case "nested":
				target.instructions[0].Flow = Successors{IndirectCall: true, Continuation: 20}
				target.linkSuccessorCount = 1
				target.linkSuccessors[0] = 32
				e.blocks[32] = makeBlock(32, Successors{Return: true})
			}
			parts, count, _, ret, call := e.graphCallParts(0, root, stamp)
			if len(parts) != want || count != want || (call != 0) != (want > 1) || call != 0 && ret[0] != 4 {
				t.Fatal(len(parts), count, ret, call)
			}
			if mode == "shared-direct-callee" && (call != 3 || ret != [3]uint64{4, 20, 24}) {
				t.Fatal("missing bounded direct continuations", call, ret)
			}
			for _, part := range parts {
				if part.pc == 32 && mode != "polymorphic" && mode != "shared-direct-callee" {
					t.Fatal("nested call admitted")
				}
			}
		})
	}
}
