//go:build !renvo

package runtime

import "testing"

func TestNativeQuantaSeparateBudgetsAndReadyGeneration(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(2, 1); err != nil {
		t.Fatal(err)
	}
	var b Builder
	b.Store(0, b.Binary(Add, b.Load(0), b.Constant(1)))
	b.Store(1, b.Constant(0))
	b.Checkpoint(2, 1)
	b.LoopContinue(b.Constant(1))
	entry, err := n.CompileLoop(b.FinishMemory(2), 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	m := new(MemoryContext)
	m.ClaimLinks(n)
	m.PublishLink(0, entry, 1, [17]uint8{})
	for _, trial := range []struct {
		limit                              int
		remaining, generation, ready, want uint64
		calls                              int
	}{
		{16, 257, 7, 7, 257, 5},
		{16, 10000, 7, 7, 1024, 16},
		{1, 10000, 7, 7, 64, 1},
		{16, 257, 7, 6, 64, 1},
		{16, 257, 7, ^uint64(0), 257, 5},
	} {
		m.Blocks[0].Reserved = trial.ready
		state := []uint64{0, 0}
		calls, err := n.RunLinkedQuanta(state, m, trial.limit, trial.remaining, trial.generation)
		if err != nil || calls != trial.calls || state[0] != trial.want || state[1] != 0 || m.Total != trial.want || m.Remaining != trial.remaining-trial.want || m.LoopIterations != trial.want || m.MemoryTotal != 0 || m.CodeView != [4]uint64{} {
			t.Fatal("separate quantum/accounting/cleanup", trial, calls, err, state, m.Total, m.Remaining, m.LoopIterations)
		}
	}
	for _, bad := range []struct {
		limit     int
		remaining uint64
	}{{0, 64}, {17, 64}, {16, 0}} {
		state := []uint64{9, 0}
		if calls, err := n.RunLinkedQuanta(state, m, bad.limit, bad.remaining, 7); err == nil || calls != 0 || state[0] != 9 {
			t.Fatal("invalid schedule executed", calls, err, state)
		}
	}
	if err = n.Close(); err != nil {
		t.Fatal(err)
	}
	state := []uint64{9, 0}
	if calls, err := n.RunLinkedQuanta(state, m, 16, 257, 7); err == nil || calls != 0 || state[0] != 9 || m.CodeView != [4]uint64{} {
		t.Fatal("closed schedule", calls, err, state)
	}
}

func TestNativeQuantaFaultAfterCompletedQuantum(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(3, 2); err != nil {
		t.Fatal(err)
	}
	var b Builder
	address := b.Load(1)
	b.Store(0, b.Binary(Add, b.Load(0), b.Constant(1)))
	b.Checkpoint(3, 1)
	b.MemoryStore(address, b.Load(0), 8)
	b.Store(1, b.Binary(Add, address, b.Constant(8)))
	b.Checkpoint(3, 2)
	b.LoopContinue(b.Constant(1))
	entry, err := n.CompileLoop(b.FinishMemory(3), 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	clock, epoch := uint64(1), uint64(1)
	page := new([4096]byte)
	m := &MemoryContext{NativeContext: NativeContext{Clock: &clock}}
	m.Fill(1, page, 3, &epoch)
	m.ClaimLinks(n)
	m.PublishLink(0, entry, 2, [17]uint8{})
	m.Blocks[0].Reserved = ^uint64(0)
	state := []uint64{0, 8192 - 40*8, 0}
	calls, err := n.RunLinkedQuanta(state, m, 16, 1000, 9)
	if err != nil || calls != 2 || state[0] != 41 || state[1] != 8192 || m.Total != 81 || m.Remaining != 919 || m.MemoryTotal != 40 || m.LoopIterations != 40 || m.Status != 1 || m.Address != 8192 || clock != 41 || epoch != 41 || page[4088] != 40 || m.CodeView != [4]uint64{} {
		t.Fatal("fault crossed boundary/replayed prefix", calls, err, state, m.Total, m.MemoryTotal, m.LoopIterations, m.Status, m.Address, clock)
	}
}
