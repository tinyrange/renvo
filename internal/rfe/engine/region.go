package engine

import (
	"fmt"
	emu "renvo.dev/internal/rfe/runtime"
)

// Already validated hot blocks may form a bounded cyclic trace. Each internal
// branch is checked natively; only its chosen edge stays inside the region.
type regionPart struct {
	pc    uint64
	block *compiledBlock
}

func (e *Engine) regionParts(c CPU, first *compiledBlock) []regionPart {
	root := c.Registers()[e.arch.PC]
	flow := first.instructions[len(first.instructions)-1].Flow
	if flow.Known && flow.Next == root {
		return []regionPart{{root, first}}
	}
	memory, ok := c.MemoryBus().(CodeContextMemory)
	if !ok {
		return nil
	}
	stamp := memory.CodeContext()
	if stamp.Identity == nil || first.context != stamp || len(first.dependencies) == 0 {
		return nil
	}
	parts := []regionPart{{root, first}}
	seen := map[uint64]bool{root: true}
	// Explore both legal direct edges, but cap cold discovery independently of
	// native execution. A failed taken edge must not hide a fallthrough cycle.
	attempts := 0
	best := []regionPart(nil)
	bestScore := float64(0)
	var search func(int, float64, float64) []regionPart
	search = func(total int, score, probability float64) []regionPart {
		if len(parts) > 1 && score > bestScore {
			best = append([]regionPart(nil), parts...)
			bestScore = score
		}
		current := parts[len(parts)-1]
		if len(current.block.instructions) == 0 || !current.block.instructions[len(current.block.instructions)-1].Flow.Known {
			return nil
		}
		flow := current.block.instructions[len(current.block.instructions)-1].Flow
		next, alternate, conditional := flow.Next, flow.Alternate, flow.Conditional
		targets := []uint64{next}
		if conditional && alternate != next {
			// Prefer the conventional loop back edge, otherwise fallthrough.
			// This is only discovery order: every selected edge is still guarded.
			if next > current.pc && alternate < next {
				targets = []uint64{alternate, next}
			} else {
				targets = append(targets, alternate)
			}
		}
		counts := e.edgeCounts[current.block.instructions[len(current.block.instructions)-1].PC]
		if conditional && counts[0]+counts[1] != 0 {
			if counts[1] > counts[0] {
				targets = []uint64{alternate, next}
			} else {
				targets = []uint64{next, alternate}
			}
		}
		for _, target := range targets {
			attempts++
			if attempts > 256 {
				return nil
			}
			if target == root {
				return append([]regionPart(nil), parts...)
			}
			candidate := e.blocks[target]
			if seen[target] || candidate == nil || !candidate.native || candidate.context != stamp || len(candidate.dependencies) == 0 || len(candidate.instructions) == 0 || total+len(candidate.instructions) > e.config.MaxRegionInstructions {
				continue
			}
			seen[target] = true
			parts = append(parts, regionPart{target, candidate})
			reach := probability
			if conditional && counts[0]+counts[1] != 0 {
				edge := 0
				if target == alternate {
					edge = 1
				}
				reach *= float64(counts[edge]+1) / float64(counts[0]+counts[1]+2)
			}
			result := search(total+len(candidate.instructions), score+reach*float64(len(candidate.instructions)), reach)
			parts = parts[:len(parts)-1]
			delete(seen, target)
			if len(result) != 0 {
				return result
			}
		}
		return nil
	}
	if cycle := search(len(first.instructions), float64(len(first.instructions)), 1); len(cycle) != 0 {
		return cycle
	}
	return best
}

func (e *Engine) prepareRegion(c CPU, block *compiledBlock) {
	if !block.native || e.native == nil || block.regionAttempted && block.regionGeneration == e.Stats.Promotions {
		return
	}
	block.regionAttempted = true
	block.regionGeneration = e.Stats.Promotions
	parts := e.regionParts(c, block)
	if len(parts) == 0 {
		return
	}
	count := 0
	for _, part := range parts {
		count += len(part.block.instructions)
	}
	if count > e.config.MaxRegionInstructions || block.region && count <= block.regionInstructions {
		return
	}
	var b emu.Builder
	pc := c.Registers()[e.arch.PC]
	b.Store(e.arch.PC, b.Constant(pc))
	completed := 0
	prefix := [17]uint8{}
	for index, part := range parts {
		for i, ins := range part.block.instructions {
			if !e.arch.Lower(&b, ins, completed) {
				return
			}
			completed++
			if completed < len(prefix) {
				prefix[completed] = prefix[completed-1] + part.block.memoryPrefix[i+1] - part.block.memoryPrefix[i]
			}
		}
		if index+1 < len(parts) {
			b.Checkpoint(e.arch.StateWords, completed)
			b.RegionGuard(b.Binary(emu.Equal, b.Load(e.arch.PC), b.Constant(parts[index+1].pc)))
			// A successful guard proves this exact edge. Refine forwarded PC
			// only after the precise checkpoint/guard, never on a side exit.
			b.Store(e.arch.PC, b.Constant(parts[index+1].pc))
		}
	}
	b.Checkpoint(e.arch.StateWords, count)
	// If neither legal direct successor can return to the root, this is
	// provably a one-shot trace. Preserve the generic cyclic path for indirect
	// branches and calls whose targets the guest adapter does not model.
	lastPart := parts[len(parts)-1]
	flow := lastPart.block.instructions[len(lastPart.block.instructions)-1].Flow
	oneShot := flow.Known && flow.Next != pc && (!flow.Conditional || flow.Alternate != pc)
	if oneShot {
		b.LoopContinue(b.Constant(0))
	} else {
		b.LoopContinue(b.Binary(emu.Equal, b.Load(e.arch.PC), b.Constant(pc)))
	}
	direct := false
	if memory, ok := c.MemoryBus().(NativeMemory); ok && e.config.MaxNativeInstructions > 64 {
		context := memory.NativeContext()
		direct = context != nil && context.DirectSize == 64<<20
	}
	ops := b.FinishMemory(e.arch.StateWords)
	var entry int
	var err error
	if memory, ok := c.MemoryBus().(CodeVersionedNativeMemory); ok && memory.NativeCodeVersions() && !direct {
		entry, err = e.native.CompileCodeVersionedLoop(ops, e.arch.StateWords, count, memory.NativeContext())
	} else {
		entry, err = e.native.CompileLoopMode(ops, e.arch.StateWords, count, direct)
	}
	if err != nil {
		e.Stats.CompileFailures++
		return
	}
	// Validation of this entry must cover all constituent pages, not just its
	// original block. An executable mutation clears links before revalidation.
	dependencies := append([]codeDependency(nil), block.dependencies...)
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
	block.dependencies = dependencies
	block.regionEntry, block.regionInstructions, block.region = entry, count, true
	block.regionPrefix = prefix
	e.Stats.Regions++
	e.Stats.NativeBytes = e.native.Bytes
	_ = e.native.NameEntry(entry, fmt.Sprintf("%s_%x_region_%d_instructions", e.arch.Name, pc, count))
	e.Stats.ProfileErrors = e.native.SymbolErrors
}

// Only completed low-tier instructions provide observations. Native-region
// discovery remains bounded and every selected edge is still guarded.
func (e *Engine) observeEdge(ins Instruction, next uint64) {
	if !ins.Flow.Known || !ins.Flow.Conditional || ins.Flow.Next == ins.Flow.Alternate {
		return
	}
	edge := 0
	if next == ins.Flow.Alternate {
		edge = 1
	} else if next != ins.Flow.Next {
		return
	}
	if e.edgeCounts == nil {
		e.edgeCounts = map[uint64][2]uint64{}
	}
	counts, exists := e.edgeCounts[ins.PC]
	if !exists && len(e.edgeCounts) >= e.config.MaxBlocks {
		return
	}
	if counts[0]+counts[1] < 65536 {
		counts[edge]++
		e.edgeCounts[ins.PC] = counts
	}
}
