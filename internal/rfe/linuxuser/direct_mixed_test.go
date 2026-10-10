//go:build !renvo && linux && amd64 && cgo

package linuxuser

import (
	emu "renvo.dev/internal/rfe/runtime"
	"testing"
)

func TestDirectLateWindowDoesNotExposeFallbackPage(t *testing.T) {
	m, err := NewDirectMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	d := m.versions.direct.(*mappedMemory)
	// Simulate a failed reservation without depending on host memory pressure.
	d.closed = true
	if err = m.Map(4096, 4096, 3); err != nil {
		t.Fatal(err)
	}
	d.closed = false
	if err = m.Write(4096, 8, 123); err != nil {
		t.Fatal(err)
	}
	if err = m.Map(8192, 4096, 3); err != nil {
		t.Fatal(err)
	}
	// This must not enable access to the unrelated zero-filled memfd page.
	if err = m.Protect(4096, 4096, 3); err != nil {
		t.Fatal(err)
	}
	ctx := m.NativeContext()
	m.NativeRefill(4096)
	n, err := emu.NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(3, 2); err != nil {
		t.Fatal(err)
	}
	var b emu.Builder
	b.Store(1, b.MemoryLoad(b.Load(0), 8))
	b.Checkpoint(3, 1)
	b.LoopContinue(b.Constant(0))
	entry, err := n.CompileLoopMode(b.FinishMemory(3), 3, 1, true)
	if err != nil {
		t.Fatal(err)
	}
	ctx.ClaimLinks(n)
	ctx.PublishLink(0, entry, 1, [17]uint8{})
	state := []uint64{4096, 99, 0}
	if err = n.RunLinkedSession(state, ctx, 64); err != nil {
		t.Fatal(err)
	}
	if ctx.Status != 1 || ctx.Total != 0 || ctx.Address != 4096 || state[1] != 99 {
		t.Fatal("fallback page exposed wrong backing", ctx.Status, ctx.Total, ctx.Address, state)
	}
	if got, e := m.Read(4096, 8, false); e != nil || got != 123 {
		t.Fatal(got, e)
	}
}
