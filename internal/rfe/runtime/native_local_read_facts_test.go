//go:build !renvo

package runtime

import (
	"encoding/binary"
	"reflect"
	"renvo.dev/internal/rfenativebridge"
	"testing"
)

// An acyclic body alternates two pages, writes a previously read byte, then
// reads it at a larger width. The scalar oracle sees fresh data and checks each
// permission/width separately, including after warmed calls revoke a page.
func TestNativeAcyclicReadFactsPrecisePrefixes(t *testing.T) {
	for _, returnToB := range []bool{false, true} {
		testNativeAcyclicReadFactsPrecisePrefixes(t, returnToB)
	}
}

func testNativeAcyclicReadFactsPrecisePrefixes(t *testing.T, returnToB bool) {
	if !NativeSessionsAvailable() {
		t.Skip("foreign session unavailable")
	}
	const words, pc, count = 9, 8, 7
	n, err := NewNative(1 << 20)
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
	addresses := []Value{b.Load(0), b.Load(1), b.Load(0), b.Load(0), b.Load(1), b.Load(0)}
	if returnToB {
		// A write uses its own translation bias. A later hit on the
		// unchanged read-page fact must not inherit the write bias.
		addresses[3] = b.Load(1)
	}
	widths := []int{1, 1, 1, 8, 2, 1}
	for phase, address := range addresses {
		if phase == 2 {
			b.MemoryStore(address, b.Constant(0xe9), 1)
		} else {
			b.Store(2+phase, b.MemoryLoad(address, widths[phase]))
		}
		b.Checkpoint(words, phase+1)
	}
	b.Store(pc, b.Constant(4))
	b.Checkpoint(words, count)
	b.LoopContinue(b.Constant(0))
	entry, err := n.CompileCodeVersionedLoop(b.FinishMemory(words), words, count, m)
	if err != nil {
		t.Fatal(err)
	}
	if err = n.PrepareTargetLink(0, entry, words, count); err != nil {
		t.Fatal(err)
	}
	backing := make([]byte, 3*4096)
	for _, foreign := range []bool{false, true} {
		for _, permissions := range [][2]uint64{{3, 3}, {0, 3}, {1, 3}, {2, 3}, {3, 0}, {3, 2}, {3, 3}} {
			for _, address := range [][2]uint64{{4096, 8192}, {8191, 8192}, {8188, 8192}, {1<<48 | 4096, 8192}, {^uint64(0) - 3, 8192}, {4096, 4096}} {
				for _, budget := range []uint64{1, 6, 7, 8, 63, 64, 65536} {
					if !foreign && budget > 64 {
						continue
					}
					for i := range backing {
						backing[i] = byte(i*37 + 11)
					}
					expected := append([]byte(nil), backing...)
					state := []uint64{address[0], address[1], 100, 101, 102, 103, 104, 105, 0}
					want := append([]uint64(nil), state...)
					var retired, memories, status, fault uint64
					if budget >= count {
						for phase, width := range widths {
							at := address[0]
							if phase == 1 || phase == 4 || phase == 3 && returnToB {
								at = address[1]
							}
							page, off := at/4096, at%4096
							permission := uint64(1)
							if phase == 2 {
								permission = 2
							}
							if page < 1 || page > 2 || off > 4096-uint64(width) || permissions[page-1]&permission == 0 {
								status, fault = 1, at
								break
							}
							if phase == 2 {
								expected[at] = 0xe9
							} else {
								var v uint64
								switch width {
								case 1:
									v = uint64(expected[at])
								case 2:
									v = uint64(binary.LittleEndian.Uint16(expected[at:]))
								case 8:
									v = binary.LittleEndian.Uint64(expected[at:])
								}
								want[2+phase] = v
							}
							retired++
							memories++
						}
						if status == 0 {
							retired++
							want[pc] = 4
						}
					}
					epoch++
					for page := 1; page <= 2; page++ {
						m.Fill(uint64(page), (*[4096]byte)(backing[page*4096:]), permissions[page-1], &epoch)
					}
					m.PublishLink(0, entry, count, [17]uint8{})
					if foreign {
						expired, e := n.RunLinkedBackingSessionUntil(state, m, budget, backing, rfenativebridge.MonotonicNS()+1000000000)
						err = e
						if expired {
							t.Fatal("unexpected timer expiry")
						}
					} else {
						err = n.CallLinked(state, m, budget)
					}
					if err != nil || !reflect.DeepEqual(state, want) || !reflect.DeepEqual(backing, expected) || m.Total != retired || m.MemoryTotal != memories || m.Remaining != budget-retired || m.Status != status || status != 0 && m.Address != fault {
						t.Fatal("acyclic scalar prefix", foreign, permissions, address, budget, err, state, want, m.Total, retired, m.MemoryTotal, memories, m.Status, status, m.Address, fault)
					}
				}
			}
		}
	}
}
