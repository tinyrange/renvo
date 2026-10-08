//go:build !renvo

package runtime

import (
	"bytes"
	"testing"
)

func TestNativePairStoreAtomicGenerationAndBudgets(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(5, 4); err != nil {
		t.Fatal(err)
	}
	var b Builder
	address, value := b.Load(0), b.Load(1)
	second := b.Choose(b.Binary(And, value, b.Constant(1)), b.Constant(0x8877665544332211), b.Constant(0x1122334455667788))
	b.MemoryPairStore(address, value, second)
	b.Store(0, b.Binary(Add, address, b.Constant(16)))
	b.Store(1, b.Binary(Add, value, b.Constant(1)))
	b.Store(2, second)
	b.Store(4, b.Constant(0))
	b.Checkpoint(5, 1)
	b.LoopContinue(b.Constant(1))
	entry, err := n.CompileLoop(b.FinishMemory(5), 5, 1)
	if err != nil {
		t.Fatal(err)
	}
	for budget := uint64(1); budget <= 64; budget++ {
		for _, address := range []uint64{0, 4064, 4080, 4088, 4096, 1 << 48} {
			for _, permissions := range []uint64{1, 2, 3, 7} {
				for _, initialClock := range []uint64{1, ^uint64(0) - 1, ^uint64(0)} {
					page := new([4096]byte)
					wantPage := make([]byte, 4096)
					for i := range page {
						page[i] = byte(i*37 + 129)
						wantPage[i] = page[i]
					}
					clock, epoch := initialClock, initialClock
					m := new(MemoryContext)
					m.Clock = &clock
					m.Fill(0, page, permissions, &epoch)
					m.ClaimLinks(n)
					m.PublishLink(0, entry, 1, [17]uint8{})
					state := []uint64{address, 0x123456789abcde00, 97, 96, 0}
					want := append([]uint64(nil), state...)
					iterations, wantClock := uint64(0), initialClock
					for iterations < budget && want[0] <= 4080 && permissions&6 == 2 && wantClock != ^uint64(0) {
						high := uint64(0x1122334455667788)
						if want[1]&1 != 0 {
							high = 0x8877665544332211
						}
						for j := uint64(0); j < 8; j++ {
							wantPage[want[0]+j] = byte(want[1] >> (j * 8))
							wantPage[want[0]+8+j] = byte(high >> (j * 8))
						}
						want[0] += 16
						want[1]++
						want[2] = high
						iterations++
						wantClock++
					}
					if err = n.CallLinked(state, m, budget); err != nil {
						t.Fatal(err)
					}
					for i := range state {
						if state[i] != want[i] {
							t.Fatal("pair store state", address, permissions, budget, initialClock, state, want)
						}
					}
					status := uint64(0)
					if iterations < budget {
						status = 1
					}
					if !bytes.Equal(page[:], wantPage) || clock != wantClock || epoch != wantClock || m.Total != iterations || m.MemoryTotal != iterations || m.LoopIterations != iterations || m.Status != status || status == 1 && m.Address != want[0] {
						t.Fatal("pair store lost atomicity/accounting", address, permissions, budget, initialClock, clock, epoch, m.Total, m.MemoryTotal, m.Status, m.Address)
					}
				}
			}
		}
	}
}
