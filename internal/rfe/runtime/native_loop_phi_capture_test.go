//go:build !renvo

package runtime

import (
	"reflect"
	"testing"
)

func TestNativeLoopInheritedPhiSurvival(t *testing.T) {
	for _, pressure := range []int{0, 16} {
		words := 6 + pressure
		pc := words - 1
		n, err := NewNative(1 << 20)
		if err != nil {
			t.Fatal(err)
		}
		defer n.Close()
		if err = n.PrepareLinks(words, pc); err != nil {
			t.Fatal(err)
		}
		var b Builder
		first, second := b.Load(0), b.Load(3)
		extra := make([]Value, pressure)
		for i := range extra {
			extra[i] = b.Load(5 + i)
		}
		b.Store(1, b.MemoryLoad(first, 1))
		next := b.Binary(Add, first, b.Constant(1))
		b.Store(0, next)
		b.Checkpoint(words, 1)
		b.Store(4, b.MemoryLoad(second, 8))
		b.Store(3, b.Binary(Add, second, b.Constant(8)))
		b.Checkpoint(words, 2)
		b.Store(2, b.StatusBits(next, 8))
		for i, value := range extra {
			b.Store(5+i, b.Binary(Add, value, b.Constant(uint64(i+1))))
		}
		b.Store(pc, b.Constant(0))
		b.Checkpoint(words, 3)
		b.LoopContinue(b.Constant(1))
		entry, err := n.CompileLoop(b.FinishMemory(words), words, 3)
		if err != nil {
			t.Fatal(err)
		}
		// Fault before the phi is overwritten, after a partial next iteration,
		// and before any complete iteration. High register pressure also covers
		// the spill fallback rather than assuming every phi stays in a register.
		for _, addresses := range [][2]uint64{{8191, 4096}, {4101, 8184}, {8192, 4096}, {4101, 8192}} {
			for _, budget := range []uint64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 64} {
				state := make([]uint64, words)
				state[0], state[1], state[2], state[3], state[4] = addresses[0], 99, 0xa5, addresses[1], 98
				for i := 0; i < pressure; i++ {
					state[5+i] = uint64(i + 17)
				}
				want := append([]uint64(nil), state...)
				retired, memories, completed, status, fault := uint64(0), uint64(0), uint64(0), uint64(0), uint64(0)
				for budget-retired >= 3 {
					if want[0] < 4096 || want[0] >= 8192 {
						status, fault = 1, want[0]
						break
					}
					want[1], want[0] = 0, want[0]+1
					retired++
					memories++
					if want[3] < 4096 || want[3] > 8184 {
						status, fault = 1, want[3]
						break
					}
					want[4], want[3] = 0, want[3]+8
					retired++
					memories++
					want[2] = statusOracle(want[0], 0, 8, false) & 0xc4
					for i := 0; i < pressure; i++ {
						want[5+i] += uint64(i + 1)
					}
					retired++
					completed++
				}
				m, page, epoch := new(MemoryContext), new([4096]byte), uint64(1)
				m.Fill(1, page, 1, &epoch)
				m.ClaimLinks(n)
				m.PublishLink(0, entry, 3, [17]uint8{})
				if err = n.CallLinked(state, m, budget); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(state, want) || m.Total != retired || m.MemoryTotal != memories || m.LoopIterations != completed || m.Status != status || m.Remaining != budget-retired || status != 0 && m.Address != fault {
					t.Fatal("inherited phi mismatch", pressure, addresses, budget, state, want, m.Total, retired, m.Status, status)
				}
			}
		}
	}
}
