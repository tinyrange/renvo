//go:build !renvo

package runtime

import (
	"encoding/binary"
	"reflect"
	"renvo.dev/internal/rfenativebridge"
	"testing"
)

// The third iteration crosses onto a differently protected page. In particular,
// a readable/non-writable page must commit the read's architectural checkpoint
// before failing the write; it must never fail at the earlier read prefix.
// Widths include a wider store, which must not reuse a narrow read's bound.
func TestNativePairedPagesCrossingAndPreciseFaults(t *testing.T) {
	if !NativeSessionsAvailable() {
		t.Skip("foreign session unavailable")
	}
	const words, pc, count = 8, 7, 4
	for _, widths := range [][2]int{{8, 8}, {8, 4}, {8, 1}, {1, 8}} {
		n, err := NewNative(1 << 20)
		if err != nil {
			t.Fatal(err)
		}
		if err = n.PrepareLinks(words, pc); err != nil {
			t.Fatal(err)
		}
		m := new(MemoryContext)
		epoch := uint64(1)
		m.Clock = &epoch
		m.ClaimLinks(n)
		var b Builder
		address := b.Load(0)
		value := b.MemoryLoad(address, widths[0])
		b.Store(2, value)
		b.Checkpoint(words, 1)
		b.MemoryStore(address, b.Binary(Xor, value, b.Constant(0x9081726354453627)), widths[1])
		b.Checkpoint(words, 2)
		b.Store(0, b.Binary(Add, address, b.Load(1)))
		iterations := b.Binary(Add, b.Load(4), b.Constant(1))
		b.Store(4, iterations)
		condition := b.Binary(Less, iterations, b.Load(5))
		b.Store(pc, b.Choose(condition, b.Constant(0), b.Constant(4)))
		b.Checkpoint(words, count)
		b.LoopContinue(condition)
		entry, err := n.CompileCodeVersionedLoop(b.FinishMemory(words), words, count, m)
		if err != nil {
			t.Fatal(err)
		}
		if err = n.PrepareTargetLink(0, entry, words, count); err != nil {
			t.Fatal(err)
		}
		for _, foreign := range []bool{false, true} {
			for _, permissions := range [][2]uint64{{3, 3}, {3, 1}, {3, 2}, {1, 3}, {2, 3}, {0, 3}, {3, 0}, {3, 3}} {
				for _, start := range []uint64{8176, 8185, 1<<48 | 8176, ^uint64(0) - 7} {
					for _, budget := range []uint64{1, 3, 4, 7, 8, 11, 12, 19, 20, 21, 64, 65536} {
						if !foreign && budget > 64 {
							continue
						}
						backing := make([]byte, 3*4096)
						for i := range backing {
							backing[i] = byte(i*37 + 13)
						}
						expected := append([]byte(nil), backing...)
						state := []uint64{start, 8, 0x1111222233334444, 71, 0, 5, 97, 0}
						want := append([]uint64(nil), state...)
						var retired, memories, status, fault uint64
						for retired+count <= budget && want[4] < want[5] {
							at := want[0]
							valid := func(width int, permission uint64) bool {
								page, off := at>>12, at&4095
								return page >= 1 && page <= 2 && off <= 4096-uint64(width) && permissions[page-1]&permission != 0
							}
							if !valid(widths[0], 1) {
								status, fault = 1, at
								break
							}
							var value uint64
							if widths[0] == 8 {
								value = binary.LittleEndian.Uint64(expected[at:])
							} else {
								value = uint64(expected[at])
							}
							want[2] = value
							retired++
							memories++
							if !valid(widths[1], 2) {
								status, fault = 1, at
								break
							}
							value ^= 0x9081726354453627
							switch widths[1] {
							case 1:
								expected[at] = byte(value)
							case 4:
								binary.LittleEndian.PutUint32(expected[at:], uint32(value))
							case 8:
								binary.LittleEndian.PutUint64(expected[at:], value)
							}
							retired += 3
							memories++
							want[0] += want[1]
							want[4]++
							if want[4] == want[5] {
								want[pc] = 4
							}
						}
						for page := 1; page <= 2; page++ {
							m.Fill(uint64(page), (*[4096]byte)(backing[page*4096:]), permissions[page-1], &epoch)
						}
						m.PublishLink(0, entry, count, [17]uint8{})
						if foreign {
							var expired bool
							expired, err = n.RunLinkedBackingSessionUntil(state, m, budget, backing, rfenativebridge.MonotonicNS()+1000000000)
							if expired {
								t.Fatal("unexpected expiry")
							}
						} else {
							err = n.CallLinked(state, m, budget)
						}
						if err != nil || !reflect.DeepEqual(state, want) || !reflect.DeepEqual(backing, expected) || m.Total != retired || m.MemoryTotal != memories || m.Remaining != budget-retired || m.Status != status || status != 0 && m.Address != fault {
							t.Fatalf("widths=%v foreign=%v permissions=%v start=%x budget=%d err=%v state=%x want=%x retired=%d/%d mem=%d/%d status=%d/%d fault=%x/%x", widths, foreign, permissions, start, budget, err, state, want, m.Total, retired, m.MemoryTotal, memories, m.Status, status, m.Address, fault)
						}
					}
				}
			}
		}
		n.Close()
	}
}
