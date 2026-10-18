//go:build !renvo

package engine

import (
	emu "renvo.dev/internal/rfe/runtime"
	"testing"
)

type changingEntrySnapshot struct {
	*prefillMemory
	mode    string
	queries int
}

func (m *changingEntrySnapshot) NativeCodeContext() (CodeStamp, *emu.MemoryContext) {
	m.queries++
	if m.mode == "epoch" {
		m.stamp.Epoch++
	}
	if m.mode == "identity" {
		m.stamp.Identity = new(CodeIdentity)
	}
	if m.mode == "no-authority" {
		return CodeStamp{}, &m.context
	}
	if m.mode == "no-context" {
		return m.stamp, nil
	}
	if m.mode == "fresh-context" {
		m.alternate = new(emu.MemoryContext)
	}
	return m.stamp, m.NativeContext()
}
func TestLinkedEntryUsesFreshCoherentSnapshot(t *testing.T) {
	for _, mode := range []string{"stable", "epoch", "identity", "no-authority", "no-context", "fresh-context"} {
		t.Run(mode, func(t *testing.T) {
			e, c, m := prefillFixture(t)
			first := e.blocks[0]
			first.context = m.stamp
			memory := &changingEntrySnapshot{prefillMemory: m, mode: mode}
			c.memory = memory
			before := c.state
			block, owner, context := e.linkedSnapshot(c, 64)
			valid := mode == "stable" || mode == "fresh-context"
			if (block != nil) != valid || memory.queries != 1 || c.state != before || c.retired != 0 || m.reads != 0 {
				t.Fatal("stale snapshot admitted or guest effects during observation", mode, block, memory.queries, m.reads)
			}
			if valid && (block != first || owner != memory || context != memory.NativeContext()) {
				t.Fatal("not the exact fresh owner/context", mode)
			}
		})
	}
}
