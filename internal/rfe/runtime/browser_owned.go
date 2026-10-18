//go:build renvo && browser

package runtime

// Until owned direct addressing is admitted, keep checked page accesses.
func (n *Native) CompileOwnedDirectLoop(ops []Op, words, instructions int, memory *MemoryContext) (int, error) {
	return n.CompileCodeVersionedLoop(ops, words, instructions, memory)
}
