//go:build !renvo

package runtime

import (
	"encoding/binary"
	"reflect"
	"testing"
	"unsafe"

	"renvo.dev/internal/rfenativebridge"
)

// Immutable entry values span two faultable reads and a parallel final update.
// Slot 2 is restored at the end but NOT immutable at the intermediate exit.
// Aliasing the readable page with state also checks that read-only does not mean
// the architectural stores are visible early to guest memory loads.
func TestNativeReadonlyAcyclicInputsAndFaults(t *testing.T) {
	if !NativeSessionsAvailable() {
		t.Skip("foreign sessions unavailable")
	}
	const words, pc, count = 25, 24, 4
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
	values := make([]Value, 12)
	for i := range values {
		values[i] = b.Load(i)
	}
	b.Store(2, b.Binary(Add, values[2], b.Constant(100)))
	b.Checkpoint(words, 1)
	first := b.MemoryLoad(values[0], 8)
	b.Store(23, first)
	b.Store(12, b.Binary(Add, values[2], first))
	b.Checkpoint(words, 2)
	second := b.MemoryLoad(values[1], 8)
	for i := 2; i < 12; i++ {
		b.Store(i+10, b.Binary(Add, values[i], b.Binary(Add, first, second)))
	}
	b.Store(2, values[2])
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
	for _, foreign := range []bool{false, true} {
		for _, alias := range []bool{false, true} {
			for _, denied := range []int{0, 1, 2, 0} {
				for _, addresses := range [][2]uint64{{4112, 8208}, {8188, 8208}, {4112, 12284}, {1<<48 | 4112, 8208}} {
					for _, budget := range []uint64{1, 3, 4, 5, 64, 65536} {
						if !foreign && budget > 64 {
							continue
						}
						backing := make([]byte, 3*4096)
						state := make([]uint64, words)
						if alias {
							state = unsafe.Slice((*uint64)(unsafe.Pointer(&backing[4096])), words)
						}
						for i := range state {
							state[i] = 0x1122334400000000 + uint64(i*17+5)
						}
						state[0], state[1], state[pc] = addresses[0], addresses[1], 0
						if !alias {
							binary.LittleEndian.PutUint64(backing[4112:], 31)
						}
						binary.LittleEndian.PutUint64(backing[8208:], 53)
						m.Pages = [64]NativePage{}
						for page := 1; page <= 2; page++ {
							permission := uint64(1)
							if page == denied {
								permission = 0
							}
							m.Fill(uint64(page), (*[4096]byte)(backing[page*4096:]), permission, &epoch)
						}
						want := append([]uint64(nil), state...)
						var retired, memories, status, fault uint64
						if budget >= count {
							original := want[2]
							want[2] += 100
							retired = 1
							valid := func(address uint64) bool {
								return address>>12 >= 1 && address>>12 <= 2 && address&4095 <= 4088 && int(address>>12) != denied
							}
							if !valid(addresses[0]) {
								status, fault = 1, addresses[0]
							} else {
								first := binary.LittleEndian.Uint64(backing[addresses[0]:])
								want[23] = first
								want[12] = original + first
								retired, memories = 2, 1
								if !valid(addresses[1]) {
									status, fault = 1, addresses[1]
								} else {
									second := binary.LittleEndian.Uint64(backing[addresses[1]:])
									for i := 2; i < 12; i++ {
										want[i+10] = state[i] + first + second
									}
									want[2], want[pc] = original, 4
									retired, memories = count, 2
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
						if err != nil || !reflect.DeepEqual(state, want) || m.Total != retired || m.Remaining != budget-retired || m.MemoryTotal != memories || m.Status != status || status != 0 && m.Address != fault {
							t.Fatalf("foreign=%v alias=%v denied=%d addresses=%x budget=%d err=%v state=%x want=%x retired=%d/%d memories=%d/%d status=%d/%d fault=%x/%x", foreign, alias, denied, addresses, budget, err, state, want, m.Total, retired, m.MemoryTotal, memories, m.Status, status, m.Address, fault)
						}
					}
				}
			}
		}
	}
}
