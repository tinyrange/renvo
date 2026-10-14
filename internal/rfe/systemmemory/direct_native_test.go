//go:build !renvo && linux && amd64 && cgo

package systemmemory

import (
	"encoding/binary"
	"fmt"
	emu "renvo.dev/internal/rfe/runtime"
	"runtime"
	"testing"
)

type directHarness struct {
	m        *RAM
	n        *emu.Native
	root     uint64
	domain   int
	physical uint64
	writable bool
	invalid  bool
	base     uint64
}

func newDirectHarness(t *testing.T, base uint64) *directHarness {
	t.Helper()
	h := &directHarness{writable: true, base: base}
	m, err := NewDirect(4*4096, func() bool { return true }, func(a uint64, x bool) (uint64, bool, error) {
		if h.invalid || a < h.base || a-h.base >= 8192 {
			return 0, false, fmt.Errorf("unmapped")
		}
		return h.physical + a - h.base, h.writable, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	h.m = m
	m.AddressSpace = func() uint64 { return h.root }
	m.Domain = func() int { return h.domain }
	if m.direct == nil {
		_ = m.Close()
		t.Skip("fault-backed RAM unavailable")
	}
	n, err := emu.NewNative(1 << 20)
	if err != nil {
		_ = m.Close()
		t.Fatal(err)
	}
	h.n = n
	t.Cleanup(func() {
		n.Close()
		if err := m.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := n.PrepareLinks(3, 0); err != nil {
		t.Fatal(err)
	}
	var b emu.Builder
	a := b.Load(1)
	b.Store(2, b.MemoryLoad(a, 8))
	b.Checkpoint(3, 1)
	b.MemoryStore(a, b.Binary(emu.Add, b.Load(2), b.Constant(1)), 8)
	b.Checkpoint(3, 2)
	b.LoopContinue(b.Constant(1))
	ctx := m.NativeContext()
	entry, err := n.CompileOwnedDirectLoop(b.FinishMemory(3), 3, 2, ctx)
	if err != nil {
		t.Fatal(err)
	}
	ctx.ClaimLinks(n)
	ctx.PublishLink(0, entry, 2, [17]uint8{})
	return h
}
func (h *directHarness) run(t *testing.T, address, budget uint64) ([]uint64, *emu.MemoryContext) {
	t.Helper()
	ctx := h.m.NativeContext()
	state := []uint64{0, address, 999}
	if err := h.n.RunLinkedBackingSession(state, ctx, budget, h.m.NativeBacking()); err != nil {
		t.Fatal(err)
	}
	return state, ctx
}
func TestDirectRAMNativeAliasRemapAndRevoke(t *testing.T) {
	for _, base := range []uint64{4096, 0xffff00001000} {
		h := newDirectHarness(t, base)
		h.m.NativeRefill(base + 7)
		if h.m.NativeContext().DirectSize != directWindowSize {
			t.Fatal("no direct view")
		}
		clock := h.m.clock
		state, ctx := h.run(t, base, 256)
		if ctx.Status != 0 || ctx.Total != 256 || ctx.MemoryTotal != 256 || state[2] != 127 || binary.LittleEndian.Uint64(h.m.Data) != 128 || h.m.clock != clock {
			t.Fatal("incorrect direct loop", state, ctx.Total, ctx.Status)
		}
		// A checked physical alias and native VA must observe the same bytes.
		binary.LittleEndian.PutUint64(h.m.Data, 400)
		runtime.GC()
		state, ctx = h.run(t, base, 2)
		if state[2] != 400 || ctx.Total != 2 || binary.LittleEndian.Uint64(h.m.Data) != 401 {
			t.Fatal("physical alias/GC lost")
		}
		// Revoke every view before publishing a new physical page at this VA.
		h.physical = 4096
		h.m.InvalidateMappings()
		if ctx.DirectSize != 0 || ctx.Pages[base>>12&63].Data != nil {
			t.Fatal("stale remap authority")
		}
		// Refilling a different page reactivates the window, but must NOT
		// reauthorize its old physical alias. Force a real direct load fault.
		h.m.NativeRefill(base + 4096)
		state, ctx = h.run(t, base, 256)
		if ctx.Status != 1 || ctx.Total != 0 || ctx.Address != base || state[2] != 999 {
			t.Fatal("stale window page survived revoke", state, ctx.Status, ctx.Total)
		}
		binary.LittleEndian.PutUint64(h.m.Data[4096:], 50)
		h.m.NativeRefill(base)
		state, ctx = h.run(t, base, 2)
		if ctx.Status != 0 || state[2] != 50 || binary.LittleEndian.Uint64(h.m.Data[4096:]) != 51 || binary.LittleEndian.Uint64(h.m.Data) != 401 {
			t.Fatal("wrong remapped physical page", state)
		}
		// Root and privilege switches clear ALL descriptors/direct fields.
		old := ctx
		h.root++
		ctx = h.m.NativeContext()
		if old.DirectSize != 0 || old.Pages[base>>12&63].Data != nil || ctx.DirectSize != 0 {
			t.Fatal("root leaked aliases")
		}
		h.m.NativeRefill(base)
		h.domain = 1
		ctx = h.m.NativeContext()
		if old.DirectSize != 0 || ctx.DirectSize != 0 {
			t.Fatal("domain leaked aliases")
		}
	}
}
func TestDirectRAMNativeFaultsAndBarriers(t *testing.T) {
	for _, barrier := range []string{"readonly", "code", "page-table", "invalid", "cross-page", "cross-window"} {
		t.Run(barrier, func(t *testing.T) {
			base := uint64(4096)
			if barrier == "cross-window" {
				base = directWindowSize - 4096
			}
			h := newDirectHarness(t, base)
			binary.LittleEndian.PutUint64(h.m.Data, 73)
			h.m.NativeRefill(base)
			switch barrier {
			case "readonly":
				h.writable = false
				h.m.InvalidateMappings()
				h.m.NativeRefill(base)
			case "code":
				if _, err := h.m.CodeVersion(base); err != nil {
					t.Fatal(err)
				}
				h.m.NativeRefill(base)
			case "page-table":
				h.m.MarkPageTable(0)
				h.m.NativeRefill(base)
			case "invalid":
				h.invalid = true
				h.m.NativeRefill(base)
			}
			address, wantTotal, wantRead := base, uint64(1), uint64(73)
			if barrier == "invalid" {
				wantTotal, wantRead = 0, 999
			}
			if barrier == "cross-page" || barrier == "cross-window" {
				address = base + 4092
				wantTotal, wantRead = 0, 999
			}
			state, ctx := h.run(t, address, 256)
			if ctx.Status != 1 || ctx.Address != address || ctx.Total != wantTotal || ctx.MemoryTotal != wantTotal || ctx.Remaining != 256-wantTotal || state[2] != wantRead || binary.LittleEndian.Uint64(h.m.Data) != 73 {
				t.Fatal("imprecise fault/barrier", barrier, state, ctx.Status, ctx.Address, ctx.Total, ctx.MemoryTotal)
			}
		})
	}
}
func TestDirectRAMBoundedWindowsAndClose(t *testing.T) {
	h := newDirectHarness(t, 0)
	d := h.m.direct.(*mappedRAM)
	for i := uint64(0); i < 9; i++ {
		d.refill(i*directWindowSize, 0, 3)
	}
	if len(d.windows) != 8 {
		t.Fatal("reservation budget", len(d.windows))
	}
	d.refill(1<<48, 0, 3)
	d.refill(0, uint64(len(h.m.Data)), 3)
	if len(d.windows) != 8 {
		t.Fatal("invalid extent/key admitted")
	}
	ctx := h.m.NativeContext()
	h.n.Close()
	if err := h.m.Close(); err != nil {
		t.Fatal(err)
	}
	if ctx.DirectSize != 0 || ctx.DirectHost != 0 || h.m.Data != nil || h.m.NativeContext() != nil {
		t.Fatal("closed owner retained authority")
	}
}
