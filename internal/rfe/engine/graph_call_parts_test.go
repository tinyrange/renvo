package engine

import "testing"

func TestDemandCallClosureIsSingleOwnedAndBounded(t *testing.T) {
	stamp := CodeStamp{Identity: new(CodeIdentity), Epoch: 9}
	for _, mode := range []string{"hot", "cold-root", "context", "dependency", "cold-target", "boundary", "cap", "missing", "nested"} {
		t.Run(mode, func(t *testing.T) {
			makeBlock := func(pc uint64, flow Successors) *compiledBlock {
				return &compiledBlock{native: true, context: stamp, dependencies: []codeDependency{{0, stamp}}, instructions: []Instruction{{PC: pc, Length: 4, Flow: flow}}}
			}
			root := makeBlock(0, Successors{Call: true, Next: 16, Continuation: 4})
			root.browserDemand = true
			target := makeBlock(16, Successors{Return: true})
			cont := makeBlock(4, Successors{Known: true, Next: 0})
			e := Engine{config: DefaultEngineConfig(), blocks: map[uint64]*compiledBlock{0: root, 4: cont, 16: target}}
			switch mode {
			case "cold-root":
				root.browserDemand = false
			case "context":
				target.context.Epoch++
			case "dependency":
				target.dependencies = nil
			case "cold-target":
				target.native = false
			case "boundary":
				target.instructions[0].ScheduleBoundary = true
			case "cap":
				e.config.MaxRegionInstructions = 2
			case "missing":
				delete(e.blocks, 16)
			case "nested":
				target.instructions[0].Flow = Successors{Call: true, Next: 32, Continuation: 20}
				e.blocks[32] = makeBlock(32, Successors{Return: true})
				e.blocks[20] = makeBlock(20, Successors{Known: true, Next: 4})
			}
			parts, count, _, returnPC, call := e.graphCallParts(0, root, stamp)
			want := 2
			if mode == "hot" || mode == "nested" {
				want = 3
			} else if mode == "cold-root" {
				want = 1
			}
			if len(parts) != want || count != want || (call != 0) != (mode != "cold-root") || call != 0 && returnPC[0] != 4 {
				t.Fatal("invalid closure", len(parts), count, returnPC, call)
			}
			for _, part := range parts {
				if part.pc == 20 || part.pc == 32 {
					t.Fatal("nested call escaped the single continuation bound")
				}
			}
		})
	}
}
