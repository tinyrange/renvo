//go:build !renvo

package runtime

import "testing"

func TestNativeLoopPhiCyclesSpillsAndBudgets(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(26, 25); err != nil {
		t.Fatal(err)
	}
	var b Builder
	inputs := make([]Value, 24)
	for i := range inputs {
		inputs[i] = b.Load(i)
	}
	for i := range inputs {
		b.Store(i, inputs[(i+1)%len(inputs)])
	}
	counter := b.Binary(Sub, b.Load(24), b.Constant(1))
	b.Store(24, counter)
	condition := b.Binary(Equal, b.Binary(Equal, counter, b.Constant(0)), b.Constant(0))
	b.Store(25, b.Choose(condition, b.Constant(0), b.Constant(4)))
	b.Checkpoint(26, 2)
	b.LoopContinue(condition)
	entry, err := n.CompileLoop(b.FinishMemory(26), 26, 2)
	if err != nil {
		t.Fatal(err)
	}
	m := new(MemoryContext)
	m.ClaimLinks(n)
	m.PublishLink(0, entry, 2, [17]uint8{})
	if n.CallMemory(entry, make([]uint64, 26), m) == nil {
		t.Fatal("loop entry accepted without dispatcher budget ABI")
	}
	for _, initial := range []uint64{1, 3, 100} {
		for _, budget := range []uint64{2, 3, 4, 7, 63, 64} {
			state := make([]uint64, 26)
			for i := range inputs {
				state[i] = uint64(i + 1001)
			}
			state[24] = initial
			if err = n.CallLinked(state, m, budget); err != nil {
				t.Fatal(err)
			}
			iterations := budget / 2
			if iterations > initial {
				iterations = initial
			}
			for i := range inputs {
				if state[i] != uint64((i+int(iterations))%len(inputs)+1001) {
					t.Fatal("parallel phi move lost source", initial, budget, i, state)
				}
			}
			exits, pc := uint64(0), uint64(0)
			if initial == iterations {
				exits, pc = 1, 4
			}
			if state[24] != initial-iterations || state[25] != pc || m.Total != iterations*2 || m.Remaining != budget-iterations*2 || m.MemoryTotal != 0 || m.LoopIterations != iterations || m.LoopExits != exits || m.Status != 0 {
				t.Fatal("loop budget/exit accounting mismatch", initial, budget, state, m)
			}
		}
	}
	// A publication cannot lie about this installed entry's body length.
	m.PublishLink(0, entry, 1, [17]uint8{})
	state := make([]uint64, 26)
	state[24] = 100
	if err = n.CallLinked(state, m, 64); err != nil || m.Total != 0 || state[24] != 100 {
		t.Fatal("wrong loop length admitted", err, m.Total, state)
	}
}

func TestNativeLoopFaultRestoresSpilledStateOnce(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(26, 25); err != nil {
		t.Fatal(err)
	}
	var b Builder
	address := b.Load(1)
	values := make([]Value, 23)
	for i := range values {
		values[i] = b.Binary(Add, b.Load(i+2), b.Constant(uint64(i+3)))
	}
	b.Store(0, b.Binary(Add, b.Load(0), b.Constant(1)))
	for i, v := range values {
		b.Store(i+2, v)
	}
	b.Store(25, b.Constant(0))
	b.Checkpoint(26, 1)
	b.MemoryStore(address, b.Load(0), 8)
	b.Store(1, b.Binary(Add, address, b.Constant(8)))
	b.Checkpoint(26, 2)
	b.LoopContinue(b.Constant(1))
	entry, err := n.CompileLoop(b.FinishMemory(26), 26, 2)
	if err != nil {
		t.Fatal(err)
	}
	var page [4096]byte
	clock, epoch := uint64(1), uint64(1)
	m := &MemoryContext{Clock: &clock}
	m.Fill(1, &page, 3, &epoch)
	m.ClaimLinks(n)
	m.PublishLink(0, entry, 2, [17]uint8{0, 0, 1})
	state := make([]uint64, 26)
	state[0], state[1] = 40, 8192-8
	for i := range values {
		state[i+2] = uint64(i + 101)
	}
	if err = n.CallLinked(state, m, 64); err != nil {
		t.Fatal(err)
	}
	if state[0] != 42 || state[1] != 8192 || state[25] != 0 || m.Total != 3 || m.Retired != 3 || m.MemoryTotal != 1 || m.Status != 1 || m.Address != 8192 || clock != 2 || epoch != 2 || page[4088] != 41 || m.LoopIterations != 1 {
		t.Fatal("loop fault lost/replayed prefix", state, m, page[4088], clock, epoch)
	}
	for i := range values {
		if state[i+2] != uint64(i+101)+uint64(i+3)*2 {
			t.Fatal("spilled exit state lost", i, state)
		}
	}
}
