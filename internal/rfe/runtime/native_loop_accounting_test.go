//go:build !renvo

package runtime

import "testing"

// Vary both dimensions independently, including every odd/non-power-of-two
// body size. A loop ends either at its predicate or at a partial call budget.
func TestNativeLoopAllLengthsAndBudgets(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(3, 2); err != nil {
		t.Fatal(err)
	}
	for instructions := 1; instructions <= 16; instructions++ {
		var b Builder
		value := b.Binary(Add, b.Load(0), b.Constant(1))
		b.Store(0, value)
		condition := b.Binary(Less, value, b.Load(1))
		b.Store(2, b.Choose(condition, b.Constant(0), b.Constant(4)))
		b.Checkpoint(3, instructions)
		b.LoopContinue(condition)
		entry, err := n.CompileLoop(b.FinishMemory(3), 3, instructions)
		if err != nil {
			t.Fatal(err)
		}
		m := new(MemoryContext)
		m.ClaimLinks(n)
		m.PublishLink(0, entry, instructions, [17]uint8{})
		for budget := uint64(1); budget <= 64; budget++ {
			for limit := uint64(1); limit <= 66; limit += 5 {
				state := []uint64{0, limit, 0}
				if err := n.CallLinked(state, m, budget); err != nil {
					t.Fatal(err)
				}
				want := budget / uint64(instructions)
				if want > limit {
					want = limit
				}
				pc, exits := uint64(0), uint64(0)
				if want == limit {
					pc, exits = 4, 1
				}
				if state[0] != want || state[1] != limit || state[2] != pc || m.Total != want*uint64(instructions) || m.Remaining != budget-m.Total || m.LoopIterations != want || m.LoopExits != exits || m.Status != 0 {
					t.Fatalf("length=%d budget=%d limit=%d state=%v total=%d remaining=%d iterations=%d exits=%d", instructions, budget, limit, state, m.Total, m.Remaining, m.LoopIterations, m.LoopExits)
				}
			}
		}
	}
}
