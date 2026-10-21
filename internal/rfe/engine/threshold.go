package engine

// Privileged adapters may delay expensive promotion while retaining the same
// interpreted/IR semantics and all original code, memory and byte-budget checks.
func (e *Engine) nativeThreshold(block *compiledBlock) uint64 {
	if e.config.Adaptive {
		saving := e.interpretedCost()
		if saving == 0 {
			return ^uint64(0)
		}
		cost := e.compilationCost(len(block.instructions))
		perVisit := saving * uint64(len(block.instructions))
		return cost/perVisit + 1
	}
	threshold := e.config.NativeThreshold
	for _, ins := range block.instructions {
		if ins.NativeThreshold > threshold {
			threshold = ins.NativeThreshold
		}
	}
	return threshold
}
