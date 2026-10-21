package engine

import emu "renvo.dev/internal/rfe/runtime"

// Ordinary leaves publish their checked entry without committing to a foreign
// session. A reused call/return path can therefore become a linked path without
// a loop-only rule or speculative session merely to discover its next leaf.
func (e *Engine) publishLeaf(c CPU, block *compiledBlock, context *emu.MemoryContext) {
	if block.instructions[0].ScheduleBoundary || !e.config.Adaptive || len(block.instructions) > 16 || block.context.Identity == nil {
		return
	}
	e.claimCodeLinks(context, block.context, block.proof != nil)
	if !e.linkReady {
		if e.native.PrepareLinks(e.arch.StateWords, e.arch.PC) != nil {
			return
		}
		e.linkReady = true
	}
	if owner, ok := c.MemoryBus().(CodeVersionedNativeMemory); ok && owner.NativeCodeVersions() {
		if !e.native.AdmitCodeVersionedContext(context) {
			return
		}
	}
	if !e.native.CanLinkMemory(context) {
		return
	}
	pc := block.instructions[0].PC
	link := context.LookupLink(pc)
	if link != nil && link.Entry == uint64(block.entry) && link.Instructions == uint64(len(block.instructions)) {
		return
	}
	if e.native.PrepareTargetLink(pc, block.entry, e.arch.StateWords, len(block.instructions)) != nil {
		return
	}
	link = context.LookupLink(pc)
	if link == nil || link.Entry != uint64(block.entry) {
		context.PublishLink(pc, block.entry, len(block.instructions), block.memoryPrefix)
	}
	if block.proof != nil {
		block.proof.Bind(context.LookupLink(pc), block.entry)
	}
}
func (e *Engine) linkedWorkReady(first *compiledBlock, context *emu.MemoryContext) bool {
	if !e.config.Adaptive || len(first.instructions) > 16 || first.region && first.regionInstructions > len(first.instructions) {
		return true
	}
	flow := first.instructions[len(first.instructions)-1].Flow
	target := flow.Next
	if !flow.Known {
		if !first.successorSeen {
			return false
		}
		target = first.successor
	} else if flow.Conditional {
		counts := e.edgeCounts[first.instructions[len(first.instructions)-1].PC]
		if counts[1] > counts[0] {
			target = flow.Alternate
		}
	}
	if e.linkContexts[context] != first.context {
		return false
	}
	return context.LookupLink(target) != nil
}
