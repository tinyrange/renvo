package engine

// Return observations add hints only for parts already owned by this exact
// graph. No guest stack is inferred, no target is decoded, and full-width
// architectural equality still decides every emitted return transfer.
func graphReturnSuccessors(block *compiledBlock, parts []regionPart, calls [3]uint64, count int, demanded bool) ([4]uint64, int) {
	var out [4]uint64
	for i := 0; i < count; i++ {
		out[i] = calls[i]
	}
	if !demanded || block.linkSuccessorCount > 2 {
		return out, count
	}
	for i := 0; i < int(block.linkSuccessorCount) && count < len(out); i++ {
		target, duplicate, present := block.linkSuccessors[i], false, false
		for j := 0; j < count; j++ {
			if out[j] == target {
				duplicate = true
			}
		}
		for _, part := range parts {
			if part.pc == target {
				present = true
			}
		}
		if present && !duplicate {
			out[count] = target
			count++
		}
	}
	return out, count
}
