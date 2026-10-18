package engine

import (
	"fmt"
	emu "renvo.dev/internal/rfe/runtime"
)

// Only blocks validated in the current code context are published. Native
// non-executable stores preserve that context. All other memory operations exit
// before returning to Go, so no code-context change can occur within this loop.
func (e *Engine) runLinked(c CPU, remaining uint64, first *compiledBlock, memory NativeMemory, context *emu.MemoryContext, stamp CodeStamp, quanta int) error {
	claimed := context.ClaimLinks(e.native)
	if e.linkContexts == nil {
		e.linkContexts = make(map[*emu.MemoryContext]CodeStamp)
	}
	previous, known := e.linkContexts[context]
	if !known && len(e.linkContexts) >= 64 {
		clear(e.linkContexts)
	}
	if claimed || previous != stamp {
		context.ClearLinks()
		e.Stats.LinkContextChanges++
		e.linkContexts[context] = stamp
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
	if owner, ok := memory.(CodeVersionedNativeMemory); ok && owner.NativeCodeVersions() {
		if !e.native.AdmitCodeVersionedContext(context) {
			return e.stepBlock(c, remaining, first)
		}
	}
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
	// The private proof table has its own bounded replacement order. A public
	// descriptor surviving another context does not prove its private target
	// is still resident. Refresh the selected immutable admission independently.
	if err := e.native.PrepareTargetLink(pc, entry, e.arch.StateWords, instructions); err != nil {
		return err
	}
	published := link == nil || link.Entry != uint64(entry) || link.Instructions != uint64(instructions)
	if published {
		if region {
			link = context.PublishLink(pc, entry, instructions, first.regionPrefix)
		} else {
			link = context.PublishLink(pc, entry, instructions, first.memoryPrefix)
		}
	}
	if link == nil {
		return e.stepBlock(c, remaining, first)
	}
	if !e.prefillBrowserLinks(c, first, memory, context, stamp, published) {
		return e.stepBlock(c, remaining, e.block(c))
	}
	link = context.LookupLink(pc)
	if link == nil {
		return e.stepBlock(c, remaining, first)
	}
	// A changed tuple starts a fresh region-demand countdown. Preserve a
	// matching region's existing countdown across host sessions; otherwise a
	// root reached fewer times in each bounded session could never demand an
	// upgrade. Ordinary leaves retain their existing generation hint.
	if published || !region {
		link.Reserved = e.regionPublicationHint(first, e.Stats.Promotions)
	}
	first.visits++
	budget := remaining
	if budget > uint64(e.config.MaxNativeInstructions) {
		budget = uint64(e.config.MaxNativeInstructions)
	}
	fastRefill := e.beginFastRefill(memory, context)
	var err error
	if e.config.MaxNativeInstructions > 64 {
		if owner, ok := memory.(NativeBackingMemory); ok && owner.NativeCodeVersions() {
			err = e.native.RunLinkedBackingSession(c.Registers(), context, budget, owner.NativeBacking())
		} else {
			err = e.native.RunLinkedSession(c.Registers(), context, budget)
		}
		if err != nil {
			if fastRefill {
				e.endFastRefill(false)
			}
			return fmt.Errorf("%w: pc=%#x budget=%d total=%d remaining=%d memory=%d status=%d retired=%d", err, c.Registers()[e.arch.PC], budget, context.Total, context.Remaining, context.MemoryTotal, context.Status, context.Retired)
		}
	} else if quanta == 1 || e.config.MaxNativeInstructions < 64 {
		err = e.native.CallLinked(c.Registers(), context, budget)
		if err != nil {
			if fastRefill {
				e.endFastRefill(false)
			}
			return err
		}
	} else {
		// No Go guest-memory mutation occurs between these distinct native
		// calls. Non-executable native stores preserve the initial code proof;
		// a fault or cold target stops scheduling before any host slow path.
		_, err = e.native.RunLinkedQuanta(c.Registers(), context, quanta, remaining, e.Stats.Promotions)
	}
	if fastRefill {
		e.endFastRefill(err == nil)
	}
	e.Stats.NativeSessions++
	if context.Status == 0 && context.Remaining != 0 {
		pc := c.Registers()[e.arch.PC]
		link := context.LookupLink(pc)
		if link == nil {
			e.Stats.LinkMissStops++
			if context.Total != 0 {
				e.observeBrowserLink(context, pc)
			}
			if e.config.ProfileNative {
				e.recordLinkMiss(pc)
			}
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
	if e.finishBrowserRegionUpgrade(c, context) {
		return nil
	}
	if context.Status == emu.RegionExit {
		e.Stats.RegionExits++
		return nil
	}
	if context.Status != 0 {
		e.Stats.MemoryExits++
		if e.config.ProfileNative {
			e.recordMemoryExit(c.Registers()[e.arch.PC], context)
		}
		if context.Status == 1 {
			memory.NativeRefill(context.Address)
			if e.memoryRetryAllowed && !e.memoryRetry && !e.inNativeBatch && context.Remaining != 0 && refillReady(context) {
				return e.retryMemory(c, context.Remaining)
			}
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
