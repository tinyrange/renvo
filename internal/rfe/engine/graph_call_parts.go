package engine

// A demanded root may discover one direct call, or one observed indirect site
// with up to two targets and two direct sites reusing one additional callee.
// Every part is already native in the current context; caps never grow, and
// these are only guarded edge hints, not assumptions about guest return stacks.
func (e *Engine) graphCallParts(root uint64, first *compiledBlock, stamp CodeStamp) ([]regionPart, int, bool, [3]uint64, int) {
	parts := e.regionResultScratch[:1]
	parts[0] = regionPart{root, first}
	count := len(first.instructions)
	branched := false
	var continuations [3]uint64
	calls, directCalls := 0, 0
	indirect := false
	returnSelected := false
	var directTarget uint64
	for i := 0; i < len(parts); i++ {
		part := parts[i]
		flow := part.block.instructions[len(part.block.instructions)-1].Flow
		targets := [3]uint64{flow.Next, flow.Alternate}
		edges := 1
		observedReturn := false
		if !flow.Known {
			if !first.browserDemand || calls == len(continuations) {
				continue
			}
			if flow.IndirectCall {
				if calls != 0 || part.block.linkSuccessorCount < 1 || part.block.linkSuccessorCount > 2 {
					continue
				}
				indirect = true
				edges = int(part.block.linkSuccessorCount)
				for j := 0; j < edges; j++ {
					targets[j] = part.block.linkSuccessors[j]
				}
			} else if flow.Call {
				if calls != 0 && (!indirect || directCalls == 2 || directCalls != 0 && flow.Next != directTarget) {
					continue
				}
				directCalls++
				directTarget = flow.Next
			} else if flow.Return && calls == 0 && !returnSelected && part.block.linkSuccessorCount >= 1 && part.block.linkSuccessorCount <= 2 {
				returnSelected, observedReturn = true, true
				edges = int(part.block.linkSuccessorCount)
				for j := 0; j < edges; j++ {
					targets[j] = part.block.linkSuccessors[j]
				}
			} else {
				continue
			}
			if flow.Call || flow.IndirectCall {
				continuations[calls] = flow.Continuation
				calls++
				targets[edges] = flow.Continuation
				edges++
			}
		} else if flow.Conditional && flow.Alternate != flow.Next {
			edges = 2
		}
		contained := 0
		for edge := 0; edge < edges; edge++ {
			target := targets[edge]
			found := false
			for _, old := range parts {
				if old.pc == target {
					found = true
					break
				}
			}
			if found {
				contained++
				continue
			}
			candidate := e.blocks[target]
			if candidate == nil || !candidate.native || candidate.context != stamp || len(candidate.dependencies) == 0 || len(candidate.instructions) == 0 || candidate.instructions[0].ScheduleBoundary || count+len(candidate.instructions) > e.config.MaxRegionInstructions {
				continue
			}
			parts = append(parts, regionPart{target, candidate})
			count += len(candidate.instructions)
			contained++
		}
		if contained >= 2 || observedReturn && contained != 0 {
			branched = true
		}
	}
	return parts, count, branched, continuations, calls
}
