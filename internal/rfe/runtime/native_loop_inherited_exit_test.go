//go:build !renvo

package runtime

import "testing"

// A fault before a slot's first assignment in a later iteration must inherit
// the previous iteration's value, even when the body never reads that slot.
func TestNativeLoopFaultInheritsUnreadOutputs(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(6, 5); err != nil {
		t.Fatal(err)
	}
	for _, budget := range []uint64{6, 7, 10, 13, 64} {
		var b Builder
		address := b.Load(0)
		b.Store(1, b.MemoryLoad(address, 8))
		b.Store(0, b.Binary(Add, address, b.Constant(8)))
		b.Checkpoint(6, 1)
		b.Store(2, b.ArithmeticFlags(address, b.Constant(8192), 64, true))
		b.Store(3, b.Choose(b.Binary(And, address, b.Constant(8)), b.Constant(37), b.Constant(19)))
		b.Store(4, b.Binary(Xor, address, b.Constant(0x123456789)))
		b.Checkpoint(6, 2)
		b.Store(5, b.Constant(0))
		b.Checkpoint(6, 3)
		b.LoopContinue(b.Constant(1))
		entry, err := n.CompileLoop(b.FinishMemory(6), 6, 3)
		if err != nil {
			t.Fatal(err)
		}
		m := new(MemoryContext)
		epoch := uint64(1)
		page := new([4096]byte)
		m.Fill(1, page, 1, &epoch)
		m.ClaimLinks(n)
		m.PublishLink(0, entry, 3, [17]uint8{})
		state := []uint64{8184, 99, 0x11111111, 98, 97, 0}
		if err = n.CallLinked(state, m, budget); err != nil {
			t.Fatal(err)
		}
		if m.Total != 3 || m.MemoryTotal != 1 || m.Status != 1 || m.Address != 8192 || state[0] != 8192 || state[1] != 0 || state[2] != 0x80000000 || state[3] != 37 || state[4] != 8184^uint64(0x123456789) || m.LoopIterations != 1 {
			t.Fatal("missing previous iteration's unread outputs", budget, state, m.Total, m.MemoryTotal, m.Status, m.Address, m.LoopIterations)
		}
	}
}
