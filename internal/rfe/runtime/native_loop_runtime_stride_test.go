//go:build !renvo

package runtime

import "testing"

// Two independent read streams exercise runtime-valued recurrences, W wraps,
// rejected span proofs and precise late faults after a completed checkpoint.
// The scalar oracle deliberately uses guest page permissions and full-width
// address arithmetic rather than the native cache's span reasoning.
func TestNativeLoopRuntimeStrideAndLateFault(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(8, 7); err != nil {
		t.Fatal(err)
	}
	for _, projectedStep := range []bool{false, true} {
		for _, varyingStep := range []bool{false, true} {
			for shift := 1; shift <= 3; shift++ {
				var b Builder
				mask := b.Constant(0xffffffff)
				i, j, step := b.Load(0), b.Load(1), b.Load(4)
				ix, jx := b.Binary(And, i, mask), b.Binary(And, j, mask)
				increment := step
				if projectedStep {
					increment = b.Binary(And, step, mask)
				}
				a := b.Binary(Add, b.Load(5), b.Shift(Shl, ix, uint64(shift)))
				z := b.Binary(Add, b.Load(6), b.Shift(Shl, jx, uint64(shift)))
				b.Store(2, b.MemoryLoad(a, 8))
				b.Checkpoint(8, 1)
				b.Store(3, b.MemoryLoad(z, 8))
				b.Checkpoint(8, 2)
				b.Store(0, b.Binary(And, b.Binary(Add, ix, b.Constant(1)), mask))
				b.Store(1, b.Binary(And, b.Binary(Add, jx, increment), mask))
				if varyingStep {
					b.Store(4, b.Binary(Add, step, b.Constant(1)))
				}
				b.Store(7, b.Constant(0))
				b.Checkpoint(8, 3)
				b.LoopContinue(b.Constant(1))
				entry, err := n.CompileLoop(b.FinishMemory(8), 8, 3)
				if err != nil {
					t.Fatal(err)
				}
				pages := [2]*[4096]byte{new([4096]byte), new([4096]byte)}
				for p, page := range pages {
					for k := range page {
						page[k] = byte(k*37 + p*19 + 129)
					}
				}
				m := new(MemoryContext)
				epoch := uint64(1)
				m.Fill(0, pages[0], 1, &epoch)
				m.ClaimLinks(n)
				m.PublishLink(0, entry, 3, [17]uint8{})
				for budget := uint64(1); budget <= 64; budget++ {
					for _, stride := range []uint64{0, 1, 3, 32, 4096, 0xffffffff, 0x100000001, ^uint64(0)} {
						for _, initial := range []uint64{0, 0xfffffff0, 0xffffffff, 0xdeadbeeffffffff0} {
							for _, offset := range []uint64{0, 4000, 4080, 4088, 4092} {
								for _, readable := range []bool{false, true} {
									permission := uint64(0)
									if readable {
										permission = 1
									}
									m.Fill(1, pages[1], permission, &epoch)
									base := offset - (uint64(uint32(initial)) << shift)
									state := []uint64{initial, initial, 99, 98, stride, base, base + 4096, 0}
									expected := append([]uint64(nil), state...)
									retired, memory, iterations, status, fault := uint64(0), uint64(0), uint64(0), uint64(0), uint64(0)
									for retired+3 <= budget {
										failed := false
										for stream := 0; stream < 2; stream++ {
											at := expected[5+stream] + (uint64(uint32(expected[stream])) << shift)
											page, off := at/4096, at%4096
											if page >= 2 || off > 4088 || page == 1 && !readable {
												status, fault, failed = 1, at, true
												break
											}
											value := uint64(0)
											for k := uint64(0); k < 8; k++ {
												value |= uint64(pages[page][off+k]) << (k * 8)
											}
											expected[2+stream] = value
											retired++
											memory++
										}
										if failed {
											break
										}
										inc := expected[4]
										if projectedStep {
											inc = uint64(uint32(inc))
										}
										expected[0] = uint64(uint32(expected[0] + 1))
										expected[1] = uint64(uint32(expected[1] + inc))
										if varyingStep {
											expected[4]++
										}
										retired++
										iterations++
									}
									if err = n.CallLinked(state, m, budget); err != nil {
										t.Fatal(err)
									}
									for k := range state {
										if state[k] != expected[k] {
											t.Fatalf("project=%v varying=%v shift=%d budget=%d stride=%x initial=%x offset=%d readable=%v state=%x expected=%x", projectedStep, varyingStep, shift, budget, stride, initial, offset, readable, state, expected)
										}
									}
									if m.Total != retired || m.MemoryTotal != memory || m.LoopIterations != iterations || m.Status != status || status == 1 && m.Address != fault {
										t.Fatalf("progress budget=%d: total=%d/%d memory=%d/%d iterations=%d/%d status=%d/%d fault=%x/%x", budget, m.Total, retired, m.MemoryTotal, memory, m.LoopIterations, iterations, m.Status, status, m.Address, fault)
									}
								}
							}
						}
					}
				}
			}
		}
	}
}
