//go:build !renvo

package runtime

import (
	"reflect"
	"testing"
)

func TestNativeLoopRangeCacheInitialAndWrappedAddresses(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(3, 2); err != nil {
		t.Fatal(err)
	}
	for _, width := range []int{1, 2, 4, 8} {
		for _, store := range []bool{false, true} {
			var b Builder
			address := b.Load(0)
			if store {
				b.MemoryStore(address, b.Load(1), width)
			} else {
				b.Store(1, b.MemoryLoad(address, width))
			}
			b.Store(0, b.Binary(Add, address, b.Constant(1)))
			b.Checkpoint(3, 1)
			b.LoopContinue(b.Constant(1))
			entry, err := n.CompileLoop(b.FinishMemory(3), 3, 1)
			if err != nil {
				t.Fatal(err)
			}
			for _, address := range []uint64{0, 1, 4094, ^uint64(0), ^uint64(0) - 4095, uint64(1) << 48} {
				page := new([4096]byte)
				epoch, clock := uint64(1), uint64(1)
				m := &MemoryContext{NativeContext: NativeContext{Clock: &clock}}
				// Even a forged matching public page must not admit high address bits.
				if address >= uint64(1)<<48 {
					m.Pages[(address>>12)&63] = NativePage{Number: address >> 12, Data: page, Permissions: 3, Epoch: &epoch}
				}
				m.ClaimLinks(n)
				m.PublishLink(0, entry, 1, [17]uint8{})
				state := []uint64{address, 0x1234, 0}
				before := append([]uint64(nil), state...)
				if err = n.CallLinked(state, m, 64); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(state, before) || m.Total != 0 || m.MemoryTotal != 0 || m.Status != 1 || m.Address != address || clock != 1 || epoch != 1 || *page != [4096]byte{} {
					t.Fatal("invalid initial fact admitted an access", width, store, address, state, m.Total, m.Status, m.Address)
				}
			}
		}
	}
}

func TestNativeLoopGuardFactDoesNotEscapeItsPreciseExit(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(6, 5); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []bool{false, true} {
		var b Builder
		next := b.Binary(Sub, b.Load(0), b.Constant(1))
		condition := next
		if !raw {
			condition = b.Binary(Xor, b.Binary(Equal, next, b.Constant(0)), b.Constant(1))
		}
		b.Store(0, next)
		b.Store(1, condition)
		b.Store(2, b.ArithmeticStatus(next, b.Constant(1), 8, true))
		b.Store(5, b.Choose(condition, b.Constant(0), b.Constant(4)))
		b.Checkpoint(6, 1)
		b.Store(3, b.MemoryLoad(b.Load(4), 1))
		b.Checkpoint(6, 2)
		b.RegionGuard(condition)
		b.Checkpoint(6, 3)
		b.LoopContinue(b.Constant(1))
		entry, err := n.CompileLoop(b.FinishMemory(6), 6, 3)
		if err != nil {
			t.Fatal(err)
		}
		for _, initial := range []uint64{1, 3, 100} {
			for _, fault := range []bool{false, true} {
				for _, budget := range []uint64{2, 3, 4, 8, 9, 64} {
					m := new(MemoryContext)
					epoch := uint64(1)
					page := new([4096]byte)
					page[0] = 42
					if !fault {
						m.Fill(1, page, 1, &epoch)
					}
					m.ClaimLinks(n)
					m.PublishLink(0, entry, 3, [17]uint8{})
					state := []uint64{initial, 99, 98, 97, 4096, 0}
					if err = n.CallLinked(state, m, budget); err != nil {
						t.Fatal(err)
					}
					want := []uint64{initial, 99, 98, 97, 4096, 0}
					retired, memories, status, completed := uint64(0), uint64(0), uint64(0), uint64(0)
					for budget-retired >= 3 {
						want[0]--
						predicate := want[0]
						if !raw && predicate != 0 {
							predicate = 1
						}
						want[1] = predicate
						want[2] = statusOracle(want[0], 1, 8, true)
						if predicate == 0 {
							want[5] = 4
						}
						retired++
						if fault {
							status = 1
							break
						}
						want[3] = 42
						retired++
						memories++
						if predicate == 0 {
							break
						}
						retired++
						completed++
					}
					if !reflect.DeepEqual(state, want) || m.Total != retired || m.MemoryTotal != memories || m.Status != status || m.LoopIterations != completed || m.Remaining != budget-retired {
						t.Fatal("guard or fault reconstruction", raw, initial, fault, budget, state, want, m.Total, retired, m.Status, status, m.LoopIterations, completed)
					}
				}
			}
		}
	}
}
