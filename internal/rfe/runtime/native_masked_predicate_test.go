//go:build !renvo

package runtime

import (
	"reflect"
	"testing"
)

// A masked terminator may be inverted only when normalized. Fault exits and
// exhausted budgets must still reconstruct all original (possibly wide) SSA.
func TestNativeMaskedPredicatePreciseExits(t *testing.T) {
	const words = 7
	n, err := NewNative(4 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(words, 6); err != nil {
		t.Fatal(err)
	}
	for _, shift := range []uint64{0, 1, 31, 63, 64} {
		for _, mask := range []uint64{1, 2, 1 << 31, 1 << 63, 3} {
			for _, invert := range []bool{false, true} {
				if invert && mask != 1 {
					continue
				}
				var b Builder
				next := b.Binary(Sub, b.Load(0), b.Constant(1))
				shifted := b.Shift(Shr, next, shift)
				masked := b.Binary(And, shifted, b.Constant(mask))
				condition := masked
				if invert {
					condition = b.Binary(Xor, masked, b.Constant(1))
				}
				b.Store(0, next)
				b.Store(1, shifted)
				b.Store(2, masked)
				b.Store(3, condition)
				b.Store(4, b.Binary(Xor, masked, b.Constant(0x1234)))
				b.Store(6, b.Choose(condition, b.Constant(0), b.Constant(4)))
				b.Checkpoint(words, 1)
				b.Store(5, b.MemoryLoad(b.Constant(4096), 1))
				b.Checkpoint(words, 2)
				b.LoopContinue(condition)
				entry, err := n.CompileLoop(b.FinishMemory(words), words, 2)
				if err != nil {
					t.Fatal(err)
				}
				for _, initial := range []uint64{0, 1, 2, 3, 0x80000001, 0x8000000000000001, ^uint64(0)} {
					for budget := uint64(1); budget <= 35; budget++ {
						for _, fault := range []bool{false, true} {
							m := new(MemoryContext)
							epoch := uint64(1)
							page := new([4096]byte)
							page[0] = 0xa5
							if !fault {
								m.Fill(1, page, 1, &epoch)
							}
							m.ClaimLinks(n)
							m.PublishLink(0, entry, 2, [17]uint8{})
							state := []uint64{initial, 91, 92, 93, 94, 95, 0}
							want := append([]uint64(nil), state...)
							var retired, memories, iterations, exits, status uint64
							for retired+2 <= budget {
								want[0]--
								want[1] = want[0] >> shift
								want[2] = want[1] & mask
								want[3] = want[2]
								if invert {
									want[3] ^= 1
								}
								want[4] = want[2] ^ 0x1234
								want[6] = 0
								if want[3] == 0 {
									want[6] = 4
								}
								retired++
								if fault {
									status = 1
									break
								}
								want[5] = 0xa5
								retired++
								memories++
								iterations++
								if want[3] == 0 {
									exits = 1
									break
								}
							}
							if err = n.CallLinked(state, m, budget); err != nil {
								t.Fatal(err)
							}
							if !reflect.DeepEqual(state, want) || m.Total != retired || m.Remaining != budget-retired || m.MemoryTotal != memories || m.LoopIterations != iterations || m.LoopExits != exits || m.Status != status || status != 0 && m.Address != 4096 {
								t.Fatalf("shift=%d mask=%x invert=%v initial=%x budget=%d fault=%v state=%x want=%x total=%d/%d memory=%d/%d iterations=%d/%d exits=%d/%d status=%d/%d address=%x", shift, mask, invert, initial, budget, fault, state, want, m.Total, retired, m.MemoryTotal, memories, m.LoopIterations, iterations, m.LoopExits, exits, m.Status, status, m.Address)
							}
						}
					}
				}
			}
		}
	}
}
