//go:build !renvo

package runtime

import "testing"

func TestNativeLoopDeferredXorPredicate(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(4, 3); err != nil {
		t.Fatal(err)
	}
	for _, boolean := range []bool{false, true} {
		var b Builder
		a, c := b.Load(0), b.Load(1)
		if boolean {
			a = b.Binary(Equal, a, b.Constant(0))
			c = b.Binary(Equal, c, b.Constant(0))
		}
		condition := b.Binary(Xor, a, c)
		b.Store(2, b.Binary(Add, b.Load(2), b.Constant(1)))
		b.Store(3, b.Choose(condition, b.Constant(0), b.Constant(4)))
		b.Checkpoint(4, 1)
		b.LoopContinue(condition)
		entry, err := n.CompileLoop(b.FinishMemory(4), 4, 1)
		if err != nil {
			t.Fatal(err)
		}
		for _, pair := range [][2]uint64{{0, 0}, {0, 1}, {1, 0}, {1, 1}, {2, 1}, {2, 2}, {3, 2}} {
			m := new(MemoryContext)
			m.ClaimLinks(n)
			m.PublishLink(0, entry, 1, [17]uint8{})
			state := []uint64{pair[0], pair[1], 0, 0}
			if err = n.CallLinked(state, m, 5); err != nil {
				t.Fatal(err)
			}
			keep := pair[0] != pair[1]
			if boolean {
				keep = (pair[0] == 0) != (pair[1] == 0)
			}
			iterations, pc := uint64(1), uint64(4)
			if keep {
				iterations, pc = 5, 0
			}
			if state[2] != iterations || state[3] != pc || m.Total != iterations {
				t.Fatalf("boolean=%v pair=%v state=%v total=%d want=%d/%d", boolean, pair, state, m.Total, iterations, pc)
			}
		}
	}
}
