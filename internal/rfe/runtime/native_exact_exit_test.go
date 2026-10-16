//go:build !renvo

package runtime

import (
	"math/bits"
	"reflect"
	"testing"
)

func scalarOtherExitPredicates(a, b uint64, width int, logical bool) uint64 {
	negative := func(x uint64) bool {
		if width == 32 {
			return int32(x) < 0
		}
		return int64(x) < 0
	}
	var r, c uint64
	if width == 32 {
		a, b = uint64(uint32(a)), uint64(uint32(b))
		x, y := bits.Add32(uint32(a), uint32(b), 0)
		r, c = uint64(x), uint64(y)
	} else {
		r, c = bits.Add64(a, b, 0)
	}
	v := negative(a) == negative(b) && negative(r) != negative(a)
	if logical {
		r, c, v = a, 0, false
	}
	n, z := negative(r), r == 0
	less := n != v
	conditions := []bool{z, !z, c != 0, c == 0, n, !n, v, !v, c != 0 && !z, c == 0 || z, !less, less, !less && !z, less || z}
	var packed uint64
	for k, on := range conditions {
		if on {
			packed |= 1 << k
		}
	}
	return packed
}

// Exact raw equality, logical zero, and SUB equality may eliminate cold flag
// calculations. A 32-bit equality must NEVER replace an architectural high word.
// The pre-guard fault has the same output roots but none of the later facts.
func TestNativeExactExitValuesAndConstantFlags(t *testing.T) {
	const words, pc, count = 8, 7, 4
	constants := []uint64{0, 1, 0x7fffffff, 0x80000000, 0xffffffff, 0x7fffffffffffffff, 1 << 63, ^uint64(0)}
	for _, width := range []int{32, 64} {
		for _, constant := range constants {
			for mode := 0; mode < 5; mode++ {
				n, err := NewNative(1 << 20)
				if err != nil {
					t.Fatal(err)
				}
				if err = n.PrepareLinks(words, pc); err != nil {
					t.Fatal(err)
				}
				var b Builder
				a, rhs := b.Load(0), b.Constant(constant)
				packed := b.Constant(0)
				flags := []Value{b.ArithmeticFlags(a, rhs, width, false), b.ArithmeticFlags(a, rhs, width, true), b.LogicalFlags(a, width)}
				for group, f := range flags {
					for k := 0; k < 14; k++ {
						p, ok := b.FlagCondition(f, k)
						if !ok {
							t.Fatal("condition")
						}
						packed = b.Binary(Or, packed, b.Shift(Shl, p, uint64(group*14+k)))
					}
				}
				control := b.Binary(Equal, a, rhs)
				if mode == 1 {
					control = b.Binary(Equal, rhs, a)
				}
				if mode == 2 {
					control, _ = b.FlagCondition(flags[2], 0)
				}
				if mode == 3 {
					control, _ = b.FlagCondition(flags[1], 0)
				}
				if mode == 4 {
					control = a
				}
				b.Store(2, packed)
				b.Store(pc, b.Choose(control, b.Constant(0), b.Constant(4)))
				b.Checkpoint(words, 1)
				b.Store(4, b.MemoryLoad(b.Load(5), 1))
				b.Checkpoint(words, 2)
				b.RegionGuard(control)
				b.Store(3, b.Binary(Xor, a, b.Constant(0x123456789abcdef0)))
				b.Checkpoint(words, 3)
				b.Store(4, b.MemoryLoad(b.Load(6), 1))
				b.Store(pc, b.Constant(8))
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
				var page [4096]byte
				page[0] = 0xa7
				epoch := uint64(1)
				m.Fill(1, &page, 1, &epoch)
				m.PublishLink(0, entry, count, [17]uint8{})
				for _, a := range []uint64{constant, constant ^ 1, constant | 0xfeed123400000000, 0, 2, 7} {
					for fault := 0; fault < 3; fault++ {
						for _, foreign := range []bool{false, true} {
							if foreign && !NativeSessionsAvailable() {
								continue
							}
							for _, budget := range []uint64{1, 3, 4, 5, 64, 65} {
								if !foreign && budget > 64 {
									continue
								}
								state := []uint64{a, 19, 23, 29, 31, 4096, 4096, 0}
								if fault == 1 {
									state[5] = 8192
								}
								if fault == 2 {
									state[6] = 8192
								}
								want := append([]uint64(nil), state...)
								var retired, memories, status, address uint64
								if budget >= count {
									add := scalarOtherExitPredicates(a, constant, width, false)
									sub := scalarSubPredicates(a, constant, width)
									log := scalarOtherExitPredicates(a, constant, width, true)
									want[2] = add | sub<<14 | log<<28
									control := a == constant
									if mode == 2 {
										control = log&1 != 0
									}
									if mode == 3 {
										control = sub&1 != 0
									}
									if mode == 4 {
										control = a != 0
									}
									want[pc] = 0
									if !control {
										want[pc] = 4
									}
									retired = 1
									if fault == 1 {
										status, address = 1, 8192
									} else {
										want[4], retired, memories = 0xa7, 2, 1
										if control {
											want[3], retired = a^0x123456789abcdef0, 3
											if fault == 2 {
												status, address = 1, 8192
											} else {
												want[pc], retired, memories = 8, count, 2
											}
										}
									}
								}
								if foreign {
									err = n.RunLinkedSession(state, m, budget)
								} else {
									err = n.CallLinked(state, m, budget)
								}
								if err != nil || !reflect.DeepEqual(state, want) || m.Total != retired || m.MemoryTotal != memories || m.Remaining != budget-retired || m.Status != status || status != 0 && m.Address != address {
									t.Fatalf("width=%d constant=%x mode=%d a=%x fault=%d foreign=%v budget=%d err=%v state=%x want=%x retired=%d/%d memory=%d/%d status=%d/%d", width, constant, mode, a, fault, foreign, budget, err, state, want, m.Total, retired, m.MemoryTotal, memories, m.Status, status)
								}
							}
						}
					}
				}
				n.Close()
			}
		}
	}
}

func TestNativeRawLoopConditionZeroAndNonzero(t *testing.T) {
	const words, pc, count = 4, 3, 2
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(words, pc); err != nil {
		t.Fatal(err)
	}
	var b Builder
	next := b.Shift(Shr, b.Load(0), 1)
	b.Store(0, next)
	b.Store(1, b.Binary(Mul, next, b.Constant(3)))
	b.Store(2, b.Shift(Shr, next, 31))
	b.Store(pc, b.Choose(next, b.Constant(0), b.Constant(4)))
	b.Checkpoint(words, count)
	b.LoopContinue(next)
	entry, err := n.CompileLoop(b.FinishMemory(words), words, count)
	if err != nil {
		t.Fatal(err)
	}
	if err = n.PrepareTargetLink(0, entry, words, count); err != nil {
		t.Fatal(err)
	}
	m := new(MemoryContext)
	m.ClaimLinks(n)
	m.PublishLink(0, entry, count, [17]uint8{})
	for _, raw := range []uint64{0, 1, 2, 7, 1 << 63, ^uint64(0)} {
		for _, foreign := range []bool{false, true} {
			if foreign && !NativeSessionsAvailable() {
				continue
			}
			for _, budget := range []uint64{1, 2, 3, 4, 5, 63, 64, 65, 65536} {
				if !foreign && budget > 64 {
					continue
				}
				state := []uint64{raw, 71, 73, 0}
				want := append([]uint64(nil), state...)
				var retired uint64
				for retired+count <= budget {
					want[0] >>= 1
					want[1], want[2] = want[0]*3, want[0]>>31
					retired += count
					if want[0] == 0 {
						want[pc] = 4
						break
					}
				}
				if foreign {
					err = n.RunLinkedSession(state, m, budget)
				} else {
					err = n.CallLinked(state, m, budget)
				}
				if err != nil || !reflect.DeepEqual(state, want) || m.Total != retired || m.Remaining != budget-retired || m.Status != 0 {
					t.Fatal(raw, foreign, budget, err, state, want, m.Total, retired, m.Status)
				}
			}
		}
	}
}
