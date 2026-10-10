//go:build !renvo && (amd64 || arm64)

package linuxuser

import (
	emu "renvo.dev/internal/rfe/runtime"
	"testing"
)

func TestCodeVersionedNativeStoreTransitions(t *testing.T) {
	for _, pair := range []bool{false, true} {
		m := NewMemory()
		defer m.Close()
		if err := m.Map(4096, 4096, 3); err != nil {
			t.Fatal(err)
		}
		ctx := m.NativeContext()
		m.NativeRefill(4096)
		n, err := emu.NewNative(1 << 20)
		if err != nil {
			t.Fatal(err)
		}
		defer n.Close()
		if err = n.PrepareLinks(4, 3); err != nil {
			t.Fatal(err)
		}
		var b emu.Builder
		address := b.Load(0)
		if pair {
			b.MemoryPairStore(address, b.Load(1), b.Load(2))
		} else {
			b.MemoryStore(address, b.Load(1), 8)
		}
		b.Store(3, b.Constant(4))
		b.Checkpoint(4, 1)
		b.LoopContinue(b.Constant(0))
		entry, err := n.CompileCodeVersionedLoop(b.FinishMemory(4), 4, 1, ctx)
		if err != nil {
			t.Fatal(err)
		}
		ctx.ClaimLinks(n)
		run := func(address, value uint64, session bool) {
			t.Helper()
			m.NativeRefill(address)
			ctx.PublishLink(0, entry, 1, [17]uint8{})
			state := []uint64{address, value, value + 1, 0}
			if session {
				err = n.RunLinkedSession(state, ctx, 64)
			} else {
				err = n.CallLinked(state, ctx, 64)
			}
			if err != nil {
				t.Fatal(err)
			}
		}
		for _, session := range []bool{false, true} {
			before := m.versions.epoch
			run(4096, 41, session)
			got, e := m.Read(4096, 8, false)
			if e != nil || got != 41 || ctx.Total != 1 || ctx.MemoryTotal != 1 || ctx.Status != 0 || ctx.Remaining != 63 || m.versions.epoch != before || m.pages[1].epoch != before {
				t.Fatal("data store changed code generation or progress", pair, session, got, e, ctx.Total, ctx.Status, m.versions.epoch, before)
			}
			if pair {
				if got, e = m.Read(4104, 8, false); e != nil || got != 42 {
					t.Fatal(got, e)
				}
			}
		}
		// Every protection transition forgets descriptors and establishes a new
		// generation. RWX stores must leave native execution before changing bytes.
		before := m.versions.epoch
		if err = m.Protect(4096, 4096, 7); err != nil {
			t.Fatal(err)
		}
		executable, e := m.CodeVersion(4096)
		if e != nil || executable.Epoch <= before {
			t.Fatal("data-to-code generation", executable, e)
		}
		run(4096, 99, true)
		got, e := m.Read(4096, 8, false)
		if e != nil || got != 41 || ctx.Status != 1 || ctx.Total != 0 || ctx.Address != 4096 || m.CodeContext().Epoch != executable.Epoch {
			t.Fatal("executable native store was not stopped", got, e, ctx.Status, ctx.Total)
		}
		if err = m.Write(4096, 8, 99); err != nil {
			t.Fatal(err)
		}
		changed, e := m.CodeVersion(4096)
		if e != nil || changed.Epoch <= executable.Epoch {
			t.Fatal("executable write not invalidated", changed, e)
		}
		if err = m.Protect(4096, 4096, 3); err != nil {
			t.Fatal(err)
		}
		before = m.versions.epoch
		run(4096, 123, true)
		if ctx.Status != 0 || m.versions.epoch != before {
			t.Fatal("data store after code transition", ctx.Status, m.versions.epoch, before)
		}
		if err = m.Protect(4096, 4096, 5); err != nil {
			t.Fatal(err)
		}
		changed, e = m.CodeVersion(4096)
		if e != nil || changed.Epoch <= before {
			t.Fatal("second executable transition reused generation", changed, e)
		}
		if err = m.Unmap(4096, 4096); err != nil {
			t.Fatal(err)
		}
		run(4096, 456, true)
		if ctx.Status != 1 || ctx.Total != 0 {
			t.Fatal("unmapped descriptor survived")
		}
		if err = m.Map(4096, 4096, 3); err != nil {
			t.Fatal(err)
		}
		if got, e = m.Read(4096, 8, false); e != nil || got != 0 {
			t.Fatal("remap retained data", got, e)
		}
		run(4096, 789, true)
		if ctx.Status != 0 {
			t.Fatal("remapped store failed")
		}
		before = m.versions.epoch
		run(8191, 999, true)
		if ctx.Status != 1 || ctx.Total != 0 || m.versions.epoch != before {
			t.Fatal("cross-page store partially retired")
		}
		if got, e = m.Read(8184, 8, false); e != nil || got != 0 {
			t.Fatal("cross-page store partially committed", got, e)
		}
		run(^uint64(0)-7, 999, true)
		if ctx.Status != 1 || ctx.Total != 0 {
			t.Fatal("invalid full-width address admitted")
		}
	}
}

func TestCodeVersionedNativeMixedGenericChain(t *testing.T) {
	m := NewMemory()
	defer m.Close()
	if err := m.Map(4096, 4096, 3); err != nil {
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
	makeOps := func(value, next uint64) []emu.Op {
		var b emu.Builder
		b.MemoryStore(b.Load(0), b.Constant(value), 8)
		b.Store(2, b.Constant(next))
		b.Checkpoint(3, 1)
		b.LoopContinue(b.Constant(0))
		return b.FinishMemory(3)
	}
	owned, err := n.CompileCodeVersionedLoop(makeOps(11, 4), 3, 1, ctx)
	if err != nil {
		t.Fatal(err)
	}
	generic, err := n.CompileLoop(makeOps(22, 8), 3, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx.ClaimLinks(n)
	ctx.PublishLink(0, owned, 1, [17]uint8{})
	ctx.PublishLink(4, generic, 1, [17]uint8{})
	before := m.versions.epoch
	state := []uint64{4096, 0, 0}
	if err = n.RunLinkedSession(state, ctx, 64); err != nil {
		t.Fatal(err)
	}
	got, e := m.Read(4096, 8, false)
	if e != nil || got != 22 || state[2] != 8 || ctx.Total != 2 || ctx.MemoryTotal != 2 || m.versions.epoch != before+1 || m.pages[1].epoch != before+1 {
		t.Fatal("shared write fact broke generic generation update", got, e, state, ctx.Total, m.versions.epoch, before)
	}
	// A public link is not authority to use the owner's specialized store policy.
	other := new(emu.MemoryContext)
	other.ClaimLinks(n)
	other.PublishLink(0, owned, 1, [17]uint8{})
	state = []uint64{4096, 0, 0}
	if err = n.CallLinked(state, other, 64); err == nil {
		t.Fatal("unregistered compatibility context admitted")
	}
	if err = n.RunLinkedSession(state, other, 64); err == nil {
		t.Fatal("unregistered session admitted")
	}
	if _, err = n.RunLinkedQuanta(state, other, 1, 64, 0); err == nil {
		t.Fatal("unregistered quanta admitted")
	}
}

func TestCodeVersionedInvariantLoopAtExhaustedClock(t *testing.T) {
	for _, pair := range []bool{false, true} {
		m := NewMemory()
		defer m.Close()
		if err := m.Map(4096, 4096, 3); err != nil {
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
		address, value := b.Load(0), b.Load(1)
		if pair {
			b.MemoryPairStore(address, value, b.Binary(emu.Add, value, b.Constant(1)))
		} else {
			b.MemoryStore(address, value, 8)
		}
		b.Store(1, b.Binary(emu.Add, value, b.Constant(2)))
		b.Checkpoint(3, 1)
		b.LoopContinue(b.Constant(1))
		entry, err := n.CompileCodeVersionedLoop(b.FinishMemory(3), 3, 1, ctx)
		if err != nil {
			t.Fatal(err)
		}
		ctx.ClaimLinks(n)
		ctx.PublishLink(0, entry, 1, [17]uint8{})
		m.versions.epoch = ^uint64(0)
		for _, budget := range []uint64{1, 2, 63, 64, 127} {
			state := []uint64{4096, 10, 0}
			if err = n.RunLinkedSession(state, ctx, budget); err != nil {
				t.Fatal(err)
			}
			got, e := m.Read(4096, 8, false)
			if e != nil || got != 10+(budget-1)*2 || state[1] != 10+budget*2 || ctx.Status != 0 || ctx.Total != budget || ctx.MemoryTotal != budget || ctx.Remaining != 0 || m.versions.epoch != ^uint64(0) || m.pages[1].epoch != 1 {
				t.Fatal("invariant loop progress/generation", pair, budget, state, got, e, ctx.Status, ctx.Total, ctx.MemoryTotal)
			}
			if pair {
				if got, e = m.Read(4104, 8, false); e != nil || got != 11+(budget-1)*2 {
					t.Fatal(got, e)
				}
			}
		}
		// Data can still change, but exhausted code generations must never permit
		// that data to become executable under a reused stamp.
		if err = m.Protect(4096, 4096, 5); err == nil {
			t.Fatal("exhausted generation allowed executable transition")
		}
		if _, err = m.CodeVersion(4096); err == nil {
			t.Fatal("failed transition changed permissions")
		}
	}
}
