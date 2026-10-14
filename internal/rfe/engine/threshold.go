package engine

// Privileged adapters may delay expensive promotion while retaining the same
// interpreted/IR semantics and all original code, memory and byte-budget checks.
func (e *Engine) nativeThreshold(block *compiledBlock) uint64 {
	threshold := e.config.NativeThreshold
	for _, ins := range block.instructions {
		if ins.NativeThreshold > threshold {
			threshold = ins.NativeThreshold
		}
	}
	return threshold
}
