//go:build !browser

package engine

import emu "renvo.dev/internal/rfe/runtime"

func (e *Engine) finishBrowserRegionUpgrade(c CPU, context *emu.MemoryContext) bool { return false }

func (e *Engine) regionPublicationHint(block *compiledBlock, generation uint64) uint64 {
	if block.region {
		return ^uint64(0)
	}
	return generation
}
