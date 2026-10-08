//go:build !renvo

package runtime

import "testing"

func TestNativeLoopTranslationFactsKeepPermissionsAndCallLifetime(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(5, 4); err != nil {
		t.Fatal(err)
	}
	var b Builder
	address := b.Load(0)
	value := b.MemoryLoad(address, 8)
	b.Store(1, value)
	b.Checkpoint(5, 1)
	b.MemoryStore(address, b.Binary(Add, value, b.Constant(1)), 8)
	b.Checkpoint(5, 2)
	left := b.Binary(Sub, b.Load(2), b.Constant(1))
	b.Store(2, left)
	condition := b.Binary(Equal, b.Binary(Equal, left, b.Constant(0)), b.Constant(0))
	b.Store(4, b.Choose(condition, b.Constant(0), b.Constant(4)))
	b.Checkpoint(5, 3)
	b.LoopContinue(condition)
	entry, err := n.CompileLoop(b.FinishMemory(5), 5, 3)
	if err != nil {
		t.Fatal(err)
	}
	m := new(MemoryContext)
	m.ClaimLinks(n)
	m.PublishLink(0, entry, 3, [17]uint8{})
	clock, epoch := uint64(10), uint64(10)
	m.Clock = &clock
	for _, permissions := range []uint64{3, 1, 7, 3} {
		// A different host backing page at the same guest address on each call.
		page := new([4096]byte)
		page[0] = 41
		m.Fill(1, page, permissions, &epoch)
		state := []uint64{4096, 0, 2, 0, 0}
		before := clock
		if err = n.CallLinked(state, m, 6); err != nil {
			t.Fatal(err)
		}
		if permissions == 3 {
			if state[1] != 42 || page[0] != 43 || m.Total != 6 || m.MemoryTotal != 4 || clock != before+2 || epoch != clock {
				t.Fatal("cache retained backing/epoch across calls", state, page[0], m.Total, m.MemoryTotal, clock, epoch)
			}
		} else {
			if state[1] != 41 || page[0] != 41 || m.Status != 1 || m.Address != 4096 || m.Total != 1 || m.MemoryTotal != 1 || clock != before {
				t.Fatal("load fact authorized forbidden store", permissions, state, page[0], m.Status, m.Total, m.MemoryTotal, clock)
			}
		}
	}
	// A hit must not skip width/boundary checks or wrap the epoch clock.
	clock = ^uint64(0)
	page := new([4096]byte)
	page[0] = 29
	m.Fill(1, page, 3, &epoch)
	state := []uint64{4096, 0, 2, 0, 0}
	if err = n.CallLinked(state, m, 6); err != nil || m.Total != 1 || page[0] != 29 || clock != ^uint64(0) {
		t.Fatal("clock wrap on cached store", err, m.Total, page[0], clock)
	}
	state = []uint64{8189, 0, 2, 0, 0}
	if err = n.CallLinked(state, m, 6); err != nil || m.Total != 0 || m.Address != 8189 {
		t.Fatal("cross-page access admitted", err, m.Total, m.Address)
	}
}

func TestNativeLoopCarryThirdOperandLivesAcrossIterations(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	const words = 27
	if err = n.PrepareLinks(words, 26); err != nil {
		t.Fatal(err)
	}
	var b Builder
	inputs := make([]Value, 24)
	for i := range inputs {
		inputs[i] = b.Load(i)
	}
	carry := b.Binary(And, b.Load(24), b.Constant(1))
	result := b.Carry(CarryArithmetic, inputs[0], inputs[1], carry, 64, false)
	flags := b.Carry(CarryFlags, inputs[0], inputs[1], carry, 64, false)
	b.Store(0, result)
	b.Store(24, b.Shift(Shr, flags, 29))
	for i := 2; i < 24; i++ {
		b.Store(i, b.Binary(Add, inputs[i], b.Constant(uint64(i))))
	}
	counter := b.Binary(Sub, b.Load(25), b.Constant(1))
	b.Store(25, counter)
	condition := b.Binary(Equal, b.Binary(Equal, counter, b.Constant(0)), b.Constant(0))
	b.Store(26, b.Choose(condition, b.Constant(0), b.Constant(4)))
	b.Checkpoint(words, 2)
	b.LoopContinue(condition)
	entry, err := n.CompileLoop(b.FinishMemory(words), words, 2)
	if err != nil {
		t.Fatal(err)
	}
	m := new(MemoryContext)
	m.ClaimLinks(n)
	m.PublishLink(0, entry, 2, [17]uint8{})
	state := make([]uint64, words)
	state[0] = ^uint64(0)
	state[1] = 3
	state[24] = 1
	state[25] = 7
	if err = n.CallLinked(state, m, 14); err != nil {
		t.Fatal(err)
	}
	// (-1 + 3 + 1) carries once, then 3+3+1, followed by five +3.
	if state[0] != 22 || state[24] != 0 || state[25] != 0 || state[26] != 4 || m.Total != 14 || m.LoopIterations != 7 {
		t.Fatal("loop carry/phi liveness", state, m.Total, m.LoopIterations)
	}
	for i := 2; i < 24; i++ {
		if state[i] != uint64(i*7) {
			t.Fatal("spilled state", i, state)
		}
	}
}
