//go:build !renvo

package runtime

import "testing"

// Exit-only arithmetic must retain loaded leaves across a later aliasing store,
// and inherit the previous iteration at a fault before the next assignment.
func TestNativeLoopExitArithmeticAliasingAndInheritance(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(8, 7); err != nil {
		t.Fatal(err)
	}
	var b Builder
	address := b.Load(0)
	original := b.MemoryLoad(address, 1)
	b.MemoryStore(address, b.Binary(Add, original, b.Constant(1)), 1)
	b.Checkpoint(8, 1)
	product := b.Binary(Mul, b.Binary(Xor, original, b.Constant(0xa5)), b.Constant(37))
	sum := b.Binary(Sub, b.Binary(Add, product, address), b.Constant(123))
	packed := b.Binary(Or, b.Shift(Shl, sum, 11), b.Shift(Shr, product, 3))
	b.Store(1, product)
	b.Store(2, sum)
	b.Store(3, packed)
	b.Store(4, b.Binary(And, packed, b.Constant(0xffffffff)))
	b.Store(5, b.Choose(b.Binary(Equal, original, b.Constant(9)), sum, product))
	b.Checkpoint(8, 2)
	b.Store(0, b.Binary(Add, address, b.Constant(8)))
	b.Checkpoint(8, 3)
	b.LoopContinue(b.Constant(1))
	entry, err := n.CompileLoop(b.FinishMemory(8), 8, 3)
	if err != nil {
		t.Fatal(err)
	}
	for _, budget := range []uint64{3, 6, 9, 30} {
		page := new([4096]byte)
		page[4080], page[4088] = 9, 217
		clock, epoch := uint64(1), uint64(1)
		m := &MemoryContext{NativeContext: NativeContext{Clock: &clock}}
		m.Fill(1, page, 3, &epoch)
		m.ClaimLinks(n)
		m.PublishLink(0, entry, 3, [17]uint8{})
		state := []uint64{8176, 11, 12, 13, 14, 15, 16, 0}
		if err = n.CallLinked(state, m, budget); err != nil {
			t.Fatal(err)
		}
		iterations := uint64(2)
		original, address := uint64(217), uint64(8184)
		if budget == 3 {
			iterations, original, address = 1, 9, 8176
		}
		product := (original ^ 0xa5) * 37
		sum := product + address - 123
		packed := sum<<11 | product>>3
		selected := product
		if original == 9 {
			selected = sum
		}
		want := []uint64{8176 + iterations*8, product, sum, packed, uint64(uint32(packed)), selected, 16, 0}
		for i := range state {
			if state[i] != want[i] {
				t.Fatalf("budget=%d slot=%d got=%x want=%x", budget, i, state[i], want[i])
			}
		}
		if m.Total != iterations*3 || m.MemoryTotal != iterations*2 || clock != 1+iterations || epoch != clock || page[4080] != 10 {
			t.Fatalf("budget=%d totals=%d/%d clock=%d epoch=%d page=%d", budget, m.Total, m.MemoryTotal, clock, epoch, page[4080])
		}
		if budget > 6 && (m.Status != 1 || m.Address != 8192) {
			t.Fatalf("missing precise fault: %+v", m)
		}
		if iterations == 2 && page[4088] != 218 {
			t.Fatal("alias store not retired")
		}
	}
}
