//go:build !renvo

package runtime

import "testing"

func TestNativeSignedLoadWidthsAndObservableIntermediate(t *testing.T) {
	for _, size := range []int{1, 2, 4} {
		for _, width := range []int{32, 64} {
			for _, observeRaw := range []bool{false, true} {
				n, err := NewNative(1 << 20)
				if err != nil {
					t.Fatal(err)
				}
				if err = n.PrepareLinks(5, 4); err != nil {
					t.Fatal(err)
				}
				var b Builder
				raw := b.MemoryLoad(b.Load(0), size)
				if observeRaw {
					b.Store(2, raw)
					b.Checkpoint(5, 1)
				}
				sign := b.Constant(uint64(1) << (size*8 - 1))
				value := b.Binary(Sub, b.Binary(Xor, raw, sign), sign)
				if width == 32 {
					value = b.Binary(And, value, b.Constant(0xffffffff))
				}
				b.Store(1, value)
				// Keep the result live across a later guarded access and many live inputs.
				b.Checkpoint(5, 1)
				b.MemoryLoad(b.Load(3), 1)
				b.Store(4, b.Constant(4))
				b.Checkpoint(5, 2)
				b.LoopContinue(b.Constant(0))
				entry, err := n.CompileLoop(b.FinishMemory(5), 5, 2)
				if err != nil {
					t.Fatal(err)
				}
				var page [4096]byte
				m := new(MemoryContext)
				epoch := uint64(1)
				m.Fill(1, &page, 1, &epoch)
				m.ClaimLinks(n)
				m.PublishLink(0, entry, 2, [17]uint8{0, 1, 2})
				x := uint64(0xfeed12345678)
				for trial := 0; trial < 1024; trial++ {
					x = x*6364136223846793005 + 1
					for i := 0; i < size; i++ {
						page[i] = byte(x >> (i * 8))
					}
					rawWant := x & ((uint64(1) << (size * 8)) - 1)
					var want uint64
					switch size {
					case 1:
						want = uint64(int64(int8(rawWant)))
					case 2:
						want = uint64(int64(int16(rawWant)))
					case 4:
						want = uint64(int64(int32(rawWant)))
					}
					if width == 32 {
						want = uint64(uint32(want))
					}
					for _, fault := range []bool{false, true} {
						address := uint64(4096 + 8)
						if fault {
							address = 8192
						}
						state := []uint64{4096, 0, 77, address, 0}
						if err := n.CallLinked(state, m, 64); err != nil {
							t.Fatal(err)
						}
						status, total, accesses, pc := uint64(0), uint64(2), uint64(2), uint64(4)
						if fault {
							status, total, accesses, pc = 1, 1, 1, 0
						}
						rawResult := uint64(77)
						if observeRaw {
							rawResult = rawWant
						}
						if state[1] != want || state[2] != rawResult || state[4] != pc || m.Status != status || m.Total != total || m.MemoryTotal != accesses || m.CodeView != [4]uint64{} || m.PreparedTargets != 0 {
							t.Fatalf("size=%d width=%d observe=%v fault=%v raw=%x want=%x state=%v status=%d total=%d memory=%d", size, width, observeRaw, fault, rawWant, want, state, m.Status, m.Total, m.MemoryTotal)
						}
					}
				}
				if err := n.Close(); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}
