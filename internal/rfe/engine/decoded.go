package engine

func (e *Engine) releaseInstruction(ins Instruction) {
	if e.arch.Release != nil && ins.Decoded != nil {
		e.arch.Release(ins)
	}
}
func (e *Engine) releaseInstructions(block *compiledBlock) {
	if e.arch.Release == nil {
		return
	}
	for i, ins := range block.instructions {
		e.releaseInstruction(ins)
		block.instructions[i].Decoded = nil
	}
}
func (e *Engine) equalInstructions(a, b Instruction) bool {
	if e.arch.EqualDecoded != nil {
		if !e.arch.EqualDecoded(a.Decoded, b.Decoded) {
			return false
		}
		a.Decoded = b.Decoded
	}
	return a == b
}
