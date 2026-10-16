//go:build !renvo

package runtime

import (
	"encoding/binary"
	"reflect"
	"renvo.dev/internal/rfenativebridge"
	"testing"
)

// Outputs 4 and 5 have no entry LoadState. A fault in a later iteration must
// recover the PREVIOUS iteration's first read together with unchanged inputs,
// not the current faulting value or stale pre-entry architectural outputs.
func TestNativeImmutableInputsInheritedAtEarlyFault(t *testing.T) {
	if !NativeSessionsAvailable() {
		t.Skip("foreign sessions unavailable")
	}
	const words, pc, count = 9, 8, 4
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
	address, counter, key, bits, limit := b.Load(0), b.Load(1), b.Load(2), b.Load(3), b.Load(7)
	first := b.MemoryLoad(address, 8)
	b.Store(4, b.Binary(Add, first, key))
	b.Store(5, b.Binary(Xor, first, bits))
	b.Checkpoint(words, 2)
	b.Store(6, b.MemoryLoad(b.Binary(Add, address, b.Constant(8)), 8))
	b.Store(0, b.Binary(Add, address, b.Constant(8)))
	next := b.Binary(Add, counter, b.Constant(1))
	b.Store(1, next)
	condition := b.Binary(Less, next, limit)
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
	backing := make([]byte, 8192)
	for off := 4096; off <= 8184; off += 8 {
		binary.LittleEndian.PutUint64(backing[off:], uint64(off*31+7))
	}
	for _, foreign := range []bool{false, true} {
		for _, permission := range []uint64{1, 0, 1, 1<<63 | 1} {
			m.Fill(1, (*[4096]byte)(backing[4096:]), permission, &epoch)
			for _, start := range []uint64{4096, 8160, 8184, 8192, 8188, 1<<48 | 4096} {
				for _, limit := range []uint64{1, 3, 2000} {
					for _, budget := range []uint64{1, 3, 4, 5, 8, 12, 63, 64, 65, 4096, 65536} {
						if !foreign && budget > 64 {
							continue
						}
						state := []uint64{start, 0, 0x123456789abcdef0, 0xa55a0ff001234567, 999, 777, 555, limit, 0}
						want := append([]uint64(nil), state...)
						var retired, memories, iterations, status, fault uint64
						for retired+count <= budget {
							at := want[0]
							if at < 4096 || at > 8184 || permission&1 == 0 || permission&^uint64(7) != 0 {
								status, fault = 1, at
								break
							}
							first := binary.LittleEndian.Uint64(backing[at:])
							want[4], want[5] = first+want[2], first^want[3]
							retired, memories = retired+2, memories+1
							at += 8
							if at > 8184 {
								status, fault = 1, at
								break
							}
							want[6] = binary.LittleEndian.Uint64(backing[at:])
							want[0], want[1] = at, want[1]+1
							retired, memories, iterations = retired+2, memories+1, iterations+1
							if want[1] >= limit {
								want[pc] = 4
								break
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
						if err != nil || !reflect.DeepEqual(state, want) || m.Total != retired || m.Remaining != budget-retired || m.MemoryTotal != memories || m.LoopIterations != iterations || m.Status != status || status != 0 && m.Address != fault {
							t.Fatalf("foreign=%v permission=%x start=%x limit=%d budget=%d err=%v state=%x want=%x total=%d/%d memories=%d/%d iterations=%d/%d status=%d/%d fault=%x/%x", foreign, permission, start, limit, budget, err, state, want, m.Total, retired, m.MemoryTotal, memories, m.LoopIterations, iterations, m.Status, status, m.Address, fault)
						}
					}
				}
			}
		}
	}
}
