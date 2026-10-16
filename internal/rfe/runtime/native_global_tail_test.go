//go:build !renvo

package runtime

import (
	"renvo.dev/internal/rfenativebridge"
	"testing"
)

// The independent scalar program has one load followed by N-1 pure
// instructions. A warmed N-instruction region and its original one-instruction
// leaf must agree at every global-budget tail, fault and real clock window.
func TestNativeGlobalLeafTailExactPrefixes(t *testing.T) {
	if !NativeSessionsAvailable() || rfenativebridge.MonotonicNS() == 0 {
		t.Skip("foreign clock unavailable")
	}
	const words = 4
	for _, length := range []int{3, 43} {
		n, err := NewNative(1 << 20)
		if err != nil {
			t.Fatal(err)
		}
		if err = n.PrepareLinks(words, 3); err != nil {
			t.Fatal(err)
		}
		backing := make([]byte, 8192)
		for i := 0; i < 4096; i += 8 {
			backing[4096+i] = 17
		}
		m := new(MemoryContext)
		epoch := uint64(1)
		m.Clock = &epoch
		m.Fill(1, (*[4096]byte)(backing[4096:]), 1, &epoch)
		m.ClaimLinks(n)
		root := func(b *Builder) {
			b.Store(1, b.MemoryLoad(b.Load(2), 8))
			b.Store(2, b.Binary(Add, b.Load(2), b.Constant(8)))
			b.Store(0, b.Binary(Add, b.Load(0), b.Constant(1)))
			b.Store(3, b.Constant(4))
			b.Checkpoint(words, 1)
		}
		var leaf Builder
		root(&leaf)
		tail, err := n.CompileMemory(leaf.FinishMemory(words), words)
		if err != nil {
			t.Fatal(err)
		}
		var b Builder
		root(&b)
		for i := 1; i < length; i++ {
			b.Store(0, b.Binary(Add, b.Load(0), b.Constant(1)))
			b.Store(3, b.Constant(uint64((i+1)%length)*4))
			b.Checkpoint(words, i+1)
		}
		b.LoopContinue(b.Constant(1))
		region, err := n.CompileCodeVersionedLoop(b.FinishMemory(words), words, length, m)
		if err != nil {
			t.Fatal(err)
		}
		prefix := [17]uint8{}
		for i := 1; i < len(prefix); i++ {
			prefix[i] = 1
		}
		m.PublishLink(0, region, length, prefix)
		if err = n.PrepareTargetLinkWithTail(0, region, words, length, tail, 1); err != nil {
			t.Fatal(err)
		}
		for i := 1; i < length; i++ {
			var q Builder
			q.Store(0, q.Binary(Add, q.Load(0), q.Constant(1)))
			q.Store(3, q.Constant(uint64((i+1)%length)*4))
			entry, e := n.Compile(q.Finish(words), words)
			if e != nil {
				t.Fatal(e)
			}
			m.PublishLink(uint64(i)*4, entry, 1, [17]uint8{})
			if e = n.PrepareTargetLink(uint64(i)*4, entry, words, 1); e != nil {
				t.Fatal(e)
			}
		}
		if !n.AdmitCodeVersionedContext(m) {
			t.Fatal("admission")
		}
		before := *m.LookupLink(0)
		budgets := []uint64{65, 127, 128, 129, 1001, 4095, 4096, 4097, 8191, 8192, 8193, 65535, 65536}
		for i := uint64(1); i <= 64; i++ {
			budgets = append(budgets, i)
		}
		for _, budget := range budgets {
			// Fixed address keeps the backing valid throughout the longest trial.
			state := []uint64{0, 999, 4096, 0}
			hit, e := n.RunLinkedBackingSessionUntil(state, m, budget, backing, rfenativebridge.MonotonicNS()+1000000000)
			expected := budget
			loads := (budget + uint64(length) - 1) / uint64(length)
			if loads > 512 {
				expected = 512 * uint64(length)
				loads = 512
			}
			wantStatus := uint64(0)
			if budget > expected {
				wantStatus = 1
			}
			if e != nil || hit || m.Total != expected || m.Remaining != budget-expected || m.MemoryTotal != loads || state[0] != expected || state[1] != 17 || state[2] != 4096+8*loads || state[3] != (expected%uint64(length))*4 || m.Status != wantStatus || *m.LookupLink(0) != before {
				t.Fatal("tail prefix", length, budget, e, hit, state, m.Total, m.Remaining, m.MemoryTotal, m.Status)
			}
		}
		// A denied first access of a native tail retains the complete preceding
		// region, rather than polling early or replaying any of its instructions.
		state := []uint64{0, 999, 8184, 0}
		hit, e := n.RunLinkedBackingSessionUntil(state, m, 64, backing, rfenativebridge.MonotonicNS()+1000000000)
		if e != nil || hit || m.Status != 1 || m.Address != 8192 || m.Total != uint64(length) || m.MemoryTotal != 1 || state[0] != uint64(length) || state[1] != 17 || state[2] != 8192 || state[3] != 0 {
			t.Fatal("tail fault", length, e, hit, state, m.Total, m.Status, m.Address)
		}
		// Fresh public mismatches cannot borrow the tail proof or stale admission.
		m.LookupLink(0).Entry++
		state = []uint64{0, 999, 4096, 0}
		hit, e = n.RunLinkedBackingSessionUntil(state, m, 1, backing, rfenativebridge.MonotonicNS()+1000000000)
		if e != nil || hit || m.Total != 0 || state[0] != 0 || state[1] != 999 || state[2] != 4096 || state[3] != 0 {
			t.Fatal("forged region granted tail", e, hit, state, m.Total)
		}
		*m.LookupLink(0) = before
		state = []uint64{0, 999, 4096, 0}
		hit, e = n.RunLinkedBackingSessionUntil(state, m, 65536, backing, rfenativebridge.MonotonicNS()-1)
		if e != nil || !hit || m.Total != 0 || state[0] != 0 || state[1] != 999 || *m.LookupLink(0) != before {
			t.Fatal("expired before tail", e, hit, state, m.Total)
		}
		n.Close()
	}
}

// The original leaf goes to an unpublished successor. Selecting it for a
// mere 64-instruction poll-window remainder would end a long session early.
func TestNativeGlobalTailNeverSelectedForPollWindow(t *testing.T) {
	if !NativeSessionsAvailable() || rfenativebridge.MonotonicNS() == 0 {
		t.Skip("foreign clock unavailable")
	}
	const words = 3
	n, e := NewNative(1 << 20)
	if e != nil {
		t.Fatal(e)
	}
	defer n.Close()
	if e = n.PrepareLinks(words, 2); e != nil {
		t.Fatal(e)
	}
	m := new(MemoryContext)
	epoch := uint64(1)
	m.Clock = &epoch
	m.ClaimLinks(n)
	backing := make([]byte, 4096)
	var leaf Builder
	leaf.Store(0, leaf.Binary(Add, leaf.Load(0), leaf.Constant(1)))
	leaf.Store(2, leaf.Constant(4))
	leaf.Checkpoint(words, 1)
	tail, e := n.CompileMemory(leaf.FinishMemory(words), words)
	if e != nil {
		t.Fatal(e)
	}
	var b Builder
	b.Store(0, b.Binary(Add, b.Load(0), b.Constant(1)))
	b.Store(2, b.Constant(0))
	b.Checkpoint(words, 43)
	b.LoopContinue(b.Constant(1))
	region, e := n.CompileCodeVersionedLoop(b.FinishMemory(words), words, 43, m)
	if e != nil {
		t.Fatal(e)
	}
	if e = n.PrepareTargetLinkWithTail(0, region, words, 43, tail, 1); e != nil {
		t.Fatal(e)
	}
	m.PublishLink(0, region, 43, [17]uint8{})
	if m.LookupLink(0) == nil {
		t.Fatal("publication")
	}
	for _, budget := range []uint64{1, 42, 43, 44, 63, 64, 65, 127, 128, 129, 1001, 4095, 4096, 4097, 8191, 8192, 8193, 65535, 65536} {
		state := []uint64{0, 7, 0}
		hit, e := n.RunLinkedBackingSessionUntil(state, m, budget, backing, rfenativebridge.MonotonicNS()+1000000000)
		loops := budget / 43
		tailCount := uint64(0)
		if budget%43 != 0 {
			tailCount = 1
		}
		if e != nil || hit || m.Total != loops*43+tailCount || m.Remaining != budget-m.Total || m.Status != 0 || state[0] != loops+tailCount || state[2] != tailCount*4 || m.LoopIterations != loops {
			t.Fatal("global/poll discriminator", budget, e, hit, state, m.Total, m.Remaining, m.Status, m.LoopIterations, loops)
		}
	}
}
