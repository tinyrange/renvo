//go:build !renvo

package runtime

import (
	"reflect"
	"testing"
)

// All fourteen independent subtraction predicates stay live together under
// spill pressure. A fault after their checkpoint must preserve every result,
// including full-width source operands and values live across the comparison.
func TestNativeAdjacentPredicatesSpillsAndFaultPrefix(t *testing.T) {
	const words, pc, count = 64, 63, 3
	pairs := [][2]uint64{{0, 0}, {0, 1}, {1, 0}, {0x7fffffff, 0xffffffff}, {0x80000000, 1}, {0x8000000000000000, 1}, {0x7fffffffffffffff, ^uint64(0)}, {0x1234567800000001, 0xdeadbeef00000001}, {^uint64(0), 0}}
	for _, width := range []int{32, 64} {
		for _, constant := range []bool{false, true} {
			n, err := NewNative(4 << 20)
			if err != nil {
				t.Fatal(err)
			}
			if err = n.PrepareLinks(words, pc); err != nil {
				t.Fatal(err)
			}
			var b Builder
			a, rhs := b.Load(0), b.Load(1)
			if constant {
				rhs = b.Constant(0xffffffff)
			}
			pressure := make([]Value, 24)
			for i := range pressure {
				pressure[i] = b.Binary(Xor, b.Load(2+i), b.Constant(uint64(0x103+i)))
			}
			flags := b.ArithmeticFlags(a, rhs, width, true)
			predicates := make([]Value, 14)
			for c := range predicates {
				q, ok := b.FlagCondition(flags, c)
				if !ok {
					t.Fatal("predicate")
				}
				predicates[c] = q
			}
			// The next two operations are full-word even when the flag
			// predicates above deliberately compare only their low words.
			b.Constant(0)
			difference := b.Binary(Sub, a, rhs)
			equal := b.Binary(Equal, rhs, a)
			b.Store(47, difference)
			b.Store(48, equal)
			for i, q := range predicates {
				b.Store(32+i, q)
			}
			for i, q := range pressure {
				b.Store(2+i, q)
			}
			b.Checkpoint(words, 1)
			b.Store(61, b.MemoryLoad(b.Load(62), 1))
			b.Store(0, b.Binary(Xor, a, b.Constant(0x8123456789abcdef)))
			b.Store(pc, b.Constant(4))
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
			page[0] = 0xd7
			epoch := uint64(1)
			m.Fill(1, &page, 1, &epoch)
			m.PublishLink(0, entry, count, [17]uint8{})
			for _, pair := range pairs {
				for _, fault := range []bool{false, true} {
					for _, foreign := range []bool{false, true} {
						if foreign && !NativeSessionsAvailable() {
							continue
						}
						for _, budget := range []uint64{1, 2, 3, 4, 64, 65} {
							if !foreign && budget > 64 {
								continue
							}
							state := make([]uint64, words)
							for i := range state {
								state[i] = uint64(i*17 + 3)
							}
							state[0], state[1], state[pc], state[62] = pair[0], pair[1], 0, 4096
							if fault {
								state[62] = 8192
							}
							want := append([]uint64(nil), state...)
							retired, status, memories := uint64(0), uint64(0), uint64(0)
							if budget >= count {
								y := pair[1]
								if constant {
									y = 0xffffffff
								}
								want[47] = pair[0] - y
								want[48] = 0
								if pair[0] == y {
									want[48] = 1
								}
								bits := scalarSubPredicates(pair[0], y, width)
								for c := range predicates {
									want[32+c] = (bits >> uint(c)) & 1
								}
								for i := range pressure {
									want[2+i] ^= uint64(0x103 + i)
								}
								retired = 1
								if fault {
									status = 1
								} else {
									retired, memories = count, 1
									want[61], want[0], want[pc] = 0xd7, pair[0]^0x8123456789abcdef, 4
								}
							}
							if foreign {
								err = n.RunLinkedSession(state, m, budget)
							} else {
								err = n.CallLinked(state, m, budget)
							}
							if err != nil || !reflect.DeepEqual(state, want) || m.Total != retired || m.Remaining != budget-retired || m.Status != status || m.MemoryTotal != memories {
								t.Fatalf("width=%d constant=%t pair=%x fault=%t foreign=%t budget=%d err=%v state=%x want=%x progress=%d/%d status=%d/%d memory=%d/%d", width, constant, pair, fault, foreign, budget, err, state, want, m.Total, retired, m.Status, status, m.MemoryTotal, memories)
							}
						}
					}
				}
			}
			n.Close()
		}
	}
}
