//go:build browser

package engine

import emu "renvo.dev/internal/rfe/runtime"

// A generated dispatcher can yield at a proved target before executing it.
// The hint grants no code authority: the ordinary block lookup revalidates
// current bytes/context and normal region discovery validates every dependency.
func (e *Engine) finishBrowserRegionUpgrade(c CPU, context *emu.MemoryContext) bool {
	if context.Status != 5 {
		return false
	}
	context.Status = 0
	block := e.block(c)
	if block == nil || !block.native || len(block.instructions) == 0 || block.instructions[0].ScheduleBoundary {
		return true
	}
	if block.visits < e.config.RegionThreshold {
		block.visits = e.config.RegionThreshold
	}
	if !block.browserDemand {
		block.browserDemand = true
		block.regionAttempted = false
	}
	e.prepareRegion(c, block)
	// Optional preparation may fail or find no larger trace. In either case,
	// acknowledge successful regions and delay unsuccessful hints, not forever.
	pc := c.Registers()[e.arch.PC]
	if link := context.LookupLink(pc); link != nil {
		if block.region {
			link.Reserved = ^uint64(0)
		} else {
			// A failed early attempt may become useful after later promotions.
			// Revisit infrequently; the normal generation check avoids recompiling
			// an unchanged region. This value grants no execution authority.
			link.Reserved = 65536
		}
	}
	return true
}

func (e *Engine) regionPublicationHint(block *compiledBlock, generation uint64) uint64 {
	if block.region {
		if block.browserDemand || block.regionInstructions >= e.config.MaxRegionInstructions {
			return ^uint64(0)
		}
		return 256
	}
	return generation
}
