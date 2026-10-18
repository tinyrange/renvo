//go:build !browser

package engine

import emu "renvo.dev/internal/rfe/runtime"

func (e *Engine) rememberNativeRecycle(old *emu.Native) {}
func beginNativeRecycle(old *emu.Native) error          { return old.Close() }
func resetRegionForRecycle(block *compiledBlock) {
	block.regionEntry, block.regionInstructions, block.regionPrefix = 0, 0, [17]uint8{}
}
