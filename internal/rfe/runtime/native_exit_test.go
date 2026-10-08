//go:build !renvo

package runtime

import "testing"

func TestNativeColdExitKeepsOldSpilledAddress(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	var b Builder
	address := b.Load(0)
	values := make([]Value, 24)
	for i := range values {
		values[i] = b.Binary(Mul, b.Load(i+1), b.Constant(uint64(i*2+3)))
	}
	b.Store(0, b.Constant(17)) // architectural address slot now differs from its SSA version
	b.Checkpoint(26, 1)
	loaded := b.MemoryLoad(address, 8)
	sum := loaded
	for _, v := range values {
		sum = b.Binary(Add, sum, v)
	}
	b.Store(24, sum)
	b.Store(25, address)
	b.Checkpoint(26, 2)
	entry, err := n.CompileMemory(b.FinishMemory(26), 26)
	if err != nil {
		t.Fatal(err)
	}
	for _, fault := range []bool{false, true} {
		state := make([]uint64, 26)
		state[0] = 4096
		if fault {
			state[0] = 8192
		}
		original := state[0]
		var want uint64
		for i := range values {
			state[i+1] = uint64(i + 101)
			want += state[i+1] * uint64(i*2+3)
		}
		var page [4096]byte
		page[0] = 7
		var clock, epoch uint64 = 1, 1
		context := &MemoryContext{Clock: &clock, Retired: 99, Status: 99}
		context.Fill(1, &page, 1, &epoch)
		if err = n.CallMemory(entry, state, context); err != nil {
			t.Fatal(err)
		}
		if state[0] != 17 {
			t.Fatal("lost committed prefix")
		}
		if fault {
			if context.Retired != 1 || context.Status != 1 || context.Address != original || state[25] != 0 {
				t.Fatal("lost old address or replayed prefix", state, context)
			}
		} else if context.Retired != 2 || context.Status != 0 || state[24] != want+7 || state[25] != original {
			t.Fatal("successful exit progress mismatch", state, context)
		}
	}
}
