//go:build renvo

package engine

import (
	emu "renvo.dev/internal/rfe/runtime"
	"testing"
)

func TestPortableEngineRejectsAdaptiveWithoutHostClock(t *testing.T) {
	arch := Architecture{Name: "clock-contract", StateWords: 1, Alignment: 1,
		Fetch: func(Memory, uint64) (Instruction, error) { return Instruction{}, nil },
		Lower: func(*emu.Builder, Instruction, int) bool { return false }}
	config := DefaultEngineConfig()
	config.Mode = "ir"
	engine, err := New(config, arch)
	if err != nil {
		t.Fatal("portable IR construction", err)
	}
	if err = engine.Close(); err != nil {
		t.Fatal(err)
	}
	config.Adaptive = true
	if engine, err = New(config, arch); err == nil || engine != nil {
		t.Fatal("adaptive promotion accepted without a monotonic host clock")
	}
}
