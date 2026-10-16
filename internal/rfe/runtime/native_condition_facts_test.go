//go:build !renvo

package runtime

import (
	"reflect"
	"testing"
)

// Typed comparisons are an independent oracle for every subtraction predicate.
// In particular, a signed overflow must not be mistaken for a result sign, and
// equal low words do not make full-width or different-source predicates equal.
func scalarSubPredicates(a, b uint64, width int) uint64 {
	if width == 32 {
		a, b = uint64(uint32(a)), uint64(uint32(b))
	}
	r := a - b
	if width == 32 {
		r = uint64(uint32(r))
	}
	negative := func(v uint64) bool {
		if width == 32 {
			return int32(v) < 0
		}
		return int64(v) < 0
	}
	sa, sb, sr := negative(a), negative(b), negative(r)
	overflow := sa != sb && sr != sa
	signedLess := sa != sb && sa || sa == sb && a < b
	conditions := []bool{a == b, a != b, a >= b, a < b, sr, !sr, overflow, !overflow, a > b, a <= b, !signedLess, signedLess, !signedLess && a != b, signedLess || a == b}
	var packed uint64
	for bit, on := range conditions {
		if on {
			packed |= 1 << bit
		}
	}
	return packed
}

func TestNativeBranchImpliedComparisonFacts(t *testing.T) {
	const words, pc, count = 8, 7, 3
	pairs := [][2]uint64{
		{1, 1}, {2, 1}, {1, 2}, {0, 1}, {1, 0},
		{0x7fffffff, 0xffffffff}, {0x80000000, 1},
		{0x80000000, 0x80000000}, {0xffffffff, 0x7fffffff},
		{0x7fffffffffffffff, ^uint64(0)}, {1 << 63, 1},
		{1 << 63, 1 << 63}, {^uint64(0), 0x7fffffffffffffff},
		{0x1234567800000001, 0xdeadbeef00000001},
		{0xfedcba9887654321, 0x0123456789abcdef},
	}
	for _, width := range []int{32, 64} {
		for known := 0; known < 14; known++ {
			for _, guard := range []bool{false, true} {
				n, err := NewNative(4 << 20)
				if err != nil {
					t.Fatal(err)
				}
				if err = n.PrepareLinks(words, pc); err != nil {
					t.Fatal(err)
				}
				var b Builder
				a, rhs := b.Load(0), b.Load(1)
				flags := b.ArithmeticFlags(a, rhs, width, true)
				otherWidth := 96 - width
				other := b.ArithmeticFlags(a, rhs, otherWidth, true)
				different := b.ArithmeticFlags(a, b.Binary(Add, rhs, b.Constant(1)), width, true)
				pack := func(flags Value) (Value, []Value) {
					v := b.Constant(0)
					conditions := []Value{}
					for condition := 0; condition < 14; condition++ {
						q, ok := b.FlagCondition(flags, condition)
						if !ok {
							t.Fatal("predicate construction")
						}
						conditions = append(conditions, q)
						v = b.Binary(Or, v, b.Shift(Shl, q, uint64(condition)))
					}
					return v, conditions
				}
				packed, conditions := pack(flags)
				otherPacked, _ := pack(other)
				differentPacked, _ := pack(different)
				control := conditions[known]
				// Exercise both supported Boolean inversion wrappers too.
				wrapper := known % 3
				if wrapper == 1 {
					control = b.Binary(Xor, control, b.Constant(1))
				} else if wrapper == 2 {
					control = b.Binary(Equal, control, b.Constant(0))
				}
				b.Store(2, packed)
				b.Store(3, b.Binary(Or, otherPacked, b.Shift(Shl, differentPacked, 16)))
				b.Store(4, b.Binary(Add, b.Load(4), b.Constant(1)))
				b.Store(pc, b.Choose(control, b.Constant(0), b.Constant(4)))
				b.Checkpoint(words, 1)
				if guard {
					b.RegionGuard(control)
				}
				b.Store(6, b.MemoryLoad(b.Load(5), 1))
				b.Store(0, b.Binary(Xor, a, b.Constant(0x8000000000000001)))
				b.Checkpoint(words, count)
				b.LoopContinue(control)
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
				page[0] = 0xd7
				epoch := uint64(1)
				m.Fill(1, &page, 1, &epoch)
				m.PublishLink(0, entry, count, [17]uint8{})
				for _, pair := range pairs {
					for _, foreign := range []bool{false, true} {
						if foreign && !NativeSessionsAvailable() {
							continue
						}
						for _, budget := range []uint64{1, 2, 3, 4, 8, 17, 64, 65} {
							if !foreign && budget > 64 {
								continue
							}
							for _, fault := range []bool{false, true} {
								address := uint64(4096)
								if fault {
									address = 8192
								}
								state := []uint64{pair[0], pair[1], 91, 92, 93, address, 94, 0}
								want := append([]uint64(nil), state...)
								var retired, memories, iterations, exits, status uint64
								for retired+count <= budget {
									packed := scalarSubPredicates(want[0], want[1], width)
									control := packed&(1<<known) != 0
									if wrapper != 0 {
										control = !control
									}
									want[2] = packed
									want[3] = scalarSubPredicates(want[0], want[1], otherWidth) | scalarSubPredicates(want[0], want[1]+1, width)<<16
									want[4]++
									want[pc] = 0
									if !control {
										want[pc] = 4
									}
									if guard && !control {
										retired++
										exits++
										break
									}
									if fault {
										retired++
										status = 1
										break
									}
									want[6] = 0xd7
									want[0] ^= 0x8000000000000001
									retired += count
									memories++
									iterations++
									if !control {
										exits++
										break
									}
								}
								if foreign {
									err = n.RunLinkedSession(state, m, budget)
								} else {
									err = n.CallLinked(state, m, budget)
								}
								if err != nil || !reflect.DeepEqual(state, want) || m.Total != retired || m.Remaining != budget-retired || m.MemoryTotal != memories || m.LoopIterations != iterations || m.LoopExits != exits || m.Status != status || status != 0 && m.Address != address {
									t.Fatalf("width=%d known=%d guard=%v pair=%x foreign=%v budget=%d fault=%v err=%v state=%x want=%x total=%d/%d memory=%d/%d loops=%d/%d exits=%d/%d status=%d/%d", width, known, guard, pair, foreign, budget, fault, err, state, want, m.Total, retired, m.MemoryTotal, memories, m.LoopIterations, iterations, m.LoopExits, exits, m.Status, status)
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
