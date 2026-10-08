//go:build !renvo

package runtime

import "testing"

func TestNativeRegionExitProgressAndAdmission(t *testing.T) {
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
	b.Store(1, b.Constant(4))
	b.Checkpoint(2, 1)
	b.RegionGuard(b.Constant(0))
	b.MemoryStore(b.Constant(8192), b.Constant(99), 1)
	b.Checkpoint(2, 2)
	ops := b.FinishMemory(2)
	if Validate(ops, 2) == nil {
		t.Fatal("region guard admitted by pure IR")
	}
	if _, err := n.Compile(ops, 2); err == nil {
		t.Fatal("region guard admitted by pure compiler")
	}
	entry, err := n.CompileMemory(ops, 2)
	if err != nil {
		t.Fatal(err)
	}
	m := new(MemoryContext)
	m.ClaimLinks(n)
	m.PublishLink(0, entry, 2, [17]uint8{0, 0, 1})
	state := []uint64{41, 0}
	if err = n.CallLinked(state, m, 64); err != nil {
		t.Fatal(err)
	}
	if state[0] != 42 || state[1] != 4 || m.Total != 1 || m.MemoryTotal != 0 || m.Remaining != 63 || m.Status != RegionExit || m.Retired != 1 || m.CodeView != [4]uint64{} {
		t.Fatal("region side exit replayed or lost progress", state, m)
	}
	// Side exits at or beyond the declared instruction count remain invalid.
	for _, completed := range []int{2, 3, 256} {
		var bad Builder
		bad.Checkpoint(2, completed)
		bad.RegionGuard(bad.Constant(0))
		entry, err := n.CompileMemory(bad.FinishMemory(2), 2)
		if err != nil {
			t.Fatal(err)
		}
		m.PublishLink(0, entry, 2, [17]uint8{})
		state[1] = 0
		if n.CallLinked(state, m, 64) == nil || m.Status != 3 || m.Total != 0 {
			t.Fatal("invalid region progress accepted", completed, m)
		}
	}
}

func TestNativeSelectConstantEquality(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	for _, reversed := range []bool{false, true} {
		var b Builder
		choice := b.Choose(b.Load(0), b.Constant(4096), b.Constant(4100))
		for i, c := range []uint64{4096, 4100, 4104} {
			left, right := choice, b.Constant(c)
			if reversed {
				left, right = right, left
			}
			b.Store(i+1, b.Binary(Equal, left, right))
		}
		ops := b.Finish(4)
		entry, err := n.Compile(ops, 4)
		if err != nil {
			t.Fatal(err)
		}
		prepared, err := Prepare(ops, 4)
		if err != nil {
			t.Fatal(err)
		}
		for _, condition := range []uint64{0, 1, 2, 1 << 63, ^uint64(0)} {
			state := []uint64{condition, 9, 9, 9}
			native := append([]uint64(nil), state...)
			var machine IRMachine
			if err = machine.Run(prepared, state); err != nil {
				t.Fatal(err)
			}
			if err = n.Call(entry, native); err != nil {
				t.Fatal(err)
			}
			want := []uint64{condition, 0, 1, 0}
			if condition != 0 {
				want[1], want[2] = 1, 0
			}
			for i := range want {
				if state[i] != want[i] || native[i] != want[i] {
					t.Fatal("select equality changed truthiness", condition, state, native, want)
				}
			}
		}
	}
}
