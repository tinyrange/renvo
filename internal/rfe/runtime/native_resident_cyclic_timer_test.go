//go:build !renvo

package runtime

import (
	"renvo.dev/internal/rfenativebridge"
	"testing"
)

func TestNativeResidentCyclicTimerBudgetAndTransitions(t *testing.T) {
	for _, counts := range [][2]int{{3, 7}, {43, 47}} {
		testNativeResidentCyclicTimerBudgetAndTransitions(t, counts)
	}
}
func testNativeResidentCyclicTimerBudgetAndTransitions(t *testing.T, counts [2]int) {
	if !NativeSessionsAvailable() || rfenativebridge.MonotonicNS() == 0 {
		t.Skip("foreign timers unavailable")
	}
	const words = 6
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(words, 5); err != nil {
		t.Fatal(err)
	}
	m := new(MemoryContext)
	backing := make([]byte, 8192)
	backing[4096] = 17
	epoch := uint64(1)
	m.Clock = &epoch
	m.Fill(1, (*[4096]byte)(backing[4096:]), 1, &epoch)
	m.ClaimLinks(n)
	for phase, count := range counts {
		var b Builder
		b.Store(0, b.Binary(Add, b.Load(0), b.Constant(1)))
		b.Checkpoint(words, 1)
		b.Store(1, b.MemoryLoad(b.Load(3), 8))
		b.Checkpoint(words, 2)
		b.Store(2, b.Elapsed(b.Constant(0), 1, 0))
		condition := b.Binary(Less, b.Load(0), b.Load(4))
		b.Store(5, b.Choose(condition, b.Constant(uint64(phase*4)), b.Constant(uint64((1-phase)*4))))
		b.Checkpoint(words, count)
		b.LoopContinue(condition)
		entry, e := n.CompileCodeVersionedLoop(b.FinishMemory(words), words, count, m)
		if e != nil {
			t.Fatal(e)
		}
		if e = n.PrepareTargetLink(uint64(phase*4), entry, words, count); e != nil {
			t.Fatal(e)
		}
		prefix := [17]uint8{}
		for i := 2; i < len(prefix); i++ {
			prefix[i] = 1
		}
		m.PublishLink(uint64(phase*4), entry, count, prefix)
		if m.LookupLink(uint64(phase*4)) == nil {
			t.Fatal("publication rejected")
		}
	}
	if !n.AdmitCodeVersionedContext(m) {
		t.Fatal("admission")
	}
	for _, limit := range []uint64{1, 9, 31, 100, 10000, 1000000} {
		for _, budget := range []uint64{3, 7, 63, 64, 65, 127, 128, 129, 1001, 4095, 4096, 4097, 8191, 8192, 8193, 65535, 65536} {
			state := []uint64{0, 999, 888, 4096, limit, 0}
			hit, e := n.RunLinkedBackingSessionUntil(state, m, budget, backing, rfenativebridge.MonotonicNS()+1000000000)
			if e != nil || hit {
				t.Fatalf("limit=%d budget=%d state=%v total=%d remaining=%d status=%d retired=%d hit=%v err=%v", limit, budget, state, m.Total, m.Remaining, m.Status, m.Retired, hit, e)
			}
			retired, iterations, phase, lastStart := uint64(0), uint64(0), uint64(0), uint64(888)
			for {
				count := uint64(counts[phase])
				if retired+count > budget {
					break
				}
				lastStart = retired
				retired += count
				iterations++
				if iterations >= limit {
					phase = 1 - phase
				}
			}
			if m.Total != retired || m.Remaining != budget-retired || m.MemoryTotal != iterations || state[0] != iterations || state[2] != lastStart || state[5] != phase*4 {
				t.Fatalf("oracle limit=%d budget=%d state=%v total=%d want=%d remaining=%d memory=%d", limit, budget, state, m.Total, retired, m.Remaining, m.MemoryTotal)
			}
		}
	}
}

// Fault after many internal polls and expiry both publish one exact scalar
// prefix. The counter before a denied load must not be replayed or lost.
func TestNativeResidentCyclicTimerPreciseFaultAndExpiry(t *testing.T) {
	for _, count := range []int{3, 43} {
		testNativeResidentCyclicTimerPreciseFaultAndExpiry(t, count)
	}
}
func testNativeResidentCyclicTimerPreciseFaultAndExpiry(t *testing.T, count int) {
	if !NativeSessionsAvailable() || rfenativebridge.MonotonicNS() == 0 {
		t.Skip("foreign timers unavailable")
	}
	const words = 5
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(words, 4); err != nil {
		t.Fatal(err)
	}
	var b Builder
	b.Store(0, b.Binary(Add, b.Load(0), b.Constant(1)))
	b.Checkpoint(words, 1)
	b.Store(1, b.MemoryLoad(b.Load(2), 8))
	b.Checkpoint(words, 2)
	b.Store(2, b.Binary(Add, b.Load(2), b.Load(3)))
	b.Store(4, b.Constant(0))
	b.Checkpoint(words, count)
	b.LoopContinue(b.Constant(1))
	backing := make([]byte, 8192)
	for i := 0; i < 4096; i += 8 {
		backing[4096+i] = 17
	}
	epoch := uint64(1)
	m := new(MemoryContext)
	m.Clock = &epoch
	m.Fill(1, (*[4096]byte)(backing[4096:]), 1, &epoch)
	entry, e := n.CompileCodeVersionedLoop(b.FinishMemory(words), words, count, m)
	if e != nil {
		t.Fatal(e)
	}
	m.ClaimLinks(n)
	if e = n.PrepareTargetLink(0, entry, words, count); e != nil {
		t.Fatal(e)
	}
	prefix := [17]uint8{}
	for i := 2; i < len(prefix); i++ {
		prefix[i] = 1
	}
	m.PublishLink(0, entry, count, prefix)
	if m.LookupLink(0) == nil {
		t.Fatal("publication rejected")
	}
	state := []uint64{0, 999, 4096, 8, 0}
	hit, e := n.RunLinkedBackingSessionUntil(state, m, 65536, backing, rfenativebridge.MonotonicNS()+1000000000)
	if e != nil || hit || m.Status != 1 || m.Total != uint64(512*count+1) || m.Remaining != 65536-uint64(512*count+1) || m.MemoryTotal != 512 || m.Address != 8192 || state[0] != 513 || state[1] != 17 || state[2] != 8192 {
		t.Fatal("precise prefix across polls", e, hit, state, m.Total, m.Remaining, m.MemoryTotal, m.Status, m.Address)
	}
	for _, delay := range []uint64{0, 10000, 50000} {
		state = []uint64{0, 999, 4096, 0, 0}
		deadline := rfenativebridge.MonotonicNS() + delay
		if delay == 0 {
			deadline--
		}
		hit, e = n.RunLinkedBackingSessionUntil(state, m, 65536, backing, deadline)
		if e != nil || delay == 0 && !hit || !hit && m.Total != 65536/uint64(count)*uint64(count) || m.Status != 0 || m.Total%uint64(count) != 0 || m.Total > 65536 || m.Remaining != 65536-m.Total || m.MemoryTotal != m.Total/uint64(count) || state[0] != m.Total/uint64(count) || state[2] != 4096 || state[4] != 0 || m.Total != 0 && state[1] != 17 {
			t.Fatal("expiry prefix", delay, e, hit, state, m.Total, m.Status, m.MemoryTotal)
		}
		if m.TimerFunction != 0 || m.TimerDeadline != 0 || m.TimerExpired != 0 {
			t.Fatal("timer borrow survived")
		}
	}
	// Epoch/state aliases still take only the original <=64 compatibility call.
	state = []uint64{0, 999, 4096, 0, 0}
	m.Pages[m.PageIndex(1)].Epoch = &state[0]
	hit, e = n.RunLinkedBackingSessionUntil(state, m, 65536, backing, rfenativebridge.MonotonicNS()+1000000000)
	if e != nil || hit || m.Total != 64/uint64(count)*uint64(count) || m.Remaining != 65536-m.Total || state[0] != 64/uint64(count) {
		t.Fatal("alias upgraded", e, hit, state, m.Total, m.Remaining)
	}
}
