//go:build !renvo

package runtime

import "testing"

// Many loop-carried inputs force spills. Exit-only flags and a select must use
// the last iteration's operands, including at a cached-store overflow exit.
func TestNativeLoopColdValuesAndCachedOverflowWithSpills(t *testing.T) {
	const words = 30
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(words, 29); err != nil {
		t.Fatal(err)
	}
	var b Builder
	values := make([]Value, 20)
	for i := range values {
		values[i] = b.Load(i)
	}
	address := b.Load(25)
	for i, v := range values {
		b.Store(i, b.Binary(Add, v, b.Constant(uint64(i+1))))
	}
	b.Store(29, b.Constant(4))
	b.Checkpoint(words, 1)
	b.Store(21, b.ArithmeticFlags(values[0], b.Constant(1), 64, true))
	b.Store(24, b.LogicalFlags(b.Binary(Add, values[0], b.Constant(1)), 64))
	b.Store(29, b.Constant(8))
	b.Checkpoint(words, 2)
	b.Store(23, b.Choose(b.Binary(And, values[0], b.Constant(1)), b.Binary(Add, values[0], b.Constant(11)), b.Binary(Add, values[1], b.Constant(17))))
	b.Store(29, b.Constant(12))
	b.Checkpoint(words, 3)
	left := b.Binary(Sub, b.Load(20), b.Constant(1))
	b.Store(20, left)
	b.Store(29, b.Constant(16))
	b.Checkpoint(words, 4)
	b.MemoryStore(address, left, 8)
	condition := b.Binary(Equal, b.Binary(Equal, left, b.Constant(0)), b.Constant(0))
	b.Store(29, b.Choose(condition, b.Constant(0), b.Constant(20)))
	b.Checkpoint(words, 5)
	b.LoopContinue(condition)
	entry, err := n.CompileLoop(b.FinishMemory(words), words, 5)
	if err != nil {
		t.Fatal(err)
	}
	if err = n.AdmitLink(entry, words, 5); err != nil {
		t.Fatal(err)
	}
	for _, overflow := range []bool{false, true} {
		page := new([4096]byte)
		clock, epoch := uint64(10), uint64(10)
		if overflow {
			clock = ^uint64(0) - 1
		}
		m := &MemoryContext{Clock: &clock}
		m.Fill(1, page, 3, &epoch)
		m.ClaimLinks(n)
		m.PublishLink(0, entry, 5, [17]uint8{})
		state := make([]uint64, words)
		for i := 0; i < 20; i++ {
			state[i] = uint64(10 + i)
		}
		state[20], state[21], state[23], state[24], state[25] = 3, 999, 998, 997, 4096
		if err = n.CallLinked(state, m, 15); err != nil {
			t.Fatal(err)
		}
		iterations := uint64(3)
		if overflow {
			iterations = 2
			if m.Total != 9 || m.MemoryTotal != 1 || m.LoopIterations != 1 || m.Status != 1 || m.Address != 4096 || state[29] != 16 || page[0] != 2 || clock != ^uint64(0) || epoch != clock {
				t.Fatal("cached store overflow replay/progress", state, m.Total, m.MemoryTotal, m.LoopIterations, m.Status, page[0], clock, epoch)
			}
		} else if m.Total != 15 || m.MemoryTotal != 3 || m.LoopIterations != 3 || state[29] != 20 || page[0] != 0 || clock != 13 || epoch != 13 {
			t.Fatal("cold normal exit", state, m.Total, m.MemoryTotal, clock)
		}
		for i := 0; i < 20; i++ {
			if state[i] != uint64(10+i)+iterations*uint64(i+1) {
				t.Fatal("spilled input", i, state[i])
			}
		}
		wantSelect := uint64(32) // third iteration: old input1=15, even old input0=12
		if overflow {
			wantSelect = 22
		}
		if state[20] != 3-iterations || state[21] != 0x20000000 || state[24] != 0 || state[23] != wantSelect || state[25] != 4096 {
			t.Fatal("cold flags/select/readonly state", state)
		}
		if overflow {
			if err = n.CallLinked(state, m, 15); err != nil || m.Total != 0 || page[0] != 2 || clock != ^uint64(0) {
				t.Fatal("fault prefix replay", err, state, m.Total)
			}
		}
	}
}

// A prior flags value is loop-carried even when each iteration overwrites it;
// exit-only classification must never discard that phi or substitute new flags.
func TestNativeLoopOldFlagsPhiRemainsHot(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(5, 4); err != nil {
		t.Fatal(err)
	}
	var b Builder
	oldFlags := b.Load(1)
	left := b.Binary(Sub, b.Load(0), b.Constant(1))
	b.Store(0, left)
	b.Store(2, b.Binary(Add, b.Load(2), b.Choose(b.Binary(And, oldFlags, b.Constant(0x40000000)), b.Constant(7), b.Constant(3))))
	b.Store(1, b.ArithmeticFlags(left, b.Constant(0), 64, true))
	condition := b.Binary(Equal, b.Binary(Equal, left, b.Constant(0)), b.Constant(0))
	b.Store(4, b.Choose(condition, b.Constant(0), b.Constant(4)))
	b.Checkpoint(5, 1)
	b.LoopContinue(condition)
	entry, err := n.CompileLoop(b.FinishMemory(5), 5, 1)
	if err != nil {
		t.Fatal(err)
	}
	m := new(MemoryContext)
	m.ClaimLinks(n)
	m.PublishLink(0, entry, 1, [17]uint8{})
	state := []uint64{3, 0x40000000, 0, 99, 0}
	if err = n.CallLinked(state, m, 3); err != nil || state[0] != 0 || state[1] != 0x60000000 || state[2] != 13 || state[3] != 99 || state[4] != 4 || m.Total != 3 {
		t.Fatal("old flags phi", err, state, m.Total)
	}
}
