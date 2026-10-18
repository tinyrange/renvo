package engine

type codeDependency struct {
	address uint64
	stamp   CodeStamp
}

func (e *Engine) validVersions(c CPU, block *compiledBlock, context CodeStamp) bool {
	memory, ok := c.MemoryBus().(VersionedMemory)
	if !ok {
		return false
	}
	for _, dependency := range block.dependencies {
		valid, _ := e.checkCodePage(memory, dependency, context)
		if !valid {
			return false
		}
	}
	block.context = context
	return true
}

// Byte validation covers each actual instruction extent, not count*word size.
func (e *Engine) validBytes(c CPU, pc uint64, block *compiledBlock) bool {
	for _, ins := range block.instructions {
		current, err := e.arch.Fetch(c.MemoryBus(), ins.PC)
		e.Stats.CodeReads++
		valid := err == nil && e.equalInstructions(current, ins)
		e.releaseInstruction(current)
		if !valid {
			return false
		}
	}
	return true
}
func (e *Engine) dependencies(c CPU, pc uint64, block *compiledBlock) {
	memory, ok := c.MemoryBus().(VersionedMemory)
	if !ok {
		return
	}
	tail := block.instructions[len(block.instructions)-1]
	first, last := pc&^uint64(4095), (tail.PC+uint64(tail.Length)-1)&^uint64(4095)
	for address := first; ; address += 4096 {
		stamp, err := memory.CodeVersion(address)
		e.Stats.VersionChecks++
		if err != nil || stamp.Identity == nil {
			block.dependencies = nil
			return
		}
		block.dependencies = append(block.dependencies, codeDependency{address, stamp})
		if address == last {
			break
		}
	}
	if contextual, ok := c.MemoryBus().(CodeContextMemory); ok {
		block.context = contextual.CodeContext()
		e.Stats.ContextChecks++
	}
}

// A changed physical identity is not necessarily changed parsed semantics (for
// example, an immutable text page remapped into another process). Preserve only
// code whose complete instruction descriptors still match execute-checked reads.
// New page stamps bracket those reads; no old identity or context is authority.
func (e *Engine) revalidateIdenticalBlock(c CPU, pc uint64, block *compiledBlock) bool {
	memory, ok := c.MemoryBus().(CodeContextMemory)
	if !ok || len(block.instructions) == 0 {
		return false
	}
	tail := block.instructions[len(block.instructions)-1]
	first, last := pc&^uint64(4095), (tail.PC+uint64(tail.Length)-1)&^uint64(4095)
	if last < first || last-first > 4096 {
		return false
	}
	var fresh [2]codeDependency
	count := 0
	for address := first; ; address += 4096 {
		stamp, err := memory.CodeVersion(address)
		e.Stats.VersionChecks++
		if err != nil || stamp.Identity == nil {
			return false
		}
		fresh[count] = codeDependency{address, stamp}
		count++
		if address == last {
			break
		}
	}
	context := memory.CodeContext()
	e.Stats.ContextChecks++
	if context.Identity == nil || !e.validBytes(c, pc, block) {
		return false
	}
	for i := 0; i < count; i++ {
		stamp, err := memory.CodeVersion(fresh[i].address)
		e.Stats.VersionChecks++
		if err != nil || stamp != fresh[i].stamp {
			return false
		}
	}
	current := memory.CodeContext()
	e.Stats.ContextChecks++
	if current != context {
		return false
	}
	// A multi-block region can contain other pages/instructions. Reusing its
	// root leaf never proves those constituents. Its immutable arena bytes remain
	// charged until normal recycling; a later region needs ordinary discovery.
	if block.region && block.regionInstructions != len(block.instructions) {
		block.region = false
		block.regionEntry, block.regionInstructions = 0, 0
		block.regionAttempted = false
	}
	block.dependencies = append(block.dependencies[:0], fresh[:count]...)
	block.context = context
	block.prefillDirty = true
	return true
}

// Different blocks in one exact code context can depend on the same physical
// instruction page. Reuse only the full virtual page + actual version stamp +
// complete context. CodeContextMemory already guarantees that mapping, execute
// permission and executable-byte changes revoke that context before reentry.
// This small fixed cache owns no code, pointers into decoded blocks or absence.
type codePageProof struct {
	address        uint64
	stamp, context CodeStamp
}

func (e *Engine) checkCodePage(memory VersionedMemory, dependency codeDependency, context CodeStamp) (bool, bool) {
	proof := &e.versionProofs[((dependency.address>>12)^(dependency.address>>20))&255]
	if context.Identity != nil && proof.context == context && proof.address == dependency.address {
		return proof.stamp == dependency.stamp, false
	}
	stamp, err := memory.CodeVersion(dependency.address)
	e.Stats.VersionChecks++
	valid := err == nil && stamp.Identity != nil && stamp == dependency.stamp
	if context.Identity != nil && err == nil && stamp.Identity != nil {
		// A resolver may itself mutate the mapping while walking. Never turn
		// that result into authority for the context observed before the call.
		owner, ok := memory.(CodeContextMemory)
		if !ok || owner.CodeContext() != context {
			return false, true
		}
		*proof = codePageProof{dependency.address, stamp, context}
	}
	return valid, true
}
