package engine

import (
	"fmt"
	emu "renvo.dev/internal/rfe/runtime"
)

// preparePureDiamondRegion closes only a direct cycle with one-block pure
// optional arms. Both possible PC values are proven from lowered SSA before a
// join is refined. No faulting or scheduling effect is speculatively executed.
func (e *Engine) preparePureDiamondRegion(c CPU, first *compiledBlock) bool {
	if !emu.NativeVariablePureLoopsAvailable() {
		return false
	}
	memory, ok := c.MemoryBus().(CodeContextMemory)
	if !ok {
		return false
	}
	stamp := memory.CodeContext()
	if stamp.Identity == nil || first.context != stamp {
		return false
	}
	root := c.Registers()[e.arch.PC]
	eligible := func(block *compiledBlock) bool {
		if block == nil || !block.native || block.context != stamp || len(block.dependencies) == 0 || len(block.instructions) == 0 {
			return false
		}
		for _, ins := range block.instructions {
			if ins.Memory || ins.InterpretOnly || ins.ScheduleBoundary {
				return false
			}
		}
		return true
	}
	b := emu.Builder{MemoryInstructionProgress: true}
	b.Store(e.arch.PC, b.Constant(root))
	parts := []regionPart{}
	seen := map[uint64]bool{}
	current, pc := first, root
	count, arms := 0, 0
	actual := b.Constant(0)
	closed := false
	guardedJoin := false
	if joined, maximum, dynamic, ok := e.lowerPureJoinEntry(&b, first, root, eligible); ok {
		parts, count, actual, arms, closed = joined, maximum, dynamic, 1, true
		guardedJoin = true
	} else {
		// A failed lowering may have emitted tentative SSA. Never reuse it.
		b = emu.Builder{MemoryInstructionProgress: true}
		b.Store(e.arch.PC, b.Constant(root))
		actual = b.Constant(0)
	}
	for attempts := 0; !closed && attempts < 64; attempts++ {
		if seen[pc] || !eligible(current) || count+len(current.instructions) > e.config.MaxRegionInstructions {
			return false
		}
		seen[pc] = true
		parts = append(parts, regionPart{pc, current})
		for _, ins := range current.instructions {
			if !e.arch.Lower(&b, ins, count) {
				return false
			}
			count++
		}
		actual = b.Binary(emu.Add, actual, b.Constant(uint64(len(current.instructions))))
		flow := current.instructions[len(current.instructions)-1].Flow
		if !flow.Known {
			return false
		}
		alternate := flow.Next
		if flow.Conditional {
			alternate = flow.Alternate
		}
		if !b.ChoiceOf(b.Load(e.arch.PC), flow.Next, alternate) {
			return false
		}
		if flow.Next == root || flow.Conditional && flow.Alternate == root {
			closed = true
			break
		}
		if !flow.Conditional || flow.Next == flow.Alternate {
			pc = flow.Next
			current = e.blocks[pc]
			continue
		}
		// Recognize a triangle: either edge reaches the other's join through
		// one unconditional block. Discovery order is not an execution prediction.
		optional, join := flow.Next, flow.Alternate
		arm := e.blocks[optional]
		triangleArm := func(q *compiledBlock, join uint64) *compiledBlock {
			if q == nil {
				return nil
			}
			// Discovery may have decoded through another entry's join. Only
			// the exact instruction prefix before that join is optional.
			for i, ins := range q.instructions {
				if i > 0 && ins.PC == join {
					prefix := *q
					prefix.instructions = q.instructions[:i]
					q = &prefix
					break
				}
			}
			if !eligible(q) {
				return nil
			}
			f := q.instructions[len(q.instructions)-1].Flow
			if !f.Known || f.Conditional || f.Next != join {
				return nil
			}
			return q
		}
		arm = triangleArm(arm, join)
		if arm == nil {
			optional, join = join, optional
			arm = triangleArm(e.blocks[optional], join)
			if arm == nil {
				return false
			}
		}
		if seen[optional] || seen[join] || count+len(arm.instructions) > e.config.MaxRegionInstructions {
			return false
		}
		take := b.Binary(emu.Equal, b.Load(e.arch.PC), b.Constant(optional))
		if !b.PureIf(take, e.arch.StateWords, func(q *emu.Builder) bool {
			q.Store(e.arch.PC, q.Constant(optional))
			for i, ins := range arm.instructions {
				if !e.arch.Lower(q, ins, count+i) {
					return false
				}
			}
			return q.ChoiceOf(q.Load(e.arch.PC), join, join)
		}) {
			return false
		}
		actual = b.Binary(emu.Add, actual, b.Choose(take, b.Constant(uint64(len(arm.instructions))), b.Constant(0)))
		count += len(arm.instructions)
		arms++
		seen[optional] = true
		parts = append(parts, regionPart{optional, arm})
		b.Store(e.arch.PC, b.Constant(join))
		pc, current = join, e.blocks[join]
	}
	if !closed || arms == 0 || count > e.config.MaxRegionInstructions {
		return false
	}
	// PureIf checks every optional arm; also reject effects in the common path.
	for _, op := range b.Ops {
		if op.Kind == emu.Guard || op.Kind == emu.RegionGuard && !guardedJoin || op.Kind == emu.NativeElapsed || op.Kind == emu.MemoryLoad || op.Kind == emu.MemoryStore || op.Kind == emu.MemoryPairStore || (op.Kind == emu.Progress || op.Kind == emu.StoreState) && !guardedJoin {
			return false
		}
	}
	condition := b.Binary(emu.Equal, b.Load(e.arch.PC), b.Constant(root))
	b.Checkpoint(e.arch.StateWords, count)
	b.LoopCount(actual)
	b.LoopContinue(condition)
	ops := b.FinishMemory(e.arch.StateWords)
	entry, err := e.native.CompileLoop(ops, e.arch.StateWords, count)
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
	first.regionCyclic, first.regionDiamond = true, true
	first.regionPrefix = [17]uint8{}
	e.Stats.Regions++
	e.Stats.NativeBytes = e.native.Bytes
	_ = e.native.NameEntry(entry, fmt.Sprintf("%s_%x_pure_diamond_%d_instructions", e.arch.Name, root, count))
	e.Stats.ProfileErrors = e.native.SymbolErrors
	return true
}

// lowerPureJoinEntry rotates a pure triangle to a branch-at-join entry. The
// entry branch exits through an exact prefix checkpoint. Only the optional
// pure arm uses SSA selection; the common continuation executes under its
// proven entry guard. No decoder metadata alone refines an actual PC.
func (e *Engine) lowerPureJoinEntry(b *emu.Builder, first *compiledBlock, root uint64, eligible func(*compiledBlock) bool) ([]regionPart, int, emu.Value, bool) {
	if !eligible(first) {
		return nil, 0, 0, false
	}
	entryFlow := first.instructions[len(first.instructions)-1].Flow
	if !entryFlow.Known || !entryFlow.Conditional || entryFlow.Next == entryFlow.Alternate {
		return nil, 0, 0, false
	}
	for _, candidate := range []uint64{entryFlow.Next, entryFlow.Alternate} {
		common := e.blocks[candidate]
		if candidate == root || !eligible(common) {
			continue
		}
		flow := common.instructions[len(common.instructions)-1].Flow
		if !flow.Known || !flow.Conditional || flow.Next == flow.Alternate {
			continue
		}
		optional := flow.Next
		if optional == root {
			optional = flow.Alternate
		} else if flow.Alternate != root {
			continue
		}
		if optional == candidate || optional == root {
			continue
		}
		arm := e.blocks[optional]
		if arm == nil {
			continue
		}
		// An optional entry can have been decoded through the join. Admit only
		// the exact prefix before it, retaining all original code dependencies.
		for i, ins := range arm.instructions {
			if i > 0 && ins.PC == root {
				prefix := *arm
				prefix.instructions = arm.instructions[:i]
				arm = &prefix
				break
			}
		}
		if !eligible(arm) {
			continue
		}
		armFlow := arm.instructions[len(arm.instructions)-1].Flow
		if !armFlow.Known || armFlow.Conditional || armFlow.Next != root {
			continue
		}
		maximum := len(first.instructions) + len(common.instructions) + len(arm.instructions)
		if maximum > e.config.MaxRegionInstructions {
			continue
		}
		for i, ins := range first.instructions {
			if !e.arch.Lower(b, ins, i) {
				return nil, 0, 0, false
			}
		}
		if !b.ChoiceOf(b.Load(e.arch.PC), entryFlow.Next, entryFlow.Alternate) {
			return nil, 0, 0, false
		}
		take := b.Binary(emu.Equal, b.Load(e.arch.PC), b.Constant(candidate))
		bodyCount := b.Constant(uint64(len(common.instructions)))
		// The entry's exact static prefix precedes every optional arm. A
		// failed edge reconstructs that prefix; successful iterations can
		// use the common state directly without an outer SSA select layer.
		b.Checkpoint(e.arch.StateWords, len(first.instructions))
		b.RegionGuard(take)
		b.Store(e.arch.PC, b.Constant(candidate))
		for i, ins := range common.instructions {
			if !e.arch.Lower(b, ins, len(first.instructions)+i) {
				return nil, 0, 0, false
			}
		}
		if !b.ChoiceOf(b.Load(e.arch.PC), root, optional) {
			return nil, 0, 0, false
		}
		takeArm := b.Binary(emu.Equal, b.Load(e.arch.PC), b.Constant(optional))
		if !b.PureIf(takeArm, e.arch.StateWords, func(q *emu.Builder) bool {
			q.Store(e.arch.PC, q.Constant(optional))
			for i, ins := range arm.instructions {
				if !e.arch.Lower(q, ins, len(first.instructions)+len(common.instructions)+i) {
					return false
				}
			}
			return q.ChoiceOf(q.Load(e.arch.PC), root, root)
		}) {
			return nil, 0, 0, false
		}
		bodyCount = b.Binary(emu.Add, bodyCount, b.Choose(takeArm, b.Constant(uint64(len(arm.instructions))), b.Constant(0)))
		b.Store(e.arch.PC, b.Constant(root))
		actual := b.Binary(emu.Add, b.Constant(uint64(len(first.instructions))), bodyCount)
		return []regionPart{{root, first}, {candidate, common}, {optional, arm}}, maximum, actual, true
	}
	return nil, 0, 0, false
}
