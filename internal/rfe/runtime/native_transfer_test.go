//go:build !renvo

package runtime

import (
	"runtime"
	"testing"
)

// The scalar model does not execute IR or use native exit maps. Long-lived
// inputs force spills, slot zero crosses edges, cold selects consume its OLD
// input, and an unread slot zero must inherit a prior cyclic iteration on trap.
func TestNativeTransferScalarOracle(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("transfer ABI is amd64 only")
	}
	const words, pcSlot = 24, 23
	engines := [2]*Native{}
	entries := [2][2]int{}
	for engine := range engines {
		n, err := NewNative(1 << 20)
		if err != nil {
			t.Fatal(err)
		}
		defer n.Close()
		engines[engine] = n
		if err = n.PrepareLinks(words, pcSlot); err != nil {
			t.Fatal(err)
		}
		for block := 0; block < 2; block++ {
			var b Builder
			if block == 0 {
				old := make([]Value, 20)
				for i := range old {
					old[i] = b.Load(i)
				}
				for i := range old {
					b.Store(i, b.Binary(Add, old[(i+1)%20], b.Constant(uint64(i+1))))
				}
				b.Store(0, b.Choose(b.Binary(And, old[0], b.Constant(1)), old[19], old[1]))
				b.Store(pcSlot, b.Constant(4))
				b.Checkpoint(words, 1)
				b.RegionGuard(b.Binary(Equal, b.Binary(And, b.Load(21), b.Constant(1)), b.Constant(0)))
				b.Store(0, b.Binary(Xor, b.Load(0), b.Constant(0xfedcba9876543210)))
				b.Store(21, b.Binary(Add, b.Load(21), b.Constant(1)))
				b.Checkpoint(words, 2)
				b.LoopContinue(b.Constant(0))
			} else {
				left := b.Binary(Sub, b.Load(22), b.Constant(1))
				b.Store(22, left)
				b.Store(pcSlot, b.Constant(8))
				b.Checkpoint(words, 1)
				b.Guard(b.Binary(Equal, b.Binary(Equal, left, b.Constant(0)), b.Constant(0)))
				// Deliberately no input for slot zero: trap inheritance needs a capture.
				b.Store(0, b.Choose(b.Binary(And, b.Load(1), b.Constant(1)), b.Load(2), b.Load(3)))
				b.Store(1, b.Binary(Add, b.Load(1), b.Constant(3)))
				condition := b.Binary(Equal, b.Binary(Equal, left, b.Constant(1)), b.Constant(0))
				b.Store(pcSlot, b.Choose(condition, b.Constant(4), b.Constant(0)))
				b.Checkpoint(words, 3)
				b.LoopContinue(condition)
			}
			ops := b.FinishMemory(words)
			if engine == 0 {
				entries[engine][block], err = n.CompileLoop(ops, words, block+2)
			} else {
				exits := []uint64{4}
				if block == 1 {
					exits = []uint64{0}
				}
				entries[engine][block], err = n.CompileLoopChained(ops, words, block+2, pcSlot, exits)
			}
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, mode := range []string{"hit", "collision", "absent", "wrong-count", "wrong-entry", "clear"} {
		for engine, n := range engines {
			for block := 0; block < 2; block++ {
				if err := n.PrepareTargetLink(uint64(block*4), entries[engine][block], words, block+2); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "collision" {
				if err := n.PrepareTargetLink(4100, entries[engine][1], words, 3); err != nil {
					t.Fatal(err)
				}
			}
		}
		for budget := uint64(1); budget <= 64; budget++ {
			for seed := uint64(0); seed < 6; seed++ {
				initial := make([]uint64, words)
				for i := range initial {
					initial[i] = ^uint64(0) - uint64(i)*0x1020304050607 - seed
				}
				initial[21], initial[22], initial[pcSlot] = seed&1, seed+1, 0
				want := append([]uint64(nil), initial...)
				var retired, status uint64
				validB := mode == "hit" || mode == "collision"
				for retired < budget {
					pc := want[pcSlot]
					if mode == "clear" || pc == 4 && !validB {
						break
					}
					cost := uint64(2)
					if pc == 4 {
						cost = 3
					} else if pc != 0 {
						break
					}
					if budget-retired < cost {
						break
					}
					if pc == 0 {
						old := append([]uint64(nil), want...)
						for i := 0; i < 20; i++ {
							want[i] = old[(i+1)%20] + uint64(i+1)
						}
						want[0] = old[1]
						if old[0]&1 != 0 {
							want[0] = old[19]
						}
						want[pcSlot] = 4
						retired++
						if want[21]&1 == 0 {
							want[0] ^= 0xfedcba9876543210
							want[21]++
							retired++
						}
					} else {
						want[22]--
						want[pcSlot] = 8
						retired++
						if want[22] == 0 {
							status = 2
							break
						}
						want[0] = want[3]
						if want[1]&1 != 0 {
							want[0] = want[2]
						}
						want[1] += 3
						want[pcSlot] = 4
						if want[22] == 1 {
							want[pcSlot] = 0
						}
						retired += 2
					}
				}
				states := [2][]uint64{}
				contexts := [2]*MemoryContext{}
				for engine, n := range engines {
					m := new(MemoryContext)
					m.ClaimLinks(n)
					m.PublishLink(0, entries[engine][0], 2, [17]uint8{})
					m.PublishLink(4, entries[engine][1], 3, [17]uint8{})
					switch mode {
					case "absent":
						m.Blocks[1] = NativeLink{}
					case "wrong-count":
						m.Blocks[1].Instructions = 1
					case "wrong-entry":
						m.Blocks[1].Entry++
					case "clear":
						m.ClearLinks()
					}
					states[engine] = append([]uint64(nil), initial...)
					contexts[engine] = m
					if err := n.CallLinked(states[engine], m, budget); err != nil {
						t.Fatal(mode, budget, seed, err)
					}
					for slot, value := range want {
						if states[engine][slot] != value {
							t.Fatalf("%s budget=%d seed=%d engine=%d slot=%d got=%x want=%x", mode, budget, seed, engine, slot, states[engine][slot], value)
						}
					}
					if m.Total != retired || m.Remaining != budget-retired || m.Status != status || m.MemoryTotal != 0 || m.CodeView != [4]uint64{} || m.PreparedTargets != 0 {
						t.Fatal("scalar progress", mode, budget, seed, engine, m.Total, retired, m.Status, status)
					}
				}
				a, b := contexts[0], contexts[1]
				if a.Retired != b.Retired || a.LoopIterations != b.LoopIterations || a.LoopExits != b.LoopExits || a.Address != b.Address {
					t.Fatal("differential accounting", mode, budget, seed, a.Retired, b.Retired, a.LoopIterations, b.LoopIterations, a.LoopExits, b.LoopExits)
				}
			}
		}
	}
}
