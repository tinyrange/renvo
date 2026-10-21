package engine

import emu "renvo.dev/internal/rfe/runtime"

type codeLinkKey struct {
	memory   *emu.MemoryContext
	identity *CodeIdentity
}
type codeLinkState struct {
	table *emu.NativeLinkTable
	epoch uint64
}

// Code modes keep their own publications while sharing the same physical RAM
// view and pin lease. Restoring a mode is selection, not invalidation; exact
// mutation proofs revoke affected descriptors even in an inactive mode table.
func (e *Engine) claimCodeLinks(context *emu.MemoryContext, stamp CodeStamp, local bool) {
	claimed := context.ClaimLinks(e.native)
	if e.linkContexts == nil {
		e.linkContexts = make(map[*emu.MemoryContext]CodeStamp)
	}
	if local && e.config.CacheContexts && stamp.Identity != nil {
		if e.codeLinks == nil {
			e.codeLinks = make(map[codeLinkKey]*codeLinkState)
		}
		key := codeLinkKey{context, stamp.Identity}
		state := e.codeLinks[key]
		if state == nil {
			state = &codeLinkState{table: emu.NewNativeLinkTable()}
			e.codeLinks[key] = state
		}
		context.SelectLinkTable(state.table)
		if state.epoch != stamp.Epoch {
			context.ClearLinks()
			state.epoch = stamp.Epoch
		}
	} else if claimed || e.linkContexts[context] != stamp {
		context.SelectLinkTable(nil)
		context.ClearLinks()
	}
	if claimed || e.linkContexts[context] != stamp {
		e.Stats.LinkContextChanges++
	}
	e.linkContexts[context] = stamp
	e.linkContext = stamp
}
