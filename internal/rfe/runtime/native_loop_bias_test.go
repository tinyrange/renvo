//go:build !renvo

package runtime

import (
	"encoding/binary"
	"reflect"
	"testing"
	"unsafe"
)

func TestNativeLoopZeroBiasAndPeeledStoreOverflow(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(3, 2); err != nil {
		t.Fatal(err)
	}
	// A real, aligned backing whose guest base equals its host base exercises
	// a valid zero translation bias, including the steady peeled iteration.
	backing := new([8192]byte)
	at := uintptr(unsafe.Pointer(&backing[0]))
	offset := int((-at) & 4095)
	page := (*[4096]byte)(unsafe.Pointer(&backing[offset]))
	base := uint64(uintptr(unsafe.Pointer(page)))
	if base >= uint64(1)<<48 {
		t.Skip("host backing is outside the guest address range")
	}
	for _, size := range []int{1, 2, 4, 8} {
		var b Builder
		address := b.Load(0)
		loaded := b.MemoryLoad(address, size)
		b.Store(1, loaded)
		b.Checkpoint(3, 1)
		b.MemoryStore(address, b.Binary(Xor, loaded, b.Constant(0x55)), size)
		b.Store(0, b.Binary(Add, address, b.Constant(uint64(size))))
		b.Checkpoint(3, 2)
		b.LoopContinue(b.Constant(1))
		entry, err := n.CompileLoop(b.FinishMemory(3), 3, 2)
		if err != nil {
			t.Fatal(err)
		}
		for _, start := range []int{0, 4096 - size} {
			for _, initialClock := range []uint64{1, ^uint64(0) - 1} {
				for _, budget := range []uint64{1, 2, 3, 4, 7, 64} {
					for i := range page {
						page[i] = byte(i*17 + 0x80)
					}
					wantPage := *page
					clock, epoch := initialClock, uint64(1)
					m := &MemoryContext{NativeContext: NativeContext{Clock: &clock}}
					m.Fill(base>>12, page, 3, &epoch)
					m.ClaimLinks(n)
					m.PublishLink(0, entry, 2, [17]uint8{})
					state := []uint64{base + uint64(start), 99, 0}
					want := append([]uint64(nil), state...)
					retired, memories, completed, status := uint64(0), uint64(0), uint64(0), uint64(0)
					wantClock, wantEpoch, faultAddress := clock, epoch, uint64(0)
					for budget-retired >= 2 {
						pos := want[0] - base
						if pos > uint64(4096-size) {
							status, faultAddress = 1, want[0]
							break
						}
						var word [8]byte
						copy(word[:], wantPage[pos:pos+uint64(size)])
						want[1] = binary.LittleEndian.Uint64(word[:])
						retired++
						memories++
						if wantClock == ^uint64(0) {
							status, faultAddress = 1, want[0]
							break
						}
						wantClock++
						wantEpoch = wantClock
						binary.LittleEndian.PutUint64(word[:], want[1]^0x55)
						copy(wantPage[pos:pos+uint64(size)], word[:size])
						want[0] += uint64(size)
						retired++
						memories++
						completed++
					}
					if err = n.CallLinked(state, m, budget); err != nil {
						t.Fatal(err)
					}
					if !reflect.DeepEqual(state, want) || *page != wantPage || clock != wantClock || epoch != wantEpoch || m.Total != retired || m.MemoryTotal != memories || m.LoopIterations != completed || m.Status != status || m.Remaining != budget-retired || status != 0 && m.Address != faultAddress {
						t.Fatal("zero-bias/peeled progress mismatch", size, start, initialClock, budget, state, want, m.Total, retired, m.Status, status, clock, wantClock)
					}
				}
			}
		}
	}
}
