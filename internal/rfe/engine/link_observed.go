package engine

// Remember only actually executed edges. These bounded full-PC hints allow
// indirect branches and returns to be republished after a context clear. A hint
// never decodes, compiles, executes or validates a target: prefill still checks
// current execute versions, boundaries and the immutable private admission.
func (e *Engine) observeLink(source, target uint64) {
	block := e.blocks[source]
	if block == nil || !block.native || len(block.instructions) == 0 || block.instructions[0].PC != source {
		return
	}
	flow := block.instructions[len(block.instructions)-1].Flow
	if flow.Known && (flow.Next == target || flow.Conditional && flow.Alternate == target) {
		return // already represented by the static graph, including cold/missing targets
	}
	for i := 0; i < int(block.linkSuccessorCount); i++ {
		if block.linkSuccessors[i] == target {
			return
		}
	}
	i := int(block.linkSuccessorCount)
	if i < len(block.linkSuccessors) {
		block.linkSuccessorCount++
	} else {
		i = int(block.linkVictim)
		block.linkVictim = (block.linkVictim + 1) & 3
	}
	block.linkSuccessors[i] = target
	block.prefillDirty = true
}
