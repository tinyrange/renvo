//go:build !renvo

package runtime

import "testing"

func TestBuilderSlotCheckpointsPreserveFaultPrefix(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Skip(err)
	}
	defer n.Close()
	var b Builder
	slots := []int{0, 31, 32, 63, 64, 127, 128, 255}
	for round := 0; round < 4; round++ {
		b.Reset(true)
		for _, slot := range slots {
			b.Store(slot, b.Binary(Add, b.Load(slot), b.Constant(uint64(round+1))))
		}
		b.Checkpoint(256, 1)
		b.Checkpoint(256, 2)
		// These writes are pending beyond the guard. Only the completed
		// prefix is architectural state when the guard exits.
		for _, slot := range slots {
			b.Store(slot, b.Constant(999))
		}
		b.Guard(b.Constant(0))
		b.Checkpoint(256, 3)
		entry, err := n.CompileMemory(b.FinishMemory(256), 256)
		if err != nil {
			t.Fatal(err)
		}
		state := make([]uint64, 256)
		for i := range state {
			state[i] = uint64(i * 17)
		}
		context := &MemoryContext{}
		if err := n.CallMemory(entry, state, context); err != nil {
			t.Fatal(err)
		}
		if context.Status != 2 || context.Retired != 2 {
			t.Fatal("lost precise checkpoint", context.Status, context.Retired)
		}
		for i, value := range state {
			want := uint64(i * 17)
			for _, slot := range slots {
				if i == slot {
					want += uint64(round + 1)
				}
			}
			if value != want {
				t.Fatalf("round=%d slot=%d got=%d want=%d", round, i, value, want)
			}
		}
	}
}

func TestBuilderSparseSlotsRemainValidatedByIR(t *testing.T) {
	var b Builder
	negative := b.Load(-1)
	if b.Load(-1) != negative {
		t.Fatal("sparse load forwarding lost")
	}
	b.Store(256, b.Constant(42))
	b.Commit(257)
	ops := b.FinishMemory(257)
	if ValidateMemory(ops, 257) == nil {
		t.Fatal("oversized state became valid")
	}
	found := false
	for _, op := range ops {
		if op.Kind == StoreState && op.Imm == 256 {
			found = true
		}
	}
	if !found {
		t.Fatal("sparse state disappeared before validation")
	}
	b.Reset(false)
	if _, ok := b.stateValue(256); ok {
		t.Fatal("sparse state survived reset")
	}
}
