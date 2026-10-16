//go:build !renvo

package runtime

import (
	"reflect"
	"testing"
)

// Acyclic traces end with a constant terminator, not the earlier conditions.
// Earlier fault maps must not borrow facts from later guards. A non-Boolean
// successful guard proves nonzero, not one, and must preserve its full value.
func TestNativeDominatingGuardsRespectFaultOrder(t *testing.T) {
	const words, pc, count = 10, 9, 6
	pairs := [][2]uint64{{1, 1}, {2, 1}, {1, 2}, {0x7fffffff, 0xffffffff}, {0x80000000, 1}, {1 << 63, 1}, {0x7fffffffffffffff, ^uint64(0)}, {0x1234567800000001, 0xdeadbeef00000001}, {^uint64(0), 0x7fffffffffffffff}}
	for _, width := range []int{32, 64} {
		for known := 0; known < 14; known++ {
			n, err := NewNative(2 << 20)
			if err != nil {
				t.Fatal(err)
			}
			if err = n.PrepareLinks(words, pc); err != nil {
				t.Fatal(err)
			}
			var b Builder
			a, rhs := b.Load(0), b.Load(1)
			pack := func(lhs, rhs Value, width int) (Value, []Value) {
				flags := b.ArithmeticFlags(lhs, rhs, width, true)
				value := b.Constant(0)
				predicates := []Value{}
				for k := 0; k < 14; k++ {
					v, ok := b.FlagCondition(flags, k)
					if !ok {
						t.Fatal("predicate construction")
					}
					predicates = append(predicates, v)
					value = b.Binary(Or, value, b.Shift(Shl, v, uint64(k)))
				}
				return value, predicates
			}
			first, one := pack(a, rhs, width)
			second, two := pack(b.Binary(Xor, a, b.Constant(0x100000001)), b.Binary(Add, rhs, b.Constant(3)), 96-width)
			b.Store(2, first)
			b.Store(pc, b.Choose(one[known], b.Constant(0), b.Constant(4)))
			b.Checkpoint(words, 1)
			b.Store(7, b.MemoryLoad(b.Load(5), 1))
			b.Checkpoint(words, 2)
			b.RegionGuard(one[known])
			b.Store(3, second)
			b.Store(pc, b.Choose(two[known], b.Constant(0), b.Constant(8)))
			b.Checkpoint(words, 3)
			b.RegionGuard(two[known])
			b.Store(pc, b.Choose(b.Load(4), b.Constant(0), b.Constant(12)))
			b.Checkpoint(words, 4)
			b.RegionGuard(b.Load(4))
			b.Store(8, b.Binary(Mul, b.Load(4), b.Constant(3)))
			b.Checkpoint(words, 5)
			b.Store(7, b.MemoryLoad(b.Load(6), 1))
			b.Store(pc, b.Constant(16))
			b.Checkpoint(words, count)
			b.LoopContinue(b.Constant(0))
			entry, err := n.CompileLoop(b.FinishMemory(words), words, count)
			if err != nil {
				t.Fatal(err)
			}
			if err = n.PrepareTargetLink(0, entry, words, count); err != nil {
				t.Fatal(err)
			}
			m := new(MemoryContext)
			m.ClaimLinks(n)
			epoch := uint64(1)
			var page [4096]byte
			page[0] = 0x9b
			m.Fill(1, &page, 1, &epoch)
			m.PublishLink(0, entry, count, [17]uint8{})
			for _, pair := range pairs {
				for _, raw := range []uint64{0, 2, 7} {
					for fault := 0; fault < 3; fault++ {
						for _, foreign := range []bool{false, true} {
							if foreign && !NativeSessionsAvailable() {
								continue
							}
							for _, budget := range []uint64{1, 5, 6, 7, 64, 65} {
								if !foreign && budget > 64 {
									continue
								}
								state := []uint64{pair[0], pair[1], 91, 92, raw, 4096, 4096, 93, 94, 0}
								if fault == 1 {
									state[5] = 8192
								} else if fault == 2 {
									state[6] = 8192
								}
								want := append([]uint64(nil), state...)
								var total, memory, status, address uint64
								if budget >= count {
									p := scalarSubPredicates(pair[0], pair[1], width)
									q := scalarSubPredicates(pair[0]^0x100000001, pair[1]+3, 96-width)
									a, c := p&(1<<known) != 0, q&(1<<known) != 0
									want[2] = p
									want[pc] = 0
									if !a {
										want[pc] = 4
									}
									total = 1
									if fault == 1 {
										status, address = 1, 8192
									} else {
										want[7], total, memory = 0x9b, 2, 1
										if a {
											want[3], want[pc], total = q, 0, 3
											if !c {
												want[pc] = 8
											} else {
												total = 4
												if raw == 0 {
													want[pc] = 12
												} else {
													want[8], total = raw*3, 5
													if fault == 2 {
														status, address = 1, 8192
													} else {
														want[pc], total, memory = 16, 6, 2
													}
												}
											}
										}
									}
								}
								if foreign {
									err = n.RunLinkedSession(state, m, budget)
								} else {
									err = n.CallLinked(state, m, budget)
								}
								if err != nil || !reflect.DeepEqual(state, want) || m.Total != total || m.Remaining != budget-total || m.MemoryTotal != memory || m.Status != status || status != 0 && m.Address != address {
									t.Fatal("guard/fault ordering", width, known, pair, raw, fault, foreign, budget, err, state, want, m.Total, total, m.MemoryTotal, memory, m.Status, status)
								}
							}
						}
					}
				}
			}
			n.Close()
		}
	}
}
