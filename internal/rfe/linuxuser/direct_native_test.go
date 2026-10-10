//go:build !renvo && linux && amd64 && cgo

package linuxuser

import (
	emu "renvo.dev/internal/rfe/runtime"
	"testing"
)

func TestDirectNativeFaultRecovery(t *testing.T) {
	for _, writeProtected := range []bool{false, true} {
		for _, compatibility := range []bool{false, true} {
			m, err := NewDirectMemory()
			if err != nil {
				t.Fatal(err)
			}
			defer m.Close()
			perm := uint8(3)
			if writeProtected {
				perm = 1
			}
			if err = m.Map(4096, 4096, perm); err != nil {
				t.Fatal(err)
			}
			m.NativeContext()
			m.NativeRefill(8184)
			if m.NativeContext().DirectSize != DirectWindowSize {
				t.Fatal("no mapped window")
			}
			n, err := emu.NewNative(1 << 20)
			if err != nil {
				t.Fatal(err)
			}
			defer n.Close()
			if err = n.PrepareLinks(3, 2); err != nil {
				t.Fatal(err)
			}
			var b emu.Builder
			a := b.Load(0)
			b.Store(1, b.MemoryLoad(a, 8))
			b.Checkpoint(3, 1)
			b.MemoryStore(a, b.Constant(0x12345678), 8)
			b.Store(0, b.Binary(emu.Add, a, b.Constant(8)))
			b.Checkpoint(3, 2)
			b.LoopContinue(b.Constant(1))
			entry, err := n.CompileLoopMode(b.FinishMemory(3), 3, 2, true)
			if err != nil {
				t.Fatal(err)
			}
			ctx := m.NativeContext()
			ctx.ClaimLinks(n)
			ctx.PublishLink(0, entry, 2, [17]uint8{})
			before := m.versions.epoch
			state := []uint64{8184, 99, 0}
			if compatibility {
				err = n.CallLinked(state, ctx, 64)
			} else {
				err = n.RunLinkedSession(state, ctx, 64)
			}
			if err != nil {
				t.Fatal(err)
			}
			wantAddress, wantTotal, wantLoops, wantValue := uint64(8192), uint64(2), uint64(1), uint64(0x12345678)
			if writeProtected {
				wantAddress, wantTotal, wantLoops, wantValue = 8184, 1, 0, 0
			}
			value, readErr := m.Read(8184, 8, false)
			if readErr != nil || value != wantValue || state[0] != wantAddress || state[1] != 0 || ctx.Status != 1 || ctx.Address != wantAddress || ctx.Total != wantTotal || ctx.MemoryTotal != wantTotal || ctx.LoopIterations != wantLoops || ctx.Remaining != 64-wantTotal {
				t.Fatal("imprecise direct fault", writeProtected, compatibility, state, value, ctx.Status, ctx.Address, ctx.Total, ctx.MemoryTotal, ctx.LoopIterations, ctx.Remaining)
			}
			if !compatibility && m.versions.epoch != before {
				t.Fatal("ordinary direct data store allocated a code generation")
			}
		}
	}
}
