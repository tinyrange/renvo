//go:build !renvo

package runtime

import "testing"

// Exercise preserved bits alongside enough loop-carried words to force spills.
// Both loads can fault: the first needs the previous iteration's flags, while
// the second needs the current flags and truncated copy. The oracle does not
// use the native IR or its register allocation.
func TestNativeLoopColdInputPressureAndFaults(t *testing.T) {
	const words = 16
	const mask = uint64(0xff00ff00ffff0000)
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(words, 15); err != nil {
		t.Fatal(err)
	}
	var b Builder
	address, flags := b.Load(0), b.Load(1)
	counter := b.Load(2)
	values := make([]Value, 10)
	for i := range values {
		values[i] = b.Load(i + 3)
	}
	first := b.MemoryLoad(address, 1)
	b.Store(14, b.Binary(And, values[0], b.Constant(0xffffffff)))
	for i, value := range values {
		b.Store(i+3, b.Binary(Add, value, b.Binary(Add, first, b.Constant(uint64(i+1)))))
	}
	next := b.Binary(Add, counter, b.Constant(1))
	b.Store(1, b.Binary(Or, b.Binary(And, flags, b.Constant(mask)), b.Binary(And, next, b.Constant(^mask))))
	b.Store(2, next)
	b.Checkpoint(words, 2)
	b.Store(13, b.MemoryLoad(b.Binary(Add, address, b.Constant(8)), 1))
	b.Store(0, b.Binary(Add, address, b.Constant(8)))
	b.Checkpoint(words, 4)
	b.LoopContinue(b.Constant(1))
	entry, err := n.CompileLoop(b.FinishMemory(words), words, 4)
	if err != nil {
		t.Fatal(err)
	}
	if err = n.PrepareTargetLink(0, entry, words, 4); err != nil {
		t.Fatal(err)
	}
	for _, session := range []bool{false, true} {
		if session && !NativeSessionsAvailable() {
			continue
		}
		for _, start := range []uint64{8168, 8184, 8192} {
			for _, budget := range []uint64{4, 8, 13, 64} {
				page := new([4096]byte)
				page[4072], page[4080], page[4088] = 7, 19, 43
				epoch := uint64(1)
				m := new(MemoryContext)
				m.Fill(1, page, 1, &epoch)
				m.ClaimLinks(n)
				m.PublishLink(0, entry, 4, [17]uint8{})
				state := make([]uint64, words)
				state[0], state[1], state[2] = start, 0xa55aa55afedcba98, 0xffff
				for i := 3; i < 15; i++ {
					state[i] = 0x12345678ffffffff + uint64(i)
				}
				want := append([]uint64(nil), state...)
				total, memory, iterations, status := uint64(0), uint64(0), uint64(0), uint64(0)
				for budget-total >= 4 {
					if want[0] >= 8192 {
						status = 1
						break
					}
					value := uint64(page[want[0]-4096])
					memory++
					want[14] = uint64(uint32(want[3]))
					for i := 3; i < 13; i++ {
						want[i] += value + uint64(i-2)
					}
					want[2]++
					want[1] = want[1]&mask | want[2]&^mask
					total += 2
					if want[0]+8 >= 8192 {
						status = 1
						break
					}
					want[13] = uint64(page[want[0]+8-4096])
					memory++
					want[0] += 8
					total += 2
					iterations++
				}
				if session {
					err = n.RunLinkedSession(state, m, budget)
				} else {
					err = n.CallLinked(state, m, budget)
				}
				if err != nil {
					t.Fatal(err)
				}
				for i := range want {
					if state[i] != want[i] {
						t.Fatalf("session=%v start=%d budget=%d slot=%d got=%x want=%x", session, start, budget, i, state[i], want[i])
					}
				}
				if m.Total != total || m.Remaining != budget-total || m.MemoryTotal != memory || m.LoopIterations != iterations || m.Status != status {
					t.Fatalf("session=%v start=%d budget=%d progress=%d/%d memory=%d/%d iterations=%d/%d status=%d/%d", session, start, budget, m.Total, total, m.MemoryTotal, memory, m.LoopIterations, iterations, m.Status, status)
				}
			}
		}
	}
}
