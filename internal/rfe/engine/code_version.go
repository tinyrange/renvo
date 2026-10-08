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
		stamp, err := memory.CodeVersion(dependency.address)
		e.Stats.VersionChecks++
		if err != nil || stamp.Identity == nil || stamp != dependency.stamp {
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
		if err != nil || current != ins {
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
