//go:build !renvo

package runtime

import (
	"renvo.dev/internal/rfenativebridge"
	"testing"
)

func TestNativeResidentTimerMixedLoops(t *testing.T) {
	if !NativeSessionsAvailable() || rfenativebridge.MonotonicNS() == 0 {
		t.Skip("foreign timers unavailable")
	}
	for _, prepared := range []bool{false, true} {
		n, err := NewNative(1 << 20)
		if err != nil {
			t.Fatal(err)
		}
		if err = n.PrepareLinks(4, 2); err != nil {
			t.Fatal(err)
		}
		entries := [2]int{}
		for i := range entries {
			var b Builder
			if i == 0 {
				b.Store(0, b.Binary(Add, b.Load(0), b.Constant(1)))
			} else {
				b.MemoryStore(b.Constant(4096), b.Load(0), 8)
				b.Store(1, b.Load(0))
			}
			b.Store(2, b.Constant(uint64((1-i)*4)))
			b.Store(3, b.Elapsed(b.Load(3), 1, 1))
			b.Checkpoint(4, 1)
			b.LoopContinue(b.Constant(0))
			entries[i], err = n.CompileCodeVersionedLoop(b.FinishMemory(4), 4, 1, new(MemoryContext))
			if err != nil {
				t.Fatal(err)
			}
			if prepared {
				if err = n.PrepareTargetLink(uint64(i*4), entries[i], 4, 1); err != nil {
					t.Fatal(err)
				}
			}
		}
		backing := make([]byte, 8192)
		backing[4096] = 37
		epoch := uint64(1)
		m := new(MemoryContext)
		m.Clock = &epoch
		m.Fill(1, (*[4096]byte)(backing[4096:]), 3, &epoch)
		m.ClaimLinks(n)
		for i := range entries {
			m.PublishLink(uint64(i*4), entries[i], 1, [17]uint8{0, uint8(i)})
		}
		n.AdmitCodeVersionedContext(m)
		for _, budget := range []uint64{63, 64, 65, 127, 128, 129, 1001, 65536} {
			state := []uint64{0, 99, 0, 0}
			hit, err := n.RunLinkedBackingSessionUntil(state, m, budget, backing, rfenativebridge.MonotonicNS()+1000000000)
			if err != nil || hit || m.Total != budget || m.Remaining != 0 || m.MemoryTotal != budget/2 || state[0] != (budget+1)/2 || state[1] != budget/2 || state[2] != (budget%2)*4 || state[3] != budget*(budget+1)/2 {
				t.Fatalf("prepared=%v budget=%d hit=%v err=%v total=%d memory=%d state=%v", prepared, budget, hit, err, m.Total, m.MemoryTotal, state)
			}
		}
		// A denied write after a completed pure instruction returns its exact
		// prefix once, without replaying the pure side effect or a timer slice.
		m.Pages[1].Permissions = 0
		state := []uint64{0, 99, 0, 0}
		hit, err := n.RunLinkedBackingSessionUntil(state, m, 65536, backing, rfenativebridge.MonotonicNS()+1000000000)
		if err != nil || hit || m.Total != 1 || m.Remaining != 65535 || m.MemoryTotal != 0 || m.Status != 1 || m.Address != 4096 || state[0] != 1 || state[1] != 99 || state[2] != 4 || state[3] != 1 {
			t.Fatal("precise fault after progress", err, hit, state, m.Total, m.Status, m.Address)
		}
		if m.TimerFunction != 0 || m.TimerDeadline != 0 || m.TimerExpired != 0 {
			t.Fatal("timer borrow survived fault")
		}
		if err = n.Close(); err != nil {
			t.Fatal(err)
		}
	}
}
