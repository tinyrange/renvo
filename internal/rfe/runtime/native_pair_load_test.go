//go:build !renvo

package runtime

import "testing"

func TestNativePairLoadWholeFaultAndBudgets(t *testing.T) {
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
	low, high := b.MemoryPairLoad(address)
	b.Store(1, low)
	b.Store(2, high)
	b.Store(3, b.Binary(Xor, low, high))
	b.Store(0, b.Binary(Add, address, b.Constant(16)))
	b.Store(4, b.Constant(0))
	b.Checkpoint(5, 1)
	b.LoopContinue(b.Constant(1))
	ops := b.FinishMemory(5)
	entry, err := n.CompileLoop(ops, 5, 1)
	if err != nil {
		t.Fatal(err)
	}
	for budget := uint64(1); budget <= 64; budget++ {
		for _, initial := range []uint64{0, 4064, 4072, 4080, 4088, 4096, 1 << 48} {
			page := new([4096]byte)
			for i := range page {
				page[i] = byte(i*37 + 129)
			}
			epoch := uint64(1)
			m := new(MemoryContext)
			m.Fill(0, page, 1, &epoch)
			m.ClaimLinks(n)
			m.PublishLink(0, entry, 1, [17]uint8{})
			state := []uint64{initial, 99, 98, 97, 0}
			expected := append([]uint64(nil), state...)
			iterations := uint64(0)
			for iterations < budget && expected[0] <= 4080 {
				a, c := uint64(0), uint64(0)
				for j := uint64(0); j < 8; j++ {
					a |= uint64(page[expected[0]+j]) << (j * 8)
					c |= uint64(page[expected[0]+8+j]) << (j * 8)
				}
				expected[0] += 16
				expected[1], expected[2], expected[3] = a, c, a^c
				iterations++
			}
			if err = n.CallLinked(state, m, budget); err != nil {
				t.Fatal(err)
			}
			for i := range expected {
				if state[i] != expected[i] {
					t.Fatal("pair value", initial, budget, state, expected)
				}
			}
			wantStatus := uint64(0)
			if iterations < budget {
				wantStatus = 1
			}
			if m.Total != iterations || m.MemoryTotal != iterations || m.LoopIterations != iterations || m.Status != wantStatus || (wantStatus == 1 && m.Address != expected[0]) {
				t.Fatal("pair progress", initial, budget, m.Total, m.MemoryTotal, m.LoopIterations, m.Status, m.Address)
			}
		}
	}
	// The high half is structural, cannot be emitted from arbitrary raw SSA or
	// removed/reordered past an access; pure emitters must reject both halves.
	for _, invalid := range [][]Op{
		{{Kind: Const}, {Kind: PairHigh, A: 0}},
		{{Kind: Const}, {Kind: MemoryLoad, A: 0, Imm: 16}},
		{{Kind: Const}, {Kind: MemoryLoad, A: 0, Imm: 16}, {Kind: PairHigh, A: 0}},
		{{Kind: Const}, {Kind: MemoryLoad, A: 0, Imm: 16}, {Kind: PairHigh, A: 1, Imm: 1}},
	} {
		if _, err := n.CompileMemory(invalid, 5); err == nil {
			t.Fatal("invalid pair accepted", invalid)
		}
	}
	if _, err = n.Compile(ops, 5); err == nil {
		t.Fatal("pair accepted by pure emitter")
	}
	// An unused low half must not discard the checked pair or its high result.
	var h Builder
	_, second := h.MemoryPairLoad(h.Load(0))
	h.Store(1, second)
	h.Checkpoint(2, 1)
	leaf, err := n.CompileMemory(h.FinishMemory(2), 2)
	if err != nil {
		t.Fatal(err)
	}
	m := new(MemoryContext)
	page := new([4096]byte)
	page[8] = 0x81
	epoch := uint64(1)
	m.Fill(0, page, 1, &epoch)
	state := []uint64{0, 0}
	if err = n.CallMemory(leaf, state, m); err != nil || state[1] != 0x81 || m.Status != 0 || m.Retired != 1 {
		t.Fatal("unused low half", err, state, m.Status, m.Retired)
	}
}
