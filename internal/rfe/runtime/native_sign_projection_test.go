//go:build !renvo

package runtime

import (
	"reflect"
	"testing"
)

// Signed projections participate in loop phis and precise fault snapshots.
// The scalar oracle uses typed conversions, not the generated shift identity.
// Observing the intermediate shift forbids eliminating it.
func TestNativeSignedProjectionLoopPrefixes(t *testing.T) {
	for _, bits := range []int{8, 16, 32} {
		for _, observe := range []bool{false, true} {
			n, e := NewNative(1 << 20)
			if e != nil {
				t.Fatal(e)
			}
			const words = 8
			if e = n.PrepareLinks(words, 5); e != nil {
				t.Fatal(e)
			}
			var b Builder
			source := b.Binary(Mul, b.Load(0), b.Load(3))
			shifted := b.Shift(Shl, source, uint64(64-bits))
			if observe {
				b.Store(2, shifted)
			}
			// Keep the full source observable too, and interleave another value
			// so the fused consumer must retain its operand past the old shift.
			b.Store(6, b.Binary(Xor, source, b.Load(7)))
			value := b.VariableShift(ArithmeticShr, shifted, b.Constant(uint64(128-bits)), 64)
			b.Store(0, b.Binary(Add, value, b.Load(7)))
			b.Store(1, b.Binary(Add, b.Load(1), value))
			b.Checkpoint(words, 2)
			b.MemoryLoad(b.Load(4), 1)
			b.Store(5, b.Constant(0))
			b.Checkpoint(words, 3)
			b.LoopContinue(b.Constant(1))
			entry, e := n.CompileLoop(b.FinishMemory(words), words, 3)
			if e != nil {
				t.Fatal(e)
			}
			m := new(MemoryContext)
			var page [4096]byte
			epoch := uint64(1)
			m.Fill(1, &page, 1, &epoch)
			m.ClaimLinks(n)
			m.PublishLink(0, entry, 3, [17]uint8{})
			if e = n.PrepareTargetLink(0, entry, words, 3); e != nil {
				t.Fatal(e)
			}
			x := uint64(0x89abcdef01234567)
			for trial := 0; trial < 40; trial++ {
				x = x*6364136223846793005 + 1
				for _, foreign := range []bool{false, true} {
					if foreign && !NativeSessionsAvailable() {
						continue
					}
					for _, budget := range []uint64{1, 2, 3, 5, 6, 63, 64, 65, 4095, 65536} {
						if !foreign && budget > 64 {
							continue
						}
						for _, fault := range []bool{false, true} {
							address := uint64(4096)
							if fault {
								address = 8192
							}
							state := []uint64{x, 17, 99, x | 1, address, 0, 55, 0xdeadbeef00000011}
							want := append([]uint64(nil), state...)
							actual := budget
							if !foreign && actual > 64 {
								actual = 64
							}
							var total, status uint64
							for total+3 <= actual {
								raw := want[0] * want[3]
								var signed uint64
								switch bits {
								case 8:
									signed = uint64(int64(int8(raw)))
								case 16:
									signed = uint64(int64(int16(raw)))
								case 32:
									signed = uint64(int64(int32(raw)))
								}
								if observe {
									want[2] = raw << uint(64-bits)
								}
								want[6] = raw ^ want[7]
								want[0] = signed + want[7]
								want[1] += signed
								if fault {
									total += 2
									status = 1
									break
								}
								total += 3
							}
							if foreign {
								e = n.RunLinkedSession(state, m, budget)
							} else {
								e = n.CallLinked(state, m, budget)
							}
							if e != nil || !reflect.DeepEqual(state, want) || m.Total != total || m.Remaining != budget-total || m.Status != status {
								t.Fatalf("bits=%d observe=%v foreign=%v fault=%v budget=%d err=%v got=%x want=%x total=%d/%d remaining=%d status=%d/%d", bits, observe, foreign, fault, budget, e, state, want, m.Total, total, m.Remaining, m.Status, status)
							}
						}
					}
				}
			}
			if e = n.Close(); e != nil {
				t.Fatal(e)
			}
		}
	}
}
