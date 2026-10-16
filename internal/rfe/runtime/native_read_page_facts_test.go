//go:build !renvo

package runtime

import (
	"encoding/binary"
	"reflect"
	"renvo.dev/internal/rfenativebridge"
	"testing"
)

// Three changing-address streams share one bounded body. The scalar oracle
// checks every memory checkpoint after warming all three pages, then revokes
// each page between fresh calls. No translation cache is consulted by the oracle.
func TestNativeThreeReadPagesPrecisePrefixes(t *testing.T) {
	for _, independent := range []bool{false, true} {
		testNativeThreeReadPagesPrecisePrefixes(t, independent)
	}
}

func testNativeThreeReadPagesPrecisePrefixes(t *testing.T, independent bool) {
	if !NativeSessionsAvailable() {
		t.Skip("foreign session unavailable")
	}
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	const words = 8
	if err = n.PrepareLinks(words, 6); err != nil {
		t.Fatal(err)
	}
	b := Builder{MemoryInstructionProgress: true}
	address := b.Load(0)
	for stream := 0; stream < 3; stream++ {
		at := b.Binary(Add, address, b.Constant(uint64(stream*4096)))
		if independent && stream == 1 {
			// A separately supplied base exercises different SSA address streams;
			// the scalar oracle still accesses the exact same three pages.
			at = b.Binary(Add, b.Load(7), b.Binary(And, address, b.Constant(4095)))
		}
		v := b.MemoryLoad(at, 8)
		same := b.MemoryLoad(at, 1)
		b.Store(1+stream, b.Binary(Add, b.Load(1+stream), b.Binary(Add, v, same)))
		b.Checkpoint(words, stream+1)
	}
	b.Store(0, b.Binary(Add, b.Constant(4096), b.Binary(And, b.Binary(Add, address, b.Constant(8)), b.Constant(4095))))
	b.Store(4, b.Binary(Add, b.Load(4), b.Constant(1)))
	condition := b.Binary(Less, b.Load(4), b.Load(5))
	b.Store(6, b.Choose(condition, b.Constant(0), b.Constant(4)))
	b.Checkpoint(words, 4)
	b.LoopContinue(condition)
	backing := make([]byte, 4*4096)
	for page := 1; page < 4; page++ {
		for off := 0; off < 4096; off += 8 {
			binary.LittleEndian.PutUint64(backing[page*4096+off:], uint64(page*10000+off))
		}
	}
	m := new(MemoryContext)
	epoch := uint64(1)
	m.Clock = &epoch
	m.ClaimLinks(n)
	entry, err := n.CompileCodeVersionedLoop(b.FinishMemory(words), words, 4, m)
	if err != nil {
		t.Fatal(err)
	}
	if err = n.PrepareTargetLink(0, entry, words, 4); err != nil {
		t.Fatal(err)
	}
	for _, foreign := range []bool{false, true} {
		for _, denied := range []int{0, 1, 2, 3, 0} {
			for page := 1; page < 4; page++ {
				permissions := uint64(1)
				if page == denied {
					permissions = 0
				}
				m.Fill(uint64(page), (*[4096]byte)(backing[page*4096:]), permissions, &epoch)
			}
			for _, start := range []uint64{4096, 8184, 8191, 1<<48 | 4096} {
				for _, budget := range []uint64{1, 3, 4, 5, 7, 8, 11, 12, 13, 63, 64, 65, 4096, 65536} {
					if !foreign && budget > 64 {
						continue
					}
					state := []uint64{start, 13, 17, 19, 0, 1000, 0, 8192}
					want := append([]uint64(nil), state...)
					var retired, memories, status, fault uint64
					for retired+4 <= budget && want[4] < want[5] {
						failed := false
						for stream := 0; stream < 3; stream++ {
							address := want[0] + uint64(stream*4096)
							page, off := address/4096, address%4096
							if page < 1 || page > 3 || off > 4088 || int(page) == denied {
								status, fault, failed = 1, address, true
								break
							}
							want[1+stream] += binary.LittleEndian.Uint64(backing[address:address+8]) + uint64(backing[address])
							retired++
							memories++
						}
						if failed {
							break
						}
						want[0] = 4096 + (want[0]+8)&4095
						want[4]++
						retired++
						if want[4] == want[5] {
							want[6] = 4
						}
					}
					m.PublishLink(0, entry, 4, [17]uint8{})
					if foreign {
						hit, e := n.RunLinkedBackingSessionUntil(state, m, budget, backing, rfenativebridge.MonotonicNS()+1000000000)
						err = e
						if hit {
							t.Fatal("unexpected expiry")
						}
					} else {
						err = n.CallLinked(state, m, budget)
					}
					if err != nil || !reflect.DeepEqual(state, want) || m.Total != retired || m.Remaining != budget-retired || m.Status != status || m.MemoryTotal != memories || status != 0 && m.Address != fault {
						t.Fatalf("foreign=%v deny=%d start=%x budget=%d err=%v state=%x want=%x total=%d/%d remaining=%d status=%d/%d memories=%d/%d fault=%x/%x", foreign, denied, start, budget, err, state, want, m.Total, retired, m.Remaining, m.Status, status, m.MemoryTotal, memories, m.Address, fault)
					}
				}
			}
		}
	}
}
