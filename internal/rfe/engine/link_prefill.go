package engine

import emu "renvo.dev/internal/rfe/runtime"

// Revalidate and publish only already-native static or observed successors.
// This is metadata preparation, never guest execution or speculative decode.
// Every page dependency, full PC tag and immutable private proof still applies.
func (e *Engine) prefillLinks(c CPU, first *compiledBlock, memory NativeMemory, context *emu.MemoryContext, stamp CodeStamp) bool {
	snapshot, _ := memory.(NativeSnapshotMemory)
	if !prefillContextCurrent(memory, snapshot, stamp, context) {
		context.ClearLinks()
		return false
	}
	var validated prefillVersions
	if first.context == stamp {
		for _, dependency := range first.dependencies {
			validated.remember(dependency)
		}
	}
	versions, _ := c.MemoryBus().(VersionedMemory)
	var pending [16]*compiledBlock
	pending[0] = first
	count, published := 1, 0
	for count != 0 && published < len(pending) {
		count--
		block := pending[count]
		flow := block.instructions[len(block.instructions)-1].Flow
		static := 0
		if flow.Known {
			static = 1
			if flow.Conditional && flow.Alternate != flow.Next {
				static = 2
			}
		}
		n := static + int(block.linkSuccessorCount)
		for i := 0; i < n && published < len(pending); i++ {
			var pc uint64
			if i < static {
				pc = flow.Next
				if i == 1 {
					pc = flow.Alternate
				}
			} else {
				pc = block.linkSuccessors[i-static]
			}
			if context.LookupLink(pc) != nil {
				continue
			}
			next := e.prefillBlock(pc)
			if next == nil || !next.native || len(next.instructions) == 0 || next.instructions[0].PC != pc || next.instructions[0].ScheduleBoundary || len(next.dependencies) == 0 {
				continue
			}
			valid := next.context == stamp
			if !valid {
				var resolved bool
				valid, resolved = validated.check(e, versions, next, stamp)
				// Only a resolver can change mappings during this synchronous
				// metadata walk. Check even when it failed; already-current
				// blocks share the initial context observation, not new authority.
				if resolved && !prefillContextCurrent(memory, snapshot, stamp, context) {
					context.ClearLinks()
					return false
				}
			}
			if !valid {
				continue
			}
			entry, instructions, prefix := next.entry, len(next.instructions), next.memoryPrefix
			if next.region {
				entry, instructions, prefix = next.regionEntry, next.regionInstructions, next.regionPrefix
			}
			if err := e.native.PrepareTargetLink(pc, entry, e.arch.StateWords, instructions); err != nil {
				return true // optional prefill stops before the unchanged arena admission bound
			}
			link := context.PublishLink(pc, entry, instructions, prefix)
			if link == nil {
				continue
			}
			if next.region {
				link.Reserved = e.regionPublicationHint(next, e.Stats.Promotions)
			}
			published++
			if count < len(pending) {
				pending[count] = next
				count++
			}
		}
	}
	return true
}

// This call-local set includes the already-validated root's dependencies.
// Additional page proofs may come from a resolver or the engine's exact-context
// cache; neither can survive a mapping/context transition. Typical successors
// share a page, so resolving it for every block adds no proof. Full addresses
// AND version stamps must match. Any resolver call (also a failure) requires
// the caller's fresh code/native-context observation before publication.
type prefillVersions struct {
	pages [34]codeDependency
	count int
}

func (v *prefillVersions) remember(dependency codeDependency) {
	if v.count < len(v.pages) {
		v.pages[v.count] = dependency
		v.count++
	}
}
func (v *prefillVersions) check(e *Engine, memory VersionedMemory, block *compiledBlock, context CodeStamp) (bool, bool) {
	resolved := false
	if memory == nil {
		return false, false
	}
	for _, dependency := range block.dependencies {
		found := false
		for i := 0; i < v.count; i++ {
			if v.pages[i].address == dependency.address {
				if v.pages[i].stamp != dependency.stamp {
					return false, resolved
				}
				found = true
				break
			}
		}
		if found {
			continue
		}
		valid, read := e.checkCodePage(memory, dependency, context)
		resolved = resolved || read
		if !valid {
			return false, resolved
		}
		v.remember(dependency)
	}
	block.context = context
	return true, resolved
}

// Cache owned decoded metadata, not admission or absence. Discard clears
// this exact full-PC entry before releasing/reusing the block; arena recycle
// clears all entries. Every hit still takes the normal version/context/proof
// path. A page-mixed index avoids the all-pages/same-offset collision pattern.
func (e *Engine) prefillBlock(pc uint64) *compiledBlock {
	cached := &e.prefillLookup[((pc>>2)^(pc>>12))&1023]
	if cached.block != nil && cached.pc == pc {
		return cached.block
	}
	block := e.blocks[pc]
	if block != nil {
		*cached = dispatchEntry{pc, block}
	}
	return block
}

func prefillContextCurrent(memory NativeMemory, snapshot NativeSnapshotMemory, stamp CodeStamp, context *emu.MemoryContext) bool {
	if snapshot != nil {
		current, native := snapshot.NativeCodeContext()
		return current == stamp && native == context
	}
	return memory.CodeContext() == stamp && memory.NativeContext() == context
}
