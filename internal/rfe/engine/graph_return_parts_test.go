package engine

import "testing"

func TestDemandObservedReturnsDoNotGuessOrDecodeTargets(t *testing.T) {
	stamp := CodeStamp{Identity: new(CodeIdentity), Epoch: 39}
	for _, mode := range []string{"mono", "poly", "cold-root", "missing", "context", "dependency", "boundary", "cap", "high-alias", "too-many", "following-call"} {
		t.Run(mode, func(t *testing.T) {
			makeBlock := func(pc uint64, flow Successors) *compiledBlock {
				return &compiledBlock{native: true, context: stamp, dependencies: []codeDependency{{0, stamp}}, instructions: []Instruction{{PC: pc, Length: 4, Flow: flow}}}
			}
			root := makeBlock(0, Successors{Return: true})
			root.browserDemand, root.linkSuccessorCount = true, 1
			root.linkSuccessors[0] = 1<<40 | 16
			target := makeBlock(1<<40|16, Successors{Known: true, Next: 0})
			e := Engine{config: DefaultEngineConfig(), blocks: map[uint64]*compiledBlock{0: root, 1<<40 | 16: target}}
			want, wantCalls := 2, 0
			switch mode {
			case "poly":
				root.linkSuccessorCount = 2
				root.linkSuccessors[1] = 32
				e.blocks[32] = makeBlock(32, Successors{Known: true, Next: 0})
				want = 3
			case "cold-root":
				root.browserDemand = false
				want = 1
			case "missing":
				delete(e.blocks, 1<<40|16)
				want = 1
			case "context":
				target.context.Epoch++
				want = 1
			case "dependency":
				target.dependencies = nil
				want = 1
			case "boundary":
				target.instructions[0].ScheduleBoundary = true
				want = 1
			case "cap":
				e.config.MaxRegionInstructions = 1
				want = 1
			case "high-alias":
				root.linkSuccessors[0] = 16
				want = 1
			case "too-many":
				root.linkSuccessorCount = 3
				want = 1
			case "following-call":
				target.instructions[0].Flow = Successors{Call: true, Next: 64, Continuation: 20}
				e.blocks[64] = makeBlock(64, Successors{Return: true})
				e.blocks[20] = makeBlock(20, Successors{Known: true, Next: 0})
				want, wantCalls = 4, 1
			}
			parts, count, branched, calls, n := e.graphCallParts(0, root, stamp)
			if len(parts) != want || count != want || branched != (want > 1) || n != wantCalls {
				t.Fatal("bad observed return closure", len(parts), count, branched, n)
			}
			hints, hn := graphReturnSuccessors(root, parts, calls, n, root.browserDemand)
			if hn != wantCalls+want-1 && mode != "following-call" || mode == "following-call" && (hn != 2 || hints[1] != 1<<40|16) {
				t.Fatal("guessed or lost return", hints, hn)
			}
		})
	}
}

func TestGraphReturnHintsFitFourSuccessorsWithoutTruncatingPCs(t *testing.T) {
	high := uint64(1<<40 | 16)
	root := &compiledBlock{linkSuccessorCount: 2, linkSuccessors: [4]uint64{high, 32}}
	parts := []regionPart{{0, root}, {high, &compiledBlock{}}, {32, &compiledBlock{}}}
	out, n := graphReturnSuccessors(root, parts, [3]uint64{4, 20, 24}, 3, true)
	if n != 4 || out != [4]uint64{4, 20, 24, high} {
		t.Fatal("unbounded or truncated return", out, n)
	}
	parts[1].pc = 16
	out, n = graphReturnSuccessors(root, parts, [3]uint64{4, 20, 24}, 3, true)
	if n != 4 || out[3] != 32 {
		t.Fatal("unowned alias admitted", out, n)
	}
}
