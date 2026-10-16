//go:build !renvo

package runtime

import (
	"reflect"
	"renvo.dev/internal/rfenativebridge"
	"testing"
)

// One installed owned entry must observe policy changes between fresh calls.
// The scalar oracle includes a committed effect before the read, a late denied
// store, malformed full tags, and a wide load crossing the page boundary.
func TestNativeOwnedHashModeChangesBetweenCalls(t *testing.T) {
	if !NativeSessionsAvailable() {
		t.Skip("foreign sessions unavailable")
	}
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	const words, pc, count = 5, 4, 4
	if err = n.PrepareLinks(words, pc); err != nil {
		t.Fatal(err)
	}
	m := new(MemoryContext)
	epoch := uint64(1)
	m.Clock = &epoch
	m.ClaimLinks(n)
	var b Builder
	b.Store(1, b.Binary(Add, b.Load(1), b.Constant(1)))
	b.Checkpoint(words, 1)
	b.Store(2, b.MemoryLoad(b.Load(0), 8))
	b.Checkpoint(words, 2)
	b.MemoryStore(b.Load(0), b.Load(3), 1)
	b.Checkpoint(words, 3)
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
	backing := make([]byte, 66*4096)
	for _, foreign := range []bool{false, true} {
		for _, mode := range []uint64{0, 1, 2, 0, 1} {
			for _, permission := range []uint64{3, 1, 2, 0, 3} {
				for _, malformed := range []bool{false, true} {
					for _, address := range []uint64{65 * 4096, 66*4096 - 4, 1<<48 | 65*4096} {
						for _, budget := range []uint64{1, 3, 4, 5, 64, 65536} {
							if !foreign && budget > 64 {
								continue
							}
							m.HashPages = mode
							m.Pages = [64]NativePage{}
							for i := range backing {
								backing[i] = 0
							}
							backing[65*4096] = 23
							m.Fill(65, (*[4096]byte)(backing[65*4096:]), permission, &epoch)
							if malformed {
								m.Pages[m.PageIndex(65)].Number = 129
							}
							state := []uint64{address, 41, 999, 0xe9, 0}
							want := append([]uint64(nil), state...)
							var retired, memories, status uint64
							wantByte := byte(23)
							if budget >= count {
								want[1]++
								retired++
								if address>>12 != 65 || address&4095 > 4088 || permission&1 == 0 || malformed {
									status = 1
								} else {
									want[2] = 23
									retired++
									memories++
									if permission&2 == 0 {
										status = 1
									} else {
										wantByte = 0xe9
										retired += 2
										memories++
										want[pc] = 4
									}
								}
							}
							m.PublishLink(0, entry, count, [17]uint8{})
							if foreign {
								var expired bool
								expired, err = n.RunLinkedBackingSessionUntil(state, m, budget, backing, rfenativebridge.MonotonicNS()+1000000000)
								if expired {
									t.Fatal("unexpected timer expiry")
								}
							} else {
								err = n.CallLinked(state, m, budget)
							}
							if err != nil || !reflect.DeepEqual(state, want) || m.Total != retired || m.MemoryTotal != memories || m.Remaining != budget-retired || m.Status != status || status != 0 && m.Address != address || backing[65*4096] != wantByte {
								t.Fatalf("foreign=%v mode=%d permission=%d malformed=%v address=%x budget=%d err=%v state=%x want=%x total=%d/%d memories=%d/%d status=%d/%d byte=%d/%d", foreign, mode, permission, malformed, address, budget, err, state, want, m.Total, retired, m.MemoryTotal, memories, m.Status, status, backing[65*4096], wantByte)
							}
						}
					}
				}
			}
		}
	}
}
