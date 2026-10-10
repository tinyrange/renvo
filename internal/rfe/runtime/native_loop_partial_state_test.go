//go:build !renvo

package runtime

import "testing"

// Preserved bits may be invariant while the other bits still need the previous
// iteration's value on an early fault, or the current value on a later fault.
func TestNativeLoopPartialStatePreciseFaults(t *testing.T) {
	const mask = uint64(0xff00ff00f0f00000)
	const initial = uint64(0xa55aa55afedcba98)
	for _, mode := range []int{0, 1, 2, 3} {
		n, err := NewNative(1 << 20)
		if err != nil {
			t.Fatal(err)
		}
		if err = n.PrepareLinks(5, 4); err != nil {
			t.Fatal(err)
		}
		var b Builder
		address, old := b.Load(0), b.Load(1)
		original := b.MemoryLoad(address, 1)
		b.MemoryStore(address, b.Binary(Add, original, b.Constant(1)), 1)
		b.Checkpoint(5, 1)
		preserved := b.Binary(And, old, b.Constant(mask))
		if mode == 2 {
			preserved = b.Binary(Xor, preserved, b.Constant(1<<20))
		}
		packed := b.Binary(Or, preserved, b.Binary(And, b.Binary(Mul, original, b.Constant(37)), b.Constant(^mask)))
		b.Store(1, packed)
		if mode == 3 {
			b.Store(2, old)
		}
		b.Checkpoint(5, 2)
		if mode == 1 {
			b.Store(3, b.MemoryLoad(b.Binary(Add, address, b.Constant(8)), 1))
		}
		b.Checkpoint(5, 3)
		b.Store(0, b.Binary(Add, address, b.Constant(8)))
		b.Checkpoint(5, 4)
		b.LoopContinue(b.Constant(1))
		entry, err := n.CompileLoop(b.FinishMemory(5), 5, 4)
		if err != nil {
			t.Fatal(err)
		}
		for _, start := range []uint64{8176, 8192} {
			for _, budget := range []uint64{4, 8, 20} {
				page := new([4096]byte)
				page[4080], page[4088] = 9, 217
				clock, epoch := uint64(1), uint64(1)
				m := &MemoryContext{NativeContext: NativeContext{Clock: &clock}}
				m.Fill(1, page, 3, &epoch)
				m.ClaimLinks(n)
				m.PublishLink(0, entry, 4, [17]uint8{})
				state := []uint64{start, initial, 55, 66, 0}
				if err = n.CallLinked(state, m, budget); err != nil {
					t.Fatal(err)
				}
				want := []uint64{start, initial, 55, 66, 0}
				total, stores := uint64(0), uint64(0)
				fault := false
				for remaining := budget; remaining >= 4; remaining -= 4 {
					if want[0] >= 8192 {
						fault = true
						break
					}
					value := uint64(9)
					if want[0] == 8184 {
						value = 217
					}
					old := want[1]
					preserved := old & mask
					if mode == 2 {
						preserved ^= 1 << 20
					}
					want[1] = preserved | (value*37)&^mask
					if mode == 3 {
						want[2] = old
					}
					stores++
					total += 2
					if mode == 1 {
						if want[0]+8 >= 8192 {
							fault = true
							break
						}
						want[3] = 217
					}
					want[0] += 8
					total += 2
				}
				for i := range want {
					if state[i] != want[i] {
						t.Fatalf("mode=%d start=%d budget=%d slot=%d got=%x want=%x", mode, start, budget, i, state[i], want[i])
					}
				}
				if m.Total != total || clock != 1+stores || (m.Status == 1) != fault {
					t.Fatalf("mode=%d start=%d budget=%d total=%d/%d clock=%d/%d fault=%v status=%d", mode, start, budget, m.Total, total, clock, 1+stores, fault, m.Status)
				}
			}
		}
		n.Close()
	}
}
