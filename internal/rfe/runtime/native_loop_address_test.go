//go:build !renvo

package runtime

import "testing"

// Projection precedes scaling and full-width base addition. Test both the
// successful wrapping sum and later width/page faults across every quantum.
func TestNativeLoopScaledAddressProjectionAndWrapping(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(5, 4); err != nil {
		t.Fatal(err)
	}
	for _, project := range []bool{false, true} {
		for shift := 1; shift <= 3; shift++ {
			for _, multiple := range []bool{false, true} {
				var b Builder
				input := b.Load(0)
				index := input
				if project {
					index = b.Binary(And, index, b.Constant(0xffffffff))
				}
				scaled := b.Shift(Shl, index, uint64(shift))
				address := b.Binary(Add, b.Load(1), scaled)
				if multiple {
					b.Store(3, scaled)
				}
				b.Store(2, b.MemoryLoad(address, 8))
				b.Store(0, b.Binary(Add, input, b.Constant(1)))
				b.Store(4, b.Constant(0))
				b.Checkpoint(5, 1)
				b.LoopContinue(b.Constant(1))
				entry, err := n.CompileLoop(b.FinishMemory(5), 5, 1)
				if err != nil {
					t.Fatal(err)
				}
				page := new([4096]byte)
				for i := range page {
					page[i] = byte(i*37 + 129)
				}
				epoch := uint64(1)
				m := new(MemoryContext)
				m.Fill(0, page, 1, &epoch)
				m.ClaimLinks(n)
				m.PublishLink(0, entry, 1, [17]uint8{})
				for budget := uint64(1); budget <= 64; budget++ {
					for _, input := range []uint64{0, 0xffffffff, 0x100000000, 0xdeadbeef00000000, 1 << 63} {
						for _, offset := range []uint64{0, 4064, 4088, 4092} {
							used := input
							if project {
								used = uint64(uint32(input))
							}
							base := offset - (used << shift)
							state := []uint64{input, base, 99, 98, 0}
							expected := append([]uint64(nil), state...)
							iterations := uint64(0)
							faultAddress := uint64(0)
							for iterations < budget {
								ix := expected[0]
								if project {
									ix = uint64(uint32(ix))
								}
								scale := ix << shift
								at := expected[1] + scale
								if at > 4088 {
									faultAddress = at
									break
								}
								value := uint64(0)
								for j := uint64(0); j < 8; j++ {
									value |= uint64(page[at+j]) << (j * 8)
								}
								if multiple {
									expected[3] = scale
								}
								expected[2] = value
								expected[0]++
								iterations++
							}
							if err = n.CallLinked(state, m, budget); err != nil {
								t.Fatal(err)
							}
							status := uint64(0)
							if iterations < budget {
								status = 1
							}
							for i := range state {
								if state[i] != expected[i] {
									t.Fatalf("projection=%v shift=%d multi=%v input=%x offset=%d budget=%d state=%x expected=%x", project, shift, multiple, input, offset, budget, state, expected)
								}
							}
							if m.Total != iterations || m.MemoryTotal != iterations || m.LoopIterations != iterations || m.Status != status || status == 1 && m.Address != faultAddress {
								t.Fatal("scaled address progress", m.Total, m.MemoryTotal, m.LoopIterations, m.Status, m.Address, faultAddress)
							}
						}
					}
				}
			}
		}
	}
}
