package engine

import emu "renvo.dev/internal/rfe/runtime"

// Only blocks validated in the current code context are published. Native
// non-executable stores preserve that context. All other memory operations exit
// before returning to Go, so no code-context change can occur within this loop.
func (e *Engine) runLinked(c CPU, remaining uint64, first *compiledBlock, memory NativeMemory, context *emu.MemoryContext, stamp CodeStamp, quanta int) error {
	if context.ClaimLinks(e.native) || e.linkContext != stamp {
		context.ClearLinks()
		e.linkContext = stamp
	}
	if !e.linkReady {
		if err := e.native.PrepareLinks(e.arch.StateWords, e.arch.PC); err != nil {
			// Arena exhaustion preserves the original leaf fallback.
			return e.stepBlock(c, remaining, first)
		}
		e.linkReady = true
		e.Stats.NativeBytes = e.native.Bytes
		e.Stats.ProfileErrors = e.native.SymbolErrors
	}
	if !first.regionAttempted || first.regionGeneration != e.Stats.Promotions {
		e.prepareRegion(c, first)
		e.Stats.NativeBytes = e.native.Bytes
		e.Stats.ProfileErrors = e.native.SymbolErrors
	}
	// An engine may be reused with another address space. Specialized regions
	// never confer their owner's contract on a generic context; its ordinary
	// checked leaves remain usable without entering the specialized dispatcher.
	if !e.native.CanLinkMemory(context) {
		return e.stepBlock(c, remaining, first)
	}
	pc := c.Registers()[e.arch.PC]
	entry, instructions := first.entry, len(first.instructions)
	region := first.region && uint64(first.regionInstructions) <= remaining
	if region {
		entry, instructions = first.regionEntry, first.regionInstructions
	}
	link := context.LookupLink(pc)
	// A block entry and its prefix are immutable in the append-only arena.
	// Avoid revalidating/copying a publication already in this exact context.
	if link == nil || link.Entry != uint64(entry) || link.Instructions != uint64(instructions) {
		if err := e.native.PrepareTargetLink(pc, entry, e.arch.StateWords, instructions); err != nil {
			return err
		}
		if region {
			context.PublishLink(pc, entry, instructions, first.regionPrefix)
		} else {
			context.PublishLink(pc, entry, instructions, first.memoryPrefix)
		}
	}
	link = context.LookupLink(pc)
	if link == nil {
		return e.stepBlock(c, remaining, first)
	}
	// This publication has completed cold region preparation at the current
	// promotion generation. Suppress per-quantum retries for an installed region;
	// the next host-selected session may upgrade it after further promotions.
	link.Reserved = e.Stats.Promotions
	if first.region {
		link.Reserved = ^uint64(0)
	}
	first.visits++
	budget := remaining
	if budget > uint64(e.config.MaxNativeInstructions) {
		budget = uint64(e.config.MaxNativeInstructions)
	}
	var err error
	if e.config.MaxNativeInstructions > 64 {
		err = e.native.RunLinkedSession(c.Registers(), context, budget)
		if err != nil {
			return err
		}
	} else if quanta == 1 || e.config.MaxNativeInstructions < 64 {
		err = e.native.CallLinked(c.Registers(), context, budget)
		if err != nil {
			return err
		}
	} else {
		// No Go guest-memory mutation occurs between these distinct native
		// calls. Non-executable native stores preserve the initial code proof;
		// a fault or cold target stops scheduling before any host slow path.
		_, err = e.native.RunLinkedQuanta(c.Registers(), context, quanta, remaining, e.Stats.Promotions)
	}
	e.Stats.NativeSessions++
	if context.Status == 0 && context.Remaining != 0 {
		pc := c.Registers()[e.arch.PC]
		link := context.LookupLink(pc)
		if link == nil {
			e.Stats.LinkMissStops++
		} else if link.Instructions > context.Remaining {
			e.Stats.BudgetStops++
		} else {
			e.Stats.OtherStops++
		}
	}
	e.Stats.RegionExits += context.LoopExits
	e.Stats.LoopIterations += context.LoopIterations
	*c.Retirement() += context.Total
	e.Stats.Native += context.Total
	e.Stats.Linked += context.Total
	e.Stats.NativeMemory += context.MemoryTotal
	if err != nil {
		return err
	}
	if context.Status == emu.RegionExit {
		e.Stats.RegionExits++
		return nil
	}
	if context.Status != 0 {
		e.Stats.MemoryExits++
		if context.Status == 1 {
			memory.NativeRefill(context.Address)
		}
		return e.interpret(c)
	}
	if context.Total == 0 {
		if quanta != 1 {
			first = e.block(c)
		}
		return e.stepBlock(c, remaining, first)
	}
	return nil
}
