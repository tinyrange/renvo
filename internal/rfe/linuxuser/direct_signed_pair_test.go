//go:build !renvo && linux && amd64 && cgo

package linuxuser

import (
	emu "renvo.dev/internal/rfe/runtime"
	"testing"
)

func TestDirectSignedLoadThenPairFault(t *testing.T) {
	for _, size := range []int{1, 2, 4} {
		for _, width := range []int{32, 64} {
			m, err := NewDirectMemory()
			if err != nil {
				t.Fatal(err)
			}
			defer m.Close()
			if err = m.Map(4096, 4096, 3); err != nil {
				t.Fatal(err)
			}
			if err = m.Write(4096, size, ^uint64(1)); err != nil {
				t.Fatal(err)
			}
			if err = m.Write(8184, 8, 77); err != nil {
				t.Fatal(err)
			}
			n, err := emu.NewNative(1 << 20)
			if err != nil {
				t.Fatal(err)
			}
			defer n.Close()
			if err = n.PrepareLinks(4, 3); err != nil {
				t.Fatal(err)
			}
			var b emu.Builder
			raw := b.MemoryLoad(b.Load(0), size)
			sign := b.Constant(uint64(1) << uint(size*8-1))
			value := b.Binary(emu.Sub, b.Binary(emu.Xor, raw, sign), sign)
			if width == 32 {
				value = b.Binary(emu.And, value, b.Constant(0xffffffff))
			}
			b.Store(1, value)
			b.Checkpoint(4, 1)
			// The first half is writable, but the second half is unmapped. The
			// checked pair must not borrow a direct hit's unestablished page facts.
			b.MemoryPairStore(b.Load(2), b.Constant(12), b.Constant(34))
			b.Checkpoint(4, 2)
			b.LoopContinue(b.Constant(0))
			entry, err := n.CompileLoopMode(b.FinishMemory(4), 4, 2, true)
			if err != nil {
				t.Fatal(err)
			}
			ctx := m.NativeContext()
			m.NativeRefill(4096)
			ctx.ClaimLinks(n)
			ctx.PublishLink(0, entry, 2, [17]uint8{})
			state := []uint64{4096, 0, 8184, 0}
			if err = n.RunLinkedSession(state, ctx, 64); err != nil {
				t.Fatal(err)
			}
			want := ^uint64(1)
			if width == 32 {
				want &= 0xffffffff
			}
			if ctx.Status != 1 || ctx.Total != 1 || ctx.MemoryTotal != 1 || ctx.Remaining != 63 || ctx.Address != 8184 || state[1] != want {
				t.Fatal("signed result or pair recovery", size, width, state, ctx.Status, ctx.Total, ctx.Address)
			}
			if got, e := m.Read(8184, 8, false); e != nil || got != 77 {
				t.Fatal("partial pair store", got, e)
			}
		}
	}
}
