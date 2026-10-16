//go:build !renvo

package runtime

import (
	"reflect"
	"renvo.dev/internal/rfenativebridge"
	"testing"
)

// Three bodies alternate two outgoing edges of one predecessor. A separate
// scalar interpreter checks exact retirement, memory faults and short budgets
// after training, re-preparation, descriptor revocation and entry replacement.
func TestNativeAlternatingSuccessorPrecisePrefixes(t *testing.T) {
	for _, profile := range []bool{false, true} {
		for _, collide := range []bool{false, true} {
			testNativeAlternatingSuccessorPrecisePrefixes(t, profile, collide)
		}
	}
}
func testNativeAlternatingSuccessorPrecisePrefixes(t *testing.T, profile, collide bool) {
	if !NativeSessionsAvailable() || rfenativebridge.MonotonicNS() == 0 {
		t.Skip("foreign session unavailable")
	}
	const words = 5
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	n.profileDispatch = profile
	if err = n.PrepareLinks(words, 4); err != nil {
		t.Fatal(err)
	}
	m := new(MemoryContext)
	epoch := uint64(1)
	backing := make([]byte, 8192)
	backing[4096] = 17
	m.Clock = &epoch
	m.Fill(1, (*[4096]byte)(backing[4096:]), 1, &epoch)
	m.ClaimLinks(n)
	entries := [3]int{}
	pcs := [3]uint64{0, 4, 8}
	if collide {
		// All four ways of set 255, including private index 1023. Encoding
		// the last row's successor as index+1 requires the full eleven bits.
		pcs = [3]uint64{0x3fc, 0x4003fc, 0x8003fc}
	}
	compile := func(phase int, bias uint64) int {
		var b Builder
		if phase == 0 {
			b.Store(0, b.Binary(Add, b.Load(0), b.Constant(bias)))
			b.Store(1, b.Binary(Xor, b.Load(1), b.Constant(1)))
			b.Store(4, b.Choose(b.Load(1), b.Constant(pcs[1]), b.Constant(pcs[2])))
		} else {
			b.Store(0, b.Binary(Add, b.Load(0), b.Constant(bias)))
			b.Checkpoint(words, 1)
			b.Store(3, b.MemoryLoad(b.Load(2), 8))
			b.Store(4, b.Constant(pcs[0]))
		}
		b.Checkpoint(words, 3)
		b.LoopContinue(b.Constant(0))
		entry, e := n.CompileCodeVersionedLoop(b.FinishMemory(words), words, 3, m)
		if e != nil {
			t.Fatal(e)
		}
		return entry
	}
	for phase := range entries {
		bias := uint64(1)
		if phase != 0 {
			bias = uint64(phase * 10)
		}
		entries[phase] = compile(phase, bias)
		if phase == 0 && collide {
			if err = n.PrepareTargetLink(0xc003fc, entries[0], words, 3); err != nil {
				t.Fatal(err)
			}
		}
		m.PublishLink(pcs[phase], entries[phase], 3, [17]uint8{0, 0, 1, 1})
		if err = n.PrepareTargetLink(pcs[phase], entries[phase], words, 3); err != nil {
			t.Fatal(err)
		}
	}
	run := func(budget, address, start, rootBias, denied uint64) {
		state := []uint64{0, 0, address, 999, start}
		want := append([]uint64(nil), state...)
		var retired, memories, status uint64
		for retired+3 <= budget && want[4] != denied {
			if want[4] == pcs[0] {
				want[0] += rootBias
				want[1] ^= 1
				want[4] = pcs[2]
				if want[1] != 0 {
					want[4] = pcs[1]
				}
			} else {
				if want[4] == pcs[1] {
					want[0] += 10
				} else {
					want[0] += 20
				}
				if address != 4096 {
					retired++
					status = 1
					break
				}
				want[3] = 17
				want[4] = pcs[0]
				memories++
			}
			retired += 3
		}
		hit, e := n.RunLinkedBackingSessionUntil(state, m, budget, backing, rfenativebridge.MonotonicNS()+1000000000)
		if e != nil || hit || !reflect.DeepEqual(state, want) || m.Total != retired || m.Remaining != budget-retired || m.Status != status || m.MemoryTotal != memories || status == 1 && m.Address != address {
			t.Fatal("scalar prefix", budget, address, start, denied, e, hit, state, want, m.Total, retired, m.Remaining, m.Status, m.MemoryTotal, memories)
		}
	}
	const none = ^uint64(0)
	for _, budget := range []uint64{65535, 1, 2, 3, 5, 6, 7, 11, 12, 13, 63, 64, 65, 4095, 4096, 4097, 65536} {
		for _, start := range pcs {
			run(budget, 4096, start, 1, none)
		}
	}
	// Host re-preparation must preserve predictions but cannot preserve the
	// previous session's checked public descriptor.
	for phase, entry := range entries {
		if err = n.PrepareTargetLink(pcs[phase], entry, words, 3); err != nil {
			t.Fatal(err)
		}
	}
	for _, pc := range pcs {
		descriptor := m.LookupLink(pc)
		saved := *descriptor
		descriptor.Entry++
		run(127, 4096, pcs[0], 1, pc)
		*descriptor = saved
	}
	for _, pc := range pcs {
		run(127, 8192, pc, 1, none)
	}
	entries[0] = compile(0, 7)
	m.PublishLink(pcs[0], entries[0], 3, [17]uint8{})
	if err = n.PrepareTargetLink(pcs[0], entries[0], words, 3); err != nil {
		t.Fatal(err)
	}
	run(65535, 4096, pcs[0], 7, none)
	// A high PC sharing low address bits is not the predicted target's full PC.
	run(127, 4096, 1<<48, 7, 1<<48)
	counters := n.DispatchCounters()
	if profile {
		if counters[0] == 0 || counters[1] == 0 || counters[3] == 0 || counters[4] == 0 || counters[1]+counters[2]+counters[3]+counters[4] > counters[0] {
			t.Fatal("invalid diagnostic partition", counters)
		}
	} else if counters != [5]uint64{} {
		t.Fatal("ordinary dispatcher executed diagnostic counters", counters)
	}
}

// A public, arena-admitted successor can lack a resident private proof. Its
// compatibility wrapper must return the extended timer budget, not the stale
// entry window, after a long resident cycle. Architectural count and elapsed
// state give an independent witness across the private/legacy boundary.
func TestNativeUnpreparedSuccessorTimerBudget(t *testing.T) {
	if !NativeSessionsAvailable() || rfenativebridge.MonotonicNS() == 0 {
		t.Skip("foreign timer unavailable")
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
	b.Store(3, b.Constant(4))
	b.Checkpoint(words, 1)
	b.LoopContinue(b.Constant(0))
	root, e := n.CompileCodeVersionedLoop(b.FinishMemory(words), words, 1, m)
	if e != nil {
		t.Fatal(e)
	}
	if e = n.PrepareTargetLink(0, root, words, 1); e != nil {
		t.Fatal(e)
	}
	m.PublishLink(0, root, 1, [17]uint8{})
	b = Builder{}
	b.Store(0, b.Binary(Add, b.Load(0), b.Constant(1)))
	b.Store(1, b.Elapsed(b.Constant(0), 1, 0))
	b.Store(2, b.Binary(Xor, b.Load(2), b.Constant(7)))
	b.Store(3, b.Constant(4))
	b.Checkpoint(words, 43)
	b.LoopContinue(b.Constant(1))
	loop, e := n.CompileCodeVersionedLoop(b.FinishMemory(words), words, 43, m)
	if e != nil {
		t.Fatal(e)
	}
	if e = n.AdmitLink(loop, words, 43); e != nil {
		t.Fatal(e)
	}
	m.PublishLink(4, loop, 43, [17]uint8{})
	backing := make([]byte, 4096)
	for _, budget := range []uint64{4095, 4096, 4097, 8191, 8192, 8193, 65535, 65536} {
		state := []uint64{0, 999, 11, 0}
		hit, e := n.RunLinkedBackingSessionUntil(state, m, budget, backing, rfenativebridge.MonotonicNS()+1000000000)
		loops := (budget - 1) / 43
		expectedXor := uint64(11)
		if loops&1 != 0 {
			expectedXor ^= 7
		}
		if e != nil || hit || m.Total != 1+43*loops || m.Remaining != budget-m.Total || state[0] != loops || state[1] != 1+43*(loops-1) || state[2] != expectedXor || state[3] != 4 || m.Status != 0 {
			t.Fatal("compatibility timer budget", budget, e, hit, state, m.Total, m.Remaining, m.Status)
		}
	}
}
