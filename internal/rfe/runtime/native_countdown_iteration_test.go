//go:build !renvo

package runtime

import (
	"reflect"
	"renvo.dev/internal/rfenativebridge"
	"testing"
)

// Scalar progress is independent of either generated counter representation.
// Eight simultaneous loop phis exercise the additional register across timer
// refills, exact-budget tails, a late memory fault, and an expired deadline.
func TestNativeCountdownIterationsAcrossPollsAndFaults(t *testing.T) {
	if !NativeSessionsAvailable() {
		t.Skip("foreign session unavailable")
	}
	const words, pc = 16, 15
	for _, count := range []int{3, 6, 9, 43, 47, 64, 255} {
		n, err := NewNative(4 << 20)
		if err != nil {
			t.Fatal(err)
		}
		if err = n.PrepareLinks(words, pc); err != nil {
			t.Fatal(err)
		}
		m := new(MemoryContext)
		epoch := uint64(1)
		m.Clock = &epoch
		backing := make([]byte, 8192)
		backing[4096] = 0x39
		m.Fill(1, (*[4096]byte)(backing[4096:]), 1, &epoch)
		m.ClaimLinks(n)
		var b Builder
		values := make([]Value, 8)
		for i := range values {
			values[i] = b.Load(i)
		}
		for i := range values {
			b.Store(i, b.Binary(Add, values[(i+1)%len(values)], b.Constant(uint64(i+17))))
		}
		next := b.Binary(Add, b.Load(10), b.Constant(1))
		b.Store(10, next)
		b.Checkpoint(words, 1)
		at := b.Load(8)
		b.Store(12, b.MemoryLoad(at, 1))
		b.Store(8, b.Binary(Add, at, b.Load(9)))
		condition := b.Binary(Less, next, b.Load(11))
		b.Store(pc, b.Choose(condition, b.Constant(0), b.Constant(4)))
		b.Checkpoint(words, count)
		b.LoopContinue(condition)
		entry, err := n.CompileCodeVersionedLoop(b.FinishMemory(words), words, count, m)
		if err != nil {
			t.Fatal(err)
		}
		if err = n.PrepareTargetLink(0, entry, words, count); err != nil {
			t.Fatal(err)
		}
		m.PublishLink(0, entry, count, [17]uint8{})
		for _, step := range []uint64{0, 8} {
			for _, limit := range []uint64{1, 7, 1000000} {
				for _, budget := range []uint64{uint64(count), uint64(count + 1), 63, 64, 65, 127, 4095, 4096, 4097, 8191, 8192, 8193, 65535, 65536} {
					state := make([]uint64, words)
					for i := 0; i < 8; i++ {
						state[i] = uint64(0xfedcba9876543210) + uint64(i)
					}
					state[8], state[9], state[11], state[12] = 4096, step, limit, 0xbeef
					want := append([]uint64(nil), state...)
					retired, memories, iterations, status := uint64(0), uint64(0), uint64(0), uint64(0)
					for retired+uint64(count) <= budget {
						old := append([]uint64(nil), want[:8]...)
						for i := 0; i < 8; i++ {
							want[i] = old[(i+1)%8] + uint64(i+17)
						}
						want[10]++
						if want[8] >= 8192 {
							retired++
							status = 1
							break
						}
						want[12] = uint64(backing[want[8]])
						want[8] += step
						retired += uint64(count)
						memories++
						iterations++
						if want[10] >= limit {
							want[pc] = 4
							break
						}
					}
					expired, err := n.RunLinkedBackingSessionUntil(state, m, budget, backing, rfenativebridge.MonotonicNS()+1000000000)
					if err != nil || expired || !reflect.DeepEqual(state, want) || m.Total != retired || m.Remaining != budget-retired || m.Status != status || m.MemoryTotal != memories || m.LoopIterations != iterations {
						t.Fatalf("count=%d step=%d limit=%d budget=%d err=%v expired=%t got=%x want=%x total=%d/%d status=%d/%d memory=%d/%d iterations=%d/%d", count, step, limit, budget, err, expired, state, want, m.Total, retired, m.Status, status, m.MemoryTotal, memories, m.LoopIterations, iterations)
					}
				}
			}
		}
		state := make([]uint64, words)
		state[8], state[11] = 4096, 1000000
		expired, err := n.RunLinkedBackingSessionUntil(state, m, 65536, backing, rfenativebridge.MonotonicNS()+50000)
		if err != nil || m.Status != 0 || m.Total%uint64(count) != 0 || m.Total+m.Remaining != 65536 || state[10] != m.Total/uint64(count) || m.LoopIterations != state[10] || m.MemoryTotal != state[10] {
			t.Fatalf("expiry count=%d err=%v expired=%t state=%v total=%d remaining=%d iterations=%d", count, err, expired, state, m.Total, m.Remaining, m.LoopIterations)
		}
		n.Close()
	}
}
