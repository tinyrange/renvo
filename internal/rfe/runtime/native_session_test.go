//go:build !renvo

package runtime

import (
	"runtime"
	"sync"
	"testing"
)

// Region length, scheduling slice, and fault-prefix retirement are independent.
// Expected state is scalar arithmetic, not a second copy of native IR/exit maps.
func TestNativeSessionWideRegionScalar(t *testing.T) {
	if !NativeSessionsAvailable() {
		t.Skip("supported foreign boundary unavailable")
	}
	for _, count := range []int{16, 17, 32, 64, 65, 128, 256} {
		for _, fault := range []bool{false, true} {
			n, err := NewNative(1 << 20)
			if err != nil {
				t.Fatal(err)
			}
			if err = n.PrepareLinks(3, 2); err != nil {
				t.Fatal(err)
			}
			var b Builder
			for instruction := 1; instruction < count; instruction++ {
				b.Store(0, b.Binary(Add, b.Load(0), b.Constant(1)))
				b.Store(2, b.Constant(uint64(instruction*4)))
				b.Checkpoint(3, instruction)
			}
			b.Store(1, b.MemoryLoad(b.Constant(4096), 8))
			b.Store(0, b.Binary(Add, b.Load(0), b.Constant(1)))
			b.Store(2, b.Constant(0))
			b.Checkpoint(3, count)
			b.LoopContinue(b.Constant(1))
			entry, err := n.CompileLoop(b.FinishMemory(3), 3, count)
			if err != nil {
				t.Fatal(count, err)
			}
			if err = n.PrepareTargetLink(0, entry, 3, count); err != nil {
				t.Fatal(err)
			}
			budgets := []uint64{65535, 65536}
			for budget := uint64(1); budget <= uint64(2*count+1); budget++ {
				budgets = append(budgets, budget)
			}
			for _, budget := range budgets {
				state := []uint64{17, 99, 0}
				m := new(MemoryContext)
				page, epoch := new([4096]byte), uint64(1)
				page[0] = 37
				if !fault {
					m.Fill(1, page, 1, &epoch)
				}
				m.ClaimLinks(n)
				m.PublishLink(0, entry, count, [17]uint8{})
				if err = n.RunLinkedSession(state, m, budget); err != nil {
					t.Fatal(count, fault, budget, err)
				}
				retired, status, memories, pc, value := budget/uint64(count)*uint64(count), uint64(0), budget/uint64(count), uint64(0), uint64(99)
				address := uint64(0)
				if fault && budget >= uint64(count) {
					retired, status, memories, pc, address = uint64(count-1), 1, 0, uint64((count-1)*4), 4096
				}
				if !fault && retired != 0 {
					value = 37
				}
				if state[0] != 17+retired || state[1] != value || state[2] != pc || m.Total != retired || m.Remaining != budget-retired || m.Status != status || m.Address != address || m.MemoryTotal != memories || m.CodeView != [4]uint64{} || m.PreparedTargets != 0 {
					t.Fatalf("count=%d fault=%v budget=%d state=%v retired=%d/%d status=%d/%d memory=%d/%d", count, fault, budget, state, m.Total, retired, m.Status, status, m.MemoryTotal, memories)
				}
			}
			if err = n.Close(); err != nil {
				t.Fatal(err)
			}
		}
	}
}

// GC must make progress while another goroutine repeatedly owns a supported
// foreign native session. Pages, epoch/clock pointers, state, proofs and private
// stack stay pinned; independent arena users are serialized by the same lock.
func TestNativeSessionGCAndScheduling(t *testing.T) {
	if !NativeSessionsAvailable() {
		t.Skip("supported foreign boundary unavailable")
	}
	previous := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(previous)
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(3, 2); err != nil {
		t.Fatal(err)
	}
	var b Builder
	address := b.Constant(4096)
	b.MemoryStore(address, b.Load(0), 8)
	b.Store(1, b.MemoryLoad(address, 8))
	b.Store(0, b.Binary(Add, b.Load(0), b.Constant(1)))
	b.Store(2, b.Constant(0))
	b.Checkpoint(3, 2)
	b.LoopContinue(b.Constant(1))
	entry, err := n.CompileLoop(b.FinishMemory(3), 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err = n.PrepareTargetLink(0, entry, 3, 2); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	stop := make(chan struct{})
	gcDone := make(chan struct{})
	go func() {
		defer close(gcDone)
		for {
			select {
			case <-stop:
				return
			default:
				runtime.GC()
			}
		}
	}()
	for worker := 0; worker < 4; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			state := []uint64{1, 0, 0}
			page, clock, epoch := new([4096]byte), uint64(1), uint64(1)
			m := &MemoryContext{Clock: &clock}
			m.Fill(1, page, 3, &epoch)
			m.ClaimLinks(n)
			m.PublishLink(0, entry, 2, [17]uint8{0, 1, 2})
			for call := 0; call < 64; call++ {
				if err := n.RunLinkedSession(state, m, 65536); err != nil {
					t.Error(err)
					return
				}
			}
			if state[0] != 1+64*32768 || state[1] != 64*32768 || clock != 1+64*32768 || epoch != clock || m.CodeView != [4]uint64{} || m.PreparedTargets != 0 {
				t.Error("pinned execution effects", state, clock, epoch)
			}
		}()
	}
	wg.Wait()
	close(stop)
	<-gcDone
}
