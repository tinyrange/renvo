//go:build !renvo

package runtime

import "testing"

func TestNativeLoopKnownPredicateExit(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(5, 4); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []bool{false, true} {
		var b Builder
		next := b.Binary(Sub, b.Load(0), b.Constant(1))
		b.Store(0, next)
		condition := next
		if !raw {
			condition = b.Binary(Equal, b.Binary(Equal, next, b.Constant(0)), b.Constant(0))
		}
		b.Store(1, condition)
		b.Store(2, b.Binary(Xor, condition, b.Constant(0x1234)))
		b.Store(4, b.Choose(condition, b.Constant(0), b.Constant(4)))
		b.Checkpoint(5, 1)
		b.Store(3, b.MemoryLoad(b.Constant(4096), 8))
		b.Checkpoint(5, 2)
		b.LoopContinue(condition)
		entry, err := n.CompileLoop(b.FinishMemory(5), 5, 2)
		if err != nil {
			t.Fatal(err)
		}
		for _, initial := range []uint64{1, 3, 100} {
			for budget := uint64(1); budget <= 64; budget++ {
				for _, fault := range []bool{false, true} {
					m := new(MemoryContext)
					epoch := uint64(1)
					page := new([4096]byte)
					if !fault {
						m.Fill(1, page, 1, &epoch)
					}
					m.ClaimLinks(n)
					m.PublishLink(0, entry, 2, [17]uint8{})
					state := []uint64{initial, 99, 98, 97, 0}
					if err = n.CallLinked(state, m, budget); err != nil {
						t.Fatal(err)
					}
					iterations := budget / 2
					if iterations > initial {
						iterations = initial
					}
					if iterations == 0 {
						if state[0] != initial || state[1] != 99 || m.Total != 0 {
							t.Fatal("unadmitted body changed state", raw, state, m.Total)
						}
						continue
					}
					if fault {
						iterations = 1
					}
					remaining := initial - iterations
					predicate := remaining
					if !raw && remaining != 0 {
						predicate = 1
					}
					pc := uint64(0)
					if predicate == 0 {
						pc = 4
					}
					retired, memory, status, completed := iterations*2, iterations, uint64(0), iterations
					if fault {
						retired, memory, status, completed = 1, 0, 1, 0
					}
					if state[0] != remaining || state[1] != predicate || state[2] != predicate^0x1234 || state[4] != pc || m.Total != retired || m.MemoryTotal != memory || m.Status != status || m.LoopIterations != completed {
						t.Fatal("predicate reconstruction", raw, fault, initial, budget, state, m.Total, m.MemoryTotal, m.Status, m.LoopIterations)
					}
				}
			}
		}
	}
}
