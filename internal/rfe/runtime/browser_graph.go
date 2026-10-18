//go:build renvo && browser

package runtime

import (
	"fmt"
	"renvo.dev/internal/arena"
)

func (n *Native) CompileCodeVersionedGraph(blocks []GraphBlock, words, pcSlot int, memory *MemoryContext) (int, error) {
	if !n.AdmitCodeVersionedContext(memory) {
		return 0, fmt.Errorf("missing browser graph owner")
	}
	low, high := arena.Mark(), arena.PersistMark()
	code, err := emitWasmGraph(blocks, words, pcSlot, n.layout)
	if err != nil {
		return 0, err
	}
	size := len(code)
	entry, err := n.exchangeWasm(code, words, true)
	if err != nil {
		return 0, err
	}
	arena.Rewind(low)
	arena.PersistReset(high)
	if err = n.recordWasm(entry, words, true, size); err != nil {
		return 0, err
	}
	total := 0
	for _, block := range blocks {
		total += block.Instructions
	}
	n.entryMetadata(entry).loopInstructions = total // linked-only private budget ABI
	n.codeVersionedLoops = true
	return entry, nil
}
