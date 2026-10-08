//go:build !renvo

package runtime

import "testing"

// The control uses the ordinary dispatcher, with independently allocated state,
// pages and generation clocks. Exercise chain hits and every admission fallback.
func TestNativeDirectChainAgainstDispatcher(t *testing.T) {
	for _, base := range []uint64{0, 1 << 40, 0xffff000000000000} {
		engines := [2]*Native{}
		entries := [2][2]int{}
		for engine := range engines {
			n, err := NewNative(1 << 20)
			if err != nil {
				t.Fatal(err)
			}
			defer n.Close()
			engines[engine] = n
			if err = n.PrepareLinks(5, 4); err != nil {
				t.Fatal(err)
			}
			for block := 0; block < 2; block++ {
				var b Builder
				address := b.Load(2)
				if block == 0 {
					value := b.MemoryLoad(address, 8)
					b.Store(1, value)
					b.Store(0, b.Binary(Add, b.Load(0), b.Constant(1)))
					b.Store(4, b.Constant(base+4))
				} else {
					b.Store(0, b.Binary(Add, b.Load(0), b.Constant(2)))
					b.Store(4, b.Constant(base+8))
					b.Checkpoint(5, 1)
					b.MemoryStore(address, b.Binary(Xor, b.Load(1), b.Constant(0x123456789abcdef0)), 8)
					b.Store(2, b.Binary(Add, address, b.Constant(8)))
					b.Store(4, b.Constant(base))
				}
				b.Checkpoint(5, block+2)
				b.LoopContinue(b.Constant(0))
				var entry int
				if engine == 0 {
					entry, err = n.CompileLoop(b.FinishMemory(5), 5, block+2)
				} else {
					target := base + 4
					if block == 1 {
						target = base
					}
					entry, err = n.CompileLoopChained(b.FinishMemory(5), 5, block+2, 4, []uint64{target})
				}
				if err != nil {
					t.Fatal(err)
				}
				entries[engine][block] = entry
				if err = n.PrepareTargetLink(base+uint64(block)*4, entry, 5, block+2); err != nil {
					t.Fatal(err)
				}
			}
		}
		for _, mode := range []string{"hit", "empty", "wrong-pc", "zero-count", "wrong-count", "huge-count", "wrong-entry", "collision", "invalidate"} {
			for _, n := range engines {
				entry := entries[0][1]
				if n == engines[1] {
					entry = entries[1][1]
				}
				if err := n.PrepareTargetLink(base+4, entry, 5, 3); err != nil {
					t.Fatal(err)
				}
				if mode == "collision" {
					if err := n.PrepareTargetLink(base+4100, entry, 5, 3); err != nil {
						t.Fatal(err)
					}
				}
			}
			for budget := uint64(1); budget <= 64; budget++ {
				for _, offset := range []uint64{0, 4080, 4088, 4089} {
					for _, permissions := range []uint64{0, 1, 3, 7} {
						for _, clockStart := range []uint64{1, ^uint64(0) - 1, ^uint64(0)} {
							states := [2][]uint64{}
							contexts := [2]*MemoryContext{}
							pages := [2]*[4096]byte{}
							clocks, epochs := [2]uint64{}, [2]uint64{}
							for engine, n := range engines {
								pages[engine] = new([4096]byte)
								for i := range pages[engine] {
									pages[engine][i] = byte(i*37 + 129)
								}
								clocks[engine], epochs[engine] = clockStart, 17
								m := &MemoryContext{Clock: &clocks[engine]}
								contexts[engine] = m
								m.Fill(1, pages[engine], permissions, &epochs[engine])
								m.ClaimLinks(n)
								m.PublishLink(base, entries[engine][0], 2, [17]uint8{0, 0, 1})
								m.PublishLink(base+4, entries[engine][1], 3, [17]uint8{0, 0, 1, 1})
								link := &m.Blocks[((base+4)>>2)&1023]
								switch mode {
								case "empty":
									*link = NativeLink{}
								case "wrong-pc":
									link.PC += 4096
								case "zero-count":
									link.Instructions = 0
								case "wrong-count":
									link.Instructions = 1
								case "huge-count":
									link.Instructions = ^uint64(0)
								case "wrong-entry":
									link.Entry++
								case "invalidate":
									m.ClearLinks()
								}
								states[engine] = []uint64{123, 99, 4096 + offset, 0, base}
								if err := n.CallLinked(states[engine], m, budget); err != nil {
									t.Fatal(base, mode, budget, offset, permissions, clockStart, err)
								}
							}
							for i, v := range states[0] {
								if states[1][i] != v {
									t.Fatal("state", base, mode, budget, offset, permissions, clockStart, states)
								}
							}
							a, b := contexts[0], contexts[1]
							if a.Retired != b.Retired || a.Total != b.Total || a.Remaining != b.Remaining || a.MemoryTotal != b.MemoryTotal || a.Status != b.Status || a.Address != b.Address || a.LoopIterations != b.LoopIterations || a.LoopExits != b.LoopExits || clocks[0] != clocks[1] || epochs[0] != epochs[1] || *pages[0] != *pages[1] {
								t.Fatalf("accounting base=%x mode=%s budget=%d offset=%d permissions=%d clock=%d: control %d/%d/%d/%d/%d native %d/%d/%d/%d/%d", base, mode, budget, offset, permissions, clockStart, a.Retired, a.Total, a.Remaining, a.MemoryTotal, a.Status, b.Retired, b.Total, b.Remaining, b.MemoryTotal, b.Status)
							}
							for _, m := range contexts {
								if m.PreparedTargets != 0 || m.CodeView != [4]uint64{} {
									t.Fatal("borrowed proof leaked")
								}
							}
						}
					}
				}
			}
		}
	}
}

func TestNativeDirectChainQuantaAndMixedFamily(t *testing.T) {
	for _, mixed := range []bool{false, true} {
		engines := [2]*Native{}
		entries := [2][2]int{}
		for engine := range engines {
			n, err := NewNative(1 << 20)
			if err != nil {
				t.Fatal(err)
			}
			defer n.Close()
			engines[engine] = n
			if err = n.PrepareLinks(2, 1); err != nil {
				t.Fatal(err)
			}
			for block := 0; block < 2; block++ {
				var b Builder
				b.Store(0, b.Binary(Add, b.Load(0), b.Constant(uint64(block+1))))
				target := uint64(4)
				if block == 1 {
					target = 0
				}
				b.Store(1, b.Constant(target))
				var entry int
				if mixed && block == 1 {
					entry, err = n.Compile(b.Finish(2), 2)
				} else {
					b.Checkpoint(2, block+1)
					b.LoopContinue(b.Constant(0))
					if engine == 0 {
						entry, err = n.CompileLoop(b.FinishMemory(2), 2, block+1)
					} else {
						entry, err = n.CompileLoopChained(b.FinishMemory(2), 2, block+1, 1, []uint64{target})
					}
				}
				if err != nil {
					t.Fatal(err)
				}
				entries[engine][block] = entry
				if err = n.PrepareTargetLink(uint64(block*4), entry, 2, block+1); err != nil {
					t.Fatal(err)
				}
			}
		}
		for remaining := uint64(1); remaining <= 1025; remaining++ {
			states := [2][]uint64{}
			contexts := [2]*MemoryContext{}
			completed := [2]int{}
			for engine, n := range engines {
				m := new(MemoryContext)
				contexts[engine] = m
				m.ClaimLinks(n)
				for block := 0; block < 2; block++ {
					m.PublishLink(uint64(block*4), entries[engine][block], block+1, [17]uint8{})
					m.Blocks[block].Reserved = ^uint64(0)
				}
				states[engine] = []uint64{0, 0}
				var err error
				completed[engine], err = n.RunLinkedQuanta(states[engine], m, 16, remaining, 0)
				if err != nil {
					t.Fatal(mixed, remaining, err)
				}
			}
			a, b := contexts[0], contexts[1]
			if states[0][0] != states[1][0] || states[0][1] != states[1][1] || completed[0] != completed[1] || a.Retired != b.Retired || a.Total != b.Total || a.Remaining != b.Remaining || a.Status != b.Status || a.LoopIterations != b.LoopIterations || a.LoopExits != b.LoopExits {
				t.Fatal("quanta", mixed, remaining, states, completed, a.Total, b.Total, a.Remaining, b.Remaining)
			}
		}
	}
}
