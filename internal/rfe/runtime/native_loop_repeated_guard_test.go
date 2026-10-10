//go:build !renvo

package runtime

import (
	"encoding/binary"
	"reflect"
	"testing"
)

func TestNativeLoopRepeatedAddressProof(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(6, 5); err != nil {
		t.Fatal(err)
	}
	type access struct {
		other, store bool
		size, slot   int
	}
	for _, widths := range [][2]int{{8, 8}, {8, 1}, {1, 8}, {1, 1}} {
		for _, intervening := range []bool{false, true} {
			accesses := []access{{size: widths[0], slot: 2}}
			if intervening {
				accesses = append(accesses, access{other: true, size: 8, slot: 3})
			}
			accesses = append(accesses, access{store: true, size: 1}, access{size: widths[1], slot: 4}, access{store: true, size: widths[1]})
			instructions := len(accesses)
			var b Builder
			address, other := b.Load(0), b.Load(1)
			for i, op := range accesses {
				a := address
				if op.other {
					a = other
				}
				if op.store {
					b.MemoryStore(a, b.Constant(0x55), op.size)
				} else {
					b.Store(op.slot, b.MemoryLoad(a, op.size))
				}
				if i == instructions-1 {
					b.Store(0, b.Binary(Add, address, b.Constant(8)))
				}
				b.Checkpoint(6, i+1)
			}
			b.LoopContinue(b.Constant(1))
			entry, err := n.CompileLoop(b.FinishMemory(6), 6, instructions)
			if err != nil {
				t.Fatal(err)
			}
			for _, start := range []uint64{4101, 8184, 8191, 8192} {
				for _, initialClock := range []uint64{1, ^uint64(0) - 1} {
					for _, budget := range []uint64{1, 3, 4, 5, 7, 8, 9, 10, 64} {
						page, second := new([4096]byte), new([4096]byte)
						for i := range page {
							page[i], second[i] = byte(i*17+3), byte(i*31+7)
						}
						wantPage := *page
						clock, epoch, otherEpoch := initialClock, uint64(1), uint64(1)
						m := &MemoryContext{NativeContext: NativeContext{Clock: &clock}}
						m.Fill(1, page, 3, &epoch)
						m.Fill(2, second, 1, &otherEpoch)
						m.ClaimLinks(n)
						m.PublishLink(0, entry, instructions, [17]uint8{})
						state := []uint64{start, 8200, 91, 92, 93, 0}
						want := append([]uint64(nil), state...)
						retired, memories, completed, status, fault := uint64(0), uint64(0), uint64(0), uint64(0), uint64(0)
						wantClock, wantEpoch := clock, epoch
						for budget-retired >= uint64(instructions) {
							for _, op := range accesses {
								a, base, data := want[0], uint64(4096), &wantPage
								if op.other {
									a, base, data = want[1], 8192, second
								}
								// The first address belongs to page 1; page 2 is also readable,
								// but never writable. Cross-page accesses still fail atomically.
								if !op.other && a >= 8192 && a < 12288 {
									base, data = 8192, second
								}
								pos := a - base
								if pos > uint64(4096-op.size) || op.store && (base != 4096 || wantClock == ^uint64(0)) {
									status, fault = 1, a
									break
								}
								var word [8]byte
								if op.store {
									wantClock++
									wantEpoch = wantClock
									binary.LittleEndian.PutUint64(word[:], 0x55)
									copy(data[pos:pos+uint64(op.size)], word[:op.size])
								} else {
									copy(word[:], data[pos:pos+uint64(op.size)])
									want[op.slot] = binary.LittleEndian.Uint64(word[:])
								}
								retired++
								memories++
							}
							if status != 0 {
								break
							}
							want[0] += 8
							completed++
						}
						if err = n.CallLinked(state, m, budget); err != nil {
							t.Fatal(err)
						}
						if !reflect.DeepEqual(state, want) || *page != wantPage || clock != wantClock || epoch != wantEpoch || m.Total != retired || m.MemoryTotal != memories || m.LoopIterations != completed || m.Status != status || m.Remaining != budget-retired || status != 0 && m.Address != fault {
							t.Fatal("repeated proof mismatch", widths, intervening, start, initialClock, budget, state, want, m.Total, retired, m.Status, status)
						}
					}
				}
			}
		}
	}
}
