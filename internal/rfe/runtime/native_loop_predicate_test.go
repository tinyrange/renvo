//go:build !renvo

package runtime

import (
	"math/bits"
	"testing"
)

// The oracle derives architectural flags independently of the IR evaluator.
// Exercise every condition, both widths, register/immediate operands, cold
// NZCV/PC selection and boolean inversion while unrelated phis force spills.
func TestNativeLoopFusedPredicatesAndColdMaps(t *testing.T) {
	const words = 25
	n, err := NewNative(4 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(words, 24); err != nil {
		t.Fatal(err)
	}
	inputs := []uint64{0, 1, 0x7fffffff, 0x80000000, 0xffffffff, 0x7fffffffffffffff, 0x8000000000000000, ^uint64(0)}
	for _, width := range []int{32, 64} {
		for kind := 0; kind < 3; kind++ {
			for _, immediate := range []bool{false, true} {
				for cond := 0; cond < 16; cond++ {
					for _, inverted := range []bool{false, true} {
						var b Builder
						live := make([]Value, 19)
						for i := range live {
							live[i] = b.Load(i + 5)
						}
						a, rhs := b.Load(1), b.Load(2)
						if immediate {
							rhs = b.Constant(1)
						}
						flags := b.ArithmeticFlags(a, rhs, width, kind == 1)
						if kind == 2 {
							flags = b.LogicalFlags(a, width)
						}
						condition, ok := b.FlagCondition(flags, cond)
						if !ok {
							t.Fatal("condition")
						}
						if inverted {
							condition = b.Binary(Xor, condition, b.Constant(1))
						}
						b.Store(0, b.Binary(Add, b.Load(0), b.Constant(1)))
						b.Store(3, flags)
						b.Store(4, b.Choose(condition, b.Constant(31), b.Constant(47)))
						for i, v := range live {
							b.Store(i+5, b.Binary(Add, v, b.Constant(uint64(i+1))))
						}
						b.Store(24, b.Choose(condition, b.Constant(0), b.Constant(4)))
						b.Checkpoint(words, 2)
						b.LoopContinue(condition)
						entry, err := n.CompileLoop(b.FinishMemory(words), words, 2)
						if err != nil {
							t.Fatal(err)
						}
						m := new(MemoryContext)
						m.ClaimLinks(n)
						m.PublishLink(0, entry, 2, [17]uint8{})
						for trial := 0; trial < len(inputs)*len(inputs); trial++ {
							a, rhs := inputs[trial%len(inputs)], inputs[trial/len(inputs)]
							state := make([]uint64, words)
							state[1], state[2] = a, rhs
							if immediate {
								rhs = 1
							}
							mask := ^uint64(0)
							if width == 32 {
								mask = 0xffffffff
							}
							a, rhs = a&mask, rhs&mask
							sign := uint64(1) << uint(width-1)
							result := (a + rhs) & mask
							_, carry := bits.Add64(a, rhs, 0)
							c := carry != 0
							if width == 32 {
								c = a+rhs > mask
							}
							v := (a^result)&(rhs^result)&sign != 0
							if kind == 1 {
								result = (a - rhs) & mask
								c = a >= rhs
								v = (a^rhs)&(a^result)&sign != 0
							}
							if kind == 2 {
								result, c, v = a, false, false
							}
							n, z := result&sign != 0, result == 0
							var wantFlags uint64
							for bit, set := range []bool{v, c, z, n} {
								if set {
									wantFlags |= uint64(1) << uint(28+bit)
								}
							}
							truth := []bool{z, !z, c, !c, n, !n, v, !v, c && !z, !c || z, n == v, n != v, n == v && !z, n != v || z, true, true}[cond]
							if inverted {
								truth = !truth
							}
							wantIterations, wantPC, wantSelect, wantExits := uint64(1), uint64(4), uint64(47), uint64(1)
							if truth {
								wantIterations, wantPC, wantSelect, wantExits = 32, 0, 31, 0
							}
							if err = m.linkOwner.CallLinked(state, m, 64); err != nil {
								t.Fatal(err)
							}
							if state[0] != wantIterations || state[3] != wantFlags || state[4] != wantSelect || state[24] != wantPC || m.Total != 2*wantIterations || m.LoopIterations != wantIterations || m.LoopExits != wantExits {
								t.Fatalf("width=%d kind=%d immediate=%v cond=%d inverted=%v trial=%d state=%v total=%d loops=%d exits=%d want flags=%x truth=%v", width, kind, immediate, cond, inverted, trial, state, m.Total, m.LoopIterations, m.LoopExits, wantFlags, truth)
							}
							for i := range live {
								if state[i+5] != uint64(i+1)*wantIterations {
									t.Fatal("predicate clobbered phi", state)
								}
							}
						}
					}
				}
			}
		}
	}
}
