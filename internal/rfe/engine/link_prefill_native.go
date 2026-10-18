//go:build !browser

package engine

import emu "renvo.dev/internal/rfe/runtime"

func (e *Engine) prefillBrowserLinks(c CPU, first *compiledBlock, memory NativeMemory, context *emu.MemoryContext, stamp CodeStamp, published bool) bool {
	return true
}

func (e *Engine) observeBrowserLink(context *emu.MemoryContext, target uint64) {}
