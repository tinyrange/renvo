//go:build !browser

package engine

import (
	"fmt"
	emu "renvo.dev/internal/rfe/runtime"
)

func (e *Engine) nameEntry(entry int, pc uint64, count int, kind string) {
	unit := "ops"
	if kind == "region" {
		unit = "instructions"
	}
	_ = e.native.NameEntry(entry, fmt.Sprintf("%s_%x_%s_%d_%s", e.arch.Name, pc, kind, count, unit))
}

func nativeEntryBudgetNeedsRecycle(n *emu.Native) bool { return false }

func (e *Engine) recycleNative(old *emu.Native, capacity int) (*emu.Native, error) {
	return emu.NewNative(capacity)
}

func (e *Engine) publishTranslation(block *compiledBlock, ops []emu.Op) []emu.Op {
	e.Stats.LeafTranslations++
	return ops
}
func (e *Engine) dropTranslation(block *compiledBlock)    {}
func (e *Engine) releaseTranslation(block *compiledBlock) {}

func (e *Engine) beginTranslation(progress bool) (*emu.Builder, int, int) {
	b := &e.translation
	b.Reset(progress)
	return b, 0, 0
}
func (e *Engine) endTranslation(low, high int) {}
func (e *Engine) finishTranslation(b *emu.Builder, memory bool, low, high int) []emu.Op {
	if memory {
		return b.FinishMemory(e.arch.StateWords)
	}
	return b.Finish(e.arch.StateWords)
}

func (e *Engine) reuseTranslation(block *compiledBlock) { e.Stats.LeafIRReuses++ }
