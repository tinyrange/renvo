//go:build browser

package engine

import (
	emu "renvo.dev/internal/rfe/runtime"
)

type graphScratch struct {
	blocks [256]emu.GraphBlock
	ops    [8192]emu.Op
}

// A graph only reads existing native block metadata in one exact code context.
// No cold decoding, speculative execution, dynamic-target guessing or cap growth.
func (e *Engine) prepareGraph(c CPU, first *compiledBlock) bool {
	memory, ok := c.MemoryBus().(CodeVersionedNativeMemory)
	if !ok || !memory.NativeCodeVersions() || first.instructions[0].ScheduleBoundary {
		return false
	}
	contextMemory, ok := c.MemoryBus().(CodeContextMemory)
	if !ok {
		return false
	}
	stamp := contextMemory.CodeContext()
	if stamp.Identity == nil || first.context != stamp || len(first.dependencies) == 0 {
		return false
	}
	parts, count, branched, continuations, callCount := e.graphCallParts(c.Registers()[e.arch.PC], first, stamp)
	if (!branched && callCount == 0) || len(parts) < 2 || first.region && count <= first.regionInstructions {
		return false
	}
	// Independent builders prevent SSA facts from crossing unexecuted CFG edges.
	// Body IR is copied out of the engine's reused temporary translation buffer.
	graph := e.graph.blocks[:len(parts)]
	used := 0
	for index, part := range parts {
		b, bl, bh := e.beginTranslation(true)
		b.Store(e.arch.PC, b.Constant(part.pc))
		for i, ins := range part.block.instructions {
			if !e.arch.Lower(b, ins, i) {
				e.endTranslation(bl, bh)
				return false
			}
		}
		b.Checkpoint(e.arch.StateWords, len(part.block.instructions))
		ops := e.finishTranslation(b, true, bl, bh)
		if len(ops) > len(e.graph.ops)-used {
			return false
		}
		copy(e.graph.ops[used:], ops)
		graph[index] = emu.GraphBlock{PC: part.pc, Instructions: len(part.block.instructions), Ops: e.graph.ops[used : used+len(ops)]}
		used += len(ops)
		flow := part.block.instructions[len(part.block.instructions)-1].Flow
		call := false
		for j := 0; j < callCount; j++ {
			if (flow.Call || flow.IndirectCall) && flow.Continuation == continuations[j] {
				call = true
			}
		}
		if flow.Known || call {
			graph[index].Successors[0] = flow.Next
			graph[index].SuccessorCount = 1
			if flow.IndirectCall {
				graph[index].SuccessorCount = int(part.block.linkSuccessorCount)
				for j := 0; j < graph[index].SuccessorCount; j++ {
					graph[index].Successors[j] = part.block.linkSuccessors[j]
				}
			}
			if flow.Conditional && flow.Alternate != flow.Next {
				graph[index].Successors[1] = flow.Alternate
				graph[index].SuccessorCount = 2
			}
		}
		if flow.Return {
			targets, n := graphReturnSuccessors(part.block, parts, continuations, callCount, first.browserDemand)
			graph[index].Successors = targets
			graph[index].SuccessorCount = n
		}
	}
	entry, err := e.native.CompileCodeVersionedGraph(graph, e.arch.StateWords, e.arch.PC, memory.NativeContext())
	// Installed code has no pointers into graph/IR scratch.
	if err != nil {
		e.Stats.CompileFailures++
		return false
	}
	dependencies := append([]codeDependency(nil), first.dependencies...)
	for _, part := range parts[1:] {
		for _, dependency := range part.block.dependencies {
			found := false
			for _, old := range dependencies {
				if old.address == dependency.address {
					found = true
					break
				}
			}
			if !found {
				dependencies = append(dependencies, dependency)
			}
		}
	}
	first.dependencies = dependencies
	first.regionEntry, first.regionInstructions, first.region = entry, count, true
	first.regionPrefix = [17]uint8{} // browser leaves publish exact memory counts themselves
	e.Stats.Regions++
	e.Stats.NativeBytes = e.native.Bytes
	return true
}
