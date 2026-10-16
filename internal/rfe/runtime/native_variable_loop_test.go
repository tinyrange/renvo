//go:build !renvo

package runtime

import (
	"reflect"
	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/rfenativebridge"
	"runtime"
	"testing"
)

// An optional two-instruction pure arm changes both data and exact iteration
// length. Speculative host evaluation must not retire that arm when unchosen.
func TestNativeVariablePureLoopExactBudgetsAndExpiry(t *testing.T) {
	for _, optional := range []uint64{1, 2} {
		testNativeVariablePureLoopExactBudgetsAndExpiry(t, optional)
	}
}
func testNativeVariablePureLoopExactBudgetsAndExpiry(t *testing.T, optional uint64) {
	if runtime.GOARCH != "amd64" || !NativeSessionsAvailable() {
		t.Skip("variable foreign loops unavailable")
	}
	n, e := NewNative(1 << 20)
	if e != nil {
		t.Fatal(e)
	}
	defer n.Close()
	const words = 5
	if e = n.PrepareLinks(words, 4); e != nil {
		t.Fatal(e)
	}
	m := new(MemoryContext)
	epoch := uint64(1)
	m.Clock = &epoch
	m.ClaimLinks(n)
	var b Builder
	b.Store(0, b.Binary(Add, b.Load(0), b.Constant(1)))
	arm := b.Binary(And, b.Load(0), b.Constant(1))
	if !b.PureIf(arm, words, func(q *Builder) bool {
		q.Store(1, q.Binary(Xor, q.Load(1), q.Constant(0xa001)))
		q.Store(2, q.Binary(Add, q.Load(2), q.Constant(17)))
		return true
	}) {
		t.Fatal("pure arm rejected")
	}
	count := b.Binary(Add, b.Constant(3), b.Choose(arm, b.Constant(optional), b.Constant(0)))
	cont := b.Binary(Less, b.Load(0), b.Load(3))
	b.Store(4, b.Choose(cont, b.Constant(0), b.Constant(4)))
	maximum := 3 + int(optional)
	b.Checkpoint(words, maximum)
	b.LoopCount(count)
	b.LoopContinue(cont)
	ops := b.FinishMemory(words)
	entry, e := n.CompileCodeVersionedLoop(ops, words, maximum, m)
	if e != nil {
		t.Fatal(e, ops)
	}
	m.PublishLink(0, entry, maximum, [17]uint8{})
	if e = n.PrepareTargetLink(0, entry, words, maximum); e != nil {
		t.Fatal(e)
	}
	backing := make([]byte, 4096)
	for _, limit := range []uint64{1, 2, 9, 1000000} {
		budgets := []uint64{4095, 4096, 4097, 8191, 8192, 8193, 65535, 65536}
		for i := uint64(1); i <= 64; i++ {
			budgets = append(budgets, i)
		}
		for _, budget := range budgets {
			state := []uint64{0, 11, 23, limit, 0}
			want := append([]uint64(nil), state...)
			retired, loops := uint64(0), uint64(0)
			for retired+uint64(maximum) <= budget && want[4] == 0 {
				want[0]++
				retired += 3
				loops++
				if want[0]&1 != 0 {
					want[1] ^= 0xa001
					want[2] += 17
					retired += optional
				}
				if want[0] >= limit {
					want[4] = 4
				}
			}
			hit, e := n.RunLinkedBackingSessionUntil(state, m, budget, backing, rfenativebridge.MonotonicNS()+1000000000)
			if e != nil || hit || !reflect.DeepEqual(state, want) || m.Total != retired || m.Remaining != budget-retired || m.Status != 0 || m.MemoryTotal != 0 || m.LoopIterations != loops {
				t.Fatal("variable scalar prefix", limit, budget, e, hit, state, want, m.Total, retired, m.Remaining, m.Status, m.LoopIterations, loops, m.Address, m.MemoryTotal)
			}
		}
	}
	for _, delay := range []uint64{0, 10000, 50000} {
		state := []uint64{0, 11, 23, 1000000, 0}
		deadline := rfenativebridge.MonotonicNS() + delay
		if delay == 0 {
			deadline--
		}
		hit, e := n.RunLinkedBackingSessionUntil(state, m, 65536, backing, deadline)
		loops := state[0]
		retired := 3*loops + optional*((loops+1)/2)
		wantXor := uint64(11)
		if ((loops+1)/2)&1 != 0 {
			wantXor ^= 0xa001
		}
		if e != nil || delay == 0 && !hit || m.Total != retired || m.Remaining != 65536-retired || state[1] != wantXor || state[2] != 23+17*((loops+1)/2) || state[4] != 0 || m.Status != 0 || m.LoopIterations != loops {
			t.Fatal("variable expiry prefix", delay, e, hit, state, m.Total, retired, m.Remaining, m.Status)
		}
	}
	// The backend independently rejects forged/unbounded counts even if a host
	// caller bypasses the normal validated operation projection.
	records := nativeRecords(ops)
	for i := 0; i < len(records); i += 4 {
		if records[i] == LoopCount {
			bad := append([]int(nil), records...)
			bad[i+1] = 0
			// Find a raw architectural input, never a bounded count constant.
			for j := 0; j < i; j += 4 {
				if bad[j] == LoadState {
					bad[i+1] = j / 4
					break
				}
			}
			if _, ok := backendcompiled.RenvoEmitCodeVersionedLoopBlock(bad, words, 5, false); ok {
				t.Fatal("unbounded raw count accepted")
			}
		}
	}
}

func TestPureIfRejectsSpeculativeEffects(t *testing.T) {
	for _, effect := range []func(*Builder){
		func(b *Builder) { b.Store(0, b.MemoryLoad(b.Constant(4096), 8)) },
		func(b *Builder) { b.Guard(b.Load(0)) },
		func(b *Builder) { b.Store(0, b.Elapsed(b.Constant(0), 1, 0)) },
		func(b *Builder) { b.Store(0, b.Constant(7)); b.Checkpoint(2, 1) },
	} {
		var b Builder
		condition := b.Load(1)
		if b.PureIf(condition, 2, func(q *Builder) bool { effect(q); return true }) {
			t.Fatal("speculative effect accepted")
		}
	}
}

// A long predecessor leaves eleven instructions in the real timer slice.
// The variable successor retires ten and expires at its resident poll. Its
// precise prefix consumes its capped entry budget exactly, not strictly less.
func TestNativeVariablePrefixExactlyConsumesExpiredSlice(t *testing.T) {
	if !NativeSessionsAvailable() || runtime.GOARCH != "amd64" {
		t.Skip("foreign loop unavailable")
	}
	n, e := NewNative(1 << 20)
	if e != nil {
		t.Fatal(e)
	}
	defer n.Close()
	const words = 4
	if e = n.PrepareLinks(words, 3); e != nil {
		t.Fatal(e)
	}
	m := new(MemoryContext)
	epoch := uint64(1)
	m.Clock = &epoch
	m.ClaimLinks(n)
	var b Builder
	b.Store(0, b.Binary(Add, b.Load(0), b.Constant(1)))
	v := b.Load(1)
	for i := 0; i < 128; i++ {
		v = b.Binary(Add, b.Binary(Mul, v, b.Constant(0x5851f42d4c957f2d)), b.Constant(uint64(i+1)))
	}
	b.Store(1, v)
	cont := b.Binary(Less, b.Load(0), b.Constant(4085))
	b.Store(3, b.Choose(cont, b.Constant(0), b.Constant(4)))
	b.Checkpoint(words, 1)
	b.LoopContinue(cont)
	first, e := n.CompileCodeVersionedLoop(b.FinishMemory(words), words, 1, m)
	if e != nil {
		t.Fatal(e)
	}
	b = Builder{}
	b.Store(2, b.Binary(Add, b.Load(2), b.Constant(1)))
	b.Store(3, b.Constant(4))
	b.Checkpoint(words, 11)
	b.LoopCount(b.Constant(10))
	b.LoopContinue(b.Constant(1))
	second, e := n.CompileCodeVersionedLoop(b.FinishMemory(words), words, 11, m)
	if e != nil {
		t.Fatal(e)
	}
	for _, item := range [][3]int{{0, first, 1}, {4, second, 11}} {
		m.PublishLink(uint64(item[0]), item[1], item[2], [17]uint8{})
		if e = n.PrepareTargetLink(uint64(item[0]), item[1], words, item[2]); e != nil {
			t.Fatal(e)
		}
	}
	backing := make([]byte, 4096)
	// Warm the independently pinned session before requesting a short deadline.
	state := []uint64{0, 17, 0, 0}
	if _, e = n.RunLinkedBackingSessionUntil(state, m, 1, backing, rfenativebridge.MonotonicNS()+1000000000); e != nil {
		t.Fatal(e)
	}
	witnessed := false
	for _, delay := range []uint64{50000, 100000, 200000} {
		state = []uint64{0, 17, 0, 0}
		hit, e := n.RunLinkedBackingSessionUntil(state, m, 65536, backing, rfenativebridge.MonotonicNS()+delay)
		expected := state[0] + 10*state[2]
		if e != nil || !hit || m.Total != expected || m.Remaining != 65536-expected || m.Status != 0 || state[0] > 4085 {
			t.Fatal("exact expired prefix", delay, e, hit, state, m.Total, expected, m.Remaining, m.Status)
		}
		if state[0] == 4085 && state[2] == 1 {
			witnessed = true
		}
	}
	if !witnessed {
		t.Fatal("did not observe the short successor expiry")
	}
}
