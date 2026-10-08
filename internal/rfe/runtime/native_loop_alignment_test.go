//go:build !renvo

package runtime

import "testing"

// Aligned and unaligned bases execute the same scalar semantics. Exercise
// boundary-crossing slow paths, wrapping addresses, store permissions and
// exhausted generation clocks after precise committed prefixes.
func TestNativeLoopAlignmentVersionsAgainstScalar(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(5, 4); err != nil {
		t.Fatal(err)
	}
	var initialPage [4096]byte
	for i := range initialPage {
		initialPage[i] = byte(i*37 + 129)
	}
	page := new([4096]byte)
	epoch, clock := uint64(0), uint64(0)
	m := &MemoryContext{Clock: &clock}
	m.ClaimLinks(n)
	for shift := 1; shift <= 4; shift++ {
		size := 1 << uint(shift)
		var b Builder
		index := b.Binary(And, b.Load(0), b.Constant(0xffffffff))
		address := b.Binary(Add, b.Load(1), b.Shift(Shl, index, uint64(shift)))
		loaded, high := Value(0), Value(0)
		if size == 16 {
			loaded, high = b.MemoryPairLoad(address)
			b.Store(3, high)
		} else {
			loaded = b.MemoryLoad(address, size)
		}
		b.Store(2, loaded)
		b.Checkpoint(5, 1)
		next := b.Binary(Add, loaded, b.Constant(1))
		if size == 16 {
			highNext := b.Binary(Add, high, b.Constant(1))
			b.MemoryPairStore(address, next, highNext)
			b.Store(3, highNext)
		} else {
			b.MemoryStore(address, next, size)
		}
		b.Store(2, next)
		b.Store(0, b.Binary(And, b.Binary(Add, index, b.Constant(1)), b.Constant(0xffffffff)))
		b.Store(4, b.Constant(0))
		b.Checkpoint(5, 3)
		b.LoopContinue(b.Constant(1))
		entry, err := n.CompileLoop(b.FinishMemory(5), 5, 3)
		if err != nil {
			t.Fatal(err)
		}
		m.PublishLink(0, entry, 3, [17]uint8{})
		for _, prepared := range []bool{false, true} {
			if prepared {
				if err = n.PrepareTargetLink(0, entry, 5, 3); err != nil {
					t.Fatal(err)
				}
			}
			for budget := uint64(1); budget <= 64; budget++ {
				for _, initial := range []uint64{0, 0xffffffff, 0xdeadbeefffffffff} {
					for _, offset := range []uint64{0, 1, 4096 - uint64(size), 4097 - uint64(size), 4095} {
						for _, perms := range []uint64{0, 1, 3, 7} {
							for _, startClock := range []uint64{1, ^uint64(0) - 1, ^uint64(0)} {
								*page = initialPage
								expectedPage := initialPage
								epoch, clock = 0, startClock
								expectedEpoch, expectedClock := epoch, clock
								m.Fill(0, page, perms, &epoch)
								base := offset - (uint64(uint32(initial)) << uint(shift))
								state := []uint64{initial, base, 99, 98, 0}
								expected := append([]uint64(nil), state...)
								retired, mem, iterations, status, fault := uint64(0), uint64(0), uint64(0), uint64(0), uint64(0)
								for retired+3 <= budget {
									at := base + (uint64(uint32(expected[0])) << uint(shift))
									if at > 4096-uint64(size) || perms&1 == 0 {
										status, fault = 1, at
										break
									}
									lowSize := size
									if lowSize == 16 {
										lowSize = 8
									}
									value := uint64(0)
									for j := 0; j < lowSize; j++ {
										value |= uint64(expectedPage[int(at)+j]) << uint(j*8)
									}
									if size == 16 {
										expected[3] = 0
										for j := 0; j < 8; j++ {
											expected[3] |= uint64(expectedPage[int(at)+8+j]) << uint(j*8)
										}
									}
									expected[2] = value
									retired++
									mem++
									if perms&6 != 2 || expectedClock == ^uint64(0) {
										status, fault = 1, at
										break
									}
									expectedClock++
									expectedEpoch = expectedClock
									for j := 0; j < lowSize; j++ {
										expectedPage[int(at)+j] = byte((value + 1) >> uint(j*8))
									}
									if size == 16 {
										expected[3]++
										for j := 0; j < 8; j++ {
											expectedPage[int(at)+8+j] = byte(expected[3] >> uint(j*8))
										}
									}
									expected[2] = value + 1
									expected[0] = uint64(uint32(expected[0] + 1))
									retired += 2
									mem++
									iterations++
								}
								if err = n.CallLinked(state, m, budget); err != nil {
									t.Fatal(err)
								}
								for i := range state {
									if state[i] != expected[i] {
										t.Fatalf("prepared=%v size=%d budget=%d initial=%x offset=%d perms=%d clock=%x state=%x want=%x", prepared, size, budget, initial, offset, perms, startClock, state, expected)
									}
								}
								if *page != expectedPage || clock != expectedClock || epoch != expectedEpoch || m.Total != retired || m.Remaining != budget-retired || m.MemoryTotal != mem || m.LoopIterations != iterations || m.Status != status || status == 1 && m.Address != fault {
									t.Fatalf("prepared=%v size=%d budget=%d offset=%d perms=%d clock=%x: accounting/memory mismatch total=%d/%d mem=%d/%d status=%d/%d fault=%x/%x", prepared, size, budget, offset, perms, startClock, m.Total, retired, m.MemoryTotal, mem, m.Status, status, m.Address, fault)
								}
							}
						}
					}
				}
			}
		}
	}
}
