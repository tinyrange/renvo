//go:build !browser

package engine

import emu "renvo.dev/internal/rfe/runtime"

func (e *Engine) endFastRefill(success bool) {}

func (e *Engine) beginFastRefill(memory NativeMemory, context *emu.MemoryContext) bool { return false }
