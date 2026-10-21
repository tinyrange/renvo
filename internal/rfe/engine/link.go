package engine

import (
	"fmt"
	emu "renvo.dev/internal/rfe/runtime"
)

// Only blocks validated in the current code context are published. Native
// non-executable stores preserve that context. All other memory operations exit
// before returning to Go, so no code-context change can occur within this loop.
func (e *Engine) runLinked(c CPU, remaining uint64, first *compiledBlock, memory NativeMemory, context *emu.MemoryContext, stamp CodeStamp, quanta int) error {
	e.claimCodeLinks(context, stamp, first.proof != nil)
	if !e.linkReady {
		if err := e.native.PrepareLinks(e.arch.StateWords, e.arch.PC); err != nil {
			// Arena exhaustion preserves the original leaf fallback.
			return e.linkFallback(c, remaining, first)
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
			return e.linkFallback(c, remaining, first)
		}
	}
	if !e.native.CanLinkMemory(context) {
		return e.linkFallback(c, remaining, first)
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
	if link == nil || link.Entry != uint64(entry) || link.Instructions != uint64(instructions) {
		if region {
			context.PublishLink(pc, entry, instructions, first.regionPrefix)
		} else {
			context.PublishLink(pc, entry, instructions, first.memoryPrefix)
		}
	}
	link = context.LookupLink(pc)
	proof := first.proof
	if region {
		proof = first.regionProof
	}
	if proof != nil {
		proof.Bind(link, entry)
	}
	if link == nil {
		return e.linkFallback(c, remaining, first)
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
	if context.Stop == nil && budget > uint64(e.config.MaxNativeInstructions) {
		budget = uint64(e.config.MaxNativeInstructions)
	}
	var err error
	sessionStart := paybackTime{}
	if e.config.Adaptive {
		sessionStart = paybackNow()
	}
	if e.config.MaxNativeInstructions > 64 {
		if owner, ok := memory.(NativeBackingMemory); ok && owner.NativeCodeVersions() {
			err = e.native.RunLinkedBackingSession(c.Registers(), context, budget, owner.NativeBacking())
		} else {
			err = e.native.RunLinkedSession(c.Registers(), context, budget)
		}
		if err != nil {
			return fmt.Errorf("%w: pc=%#x budget=%d total=%d remaining=%d memory=%d status=%d retired=%d", err, c.Registers()[e.arch.PC], budget, context.Total, context.Remaining, context.MemoryTotal, context.Status, context.Retired)
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
	// A hot loop is not necessarily a profitable session: watched stores,
	// segment guards and short side exits can force a host return every few
	// instructions. Keep its ordinary native leaves, but periodically retry
	// the owned session instead of paying foreign-call/pin costs each time.
	// Allow one short refill/side exit before backing off: a one-off missing
	// descriptor should get its immediate chance to become a resident loop.
	// Repeated status-1 exits can also be persistent watched/readonly stores.
	if e.config.LoopRegionsOnly && remaining >= 64 {
		if context.Total >= 64 {
			first.sessionShort = 0
		} else {
			first.sessionShort++
			if first.sessionShort >= 2 {
				first.sessionRetry = first.visits + e.config.RegionThreshold
				first.sessionShort = 0
			}
		}
	}
	e.sessionResult(first, sessionStart, context, budget)
	e.Stats.NativeSessions++
	if context.Status == 0 && context.Remaining != 0 {
		pc := c.Registers()[e.arch.PC]
		link := context.LookupLink(pc)
		if link == nil {
			e.Stats.LinkMissStops++
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
			var old emu.NativePage
			if context.Memory != nil {
				old, _ = context.Memory.Lookup(context.Address >> 12)
			}
			memory.NativeRefill(context.Address)
			if context.Memory != nil {
				next, ok := context.Memory.Lookup(context.Address >> 12)
				if ok && (old.Data != next.Data || old.Permissions != next.Permissions) {
					first.sessionDebt = 0
				}
			}
		}
		return e.interpret(c)
	}
	if context.Total == 0 && context.Stop != nil && nativeStopRequested(context.Stop) {
		return nil
	}
	if context.Total == 0 {
		if quanta != 1 {
			first = e.block(c)
		}
		return e.linkFallback(c, remaining, first)
	}
	return nil
}

func (e *Engine) linkFallback(c CPU, remaining uint64, first *compiledBlock) error {
	if len(first.instructions) > 16 {
		return e.executeCached(c, first.instructions[0])
	}
	return e.stepBlock(c, remaining, first)
}
