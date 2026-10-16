//go:build !renvo

package runtime

import (
	"fmt"
	"reflect"
	"renvo.dev/internal/rfenativebridge"
	"testing"
)

// Compare complete architectural state with scalar uint64 arithmetic. Boolean
// wrappers, unrelated live arithmetic, high words, and the low-word MOV mask
// distinguish the real predicate from unrelated or partial-width host flags.
func TestNativeFinalArithmeticTruth(t *testing.T) {
	if !NativeSessionsAvailable() {
		t.Skip("foreign session unavailable")
	}
	for _, op := range []int{Add, Sub, And, Or, Xor} {
		for _, wrap := range []int{0, 1, 2} {
			for _, clobber := range []bool{false, true} {
				t.Run(fmt.Sprintf("%d/%d/%t", op, wrap, clobber), func(t *testing.T) {
					const words, pc = 8, 7
					n, err := NewNative(4 << 20)
					if err != nil {
						t.Fatal(err)
					}
					defer n.Close()
					if err = n.PrepareLinks(words, pc); err != nil {
						t.Fatal(err)
					}
					m := new(MemoryContext)
					epoch := uint64(1)
					m.Clock = &epoch
					m.ClaimLinks(n)
					var b Builder
					x, y := b.Load(0), b.Load(1)
					next := b.Binary(op, x, y)
					b.Store(0, next)
					cond := next
					if wrap != 0 {
						cond = b.Binary(Equal, cond, b.Constant(0))
					}
					if wrap == 2 {
						cond = b.Binary(Xor, cond, b.Constant(1))
					}
					if clobber {
						b.Store(4, b.Binary(Add, b.Load(5), b.Load(6)))
					}
					b.Store(pc, b.Choose(cond, b.Constant(0), b.Constant(4)))
					b.Checkpoint(words, 9)
					b.LoopContinue(cond)
					entry, err := n.CompileCodeVersionedLoop(b.FinishMemory(words), words, 9, m)
					if err != nil {
						t.Fatal(err)
					}
					if err = n.PrepareTargetLink(0, entry, words, 9); err != nil {
						t.Fatal(err)
					}
					m.PublishLink(0, entry, 9, [17]uint8{})
					for _, pair := range [][2]uint64{{0, 0}, {1, 1}, {0xffffffffffffffff, 1}, {0x100000000, 0xffffffff}, {0x100000001, 1}, {0x8000000000000000, 0x8000000000000000}} {
						for _, budget := range []uint64{9, 10, 63, 4095, 4096, 4097, 65536} {
							state := make([]uint64, words)
							state[0], state[1], state[5] = pair[0], pair[1], pair[0]^1
							want := append([]uint64(nil), state...)
							retired, iters := uint64(0), uint64(0)
							for retired+9 <= budget {
								x, y := want[0], want[1]
								switch op {
								case Add:
									x += y
								case Sub:
									x -= y
								case And:
									x &= y
								case Or:
									x |= y
								case Xor:
									x ^= y
								}
								want[0] = x
								if clobber {
									want[4] = want[5] + want[6]
								}
								truth := x != 0
								if wrap == 1 {
									truth = !truth
								}
								retired += 9
								iters++
								if !truth {
									want[pc] = 4
									break
								}
							}
							expired, err := n.RunLinkedBackingSessionUntil(state, m, budget, nil, rfenativebridge.MonotonicNS()+1000000000)
							if err != nil || expired || !reflect.DeepEqual(state, want) || m.Status != 0 || m.Total != retired || m.Remaining != budget-retired || m.LoopIterations != iters {
								t.Fatalf("pair=%x budget=%d err=%v expired=%t got=%x want=%x total=%d/%d remaining=%d iterations=%d/%d", pair, budget, err, expired, state, want, m.Total, retired, m.Remaining, m.LoopIterations, iters)
							}
						}
					}
				})
			}
		}
	}
}
