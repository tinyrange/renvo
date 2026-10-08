//go:build !renvo

package runtime

import "testing"

func TestNativeSelectPressureAndTruthiness(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	var b Builder
	b.Binary(Add, b.Load(0), b.Constant(99)) // dead values force third-operand remapping
	values := make([]Value, 24)
	for i := range values {
		values[i] = b.Binary(Xor, b.Load(i+1), b.Constant(uint64(i+37)))
	}
	sum := b.Constant(0)
	for i := range values {
		sum = b.Binary(Add, sum, b.Choose(b.Load(0), values[i], values[23-i]))
	}
	b.Store(25, sum)
	ops := b.Finish(26)
	entry, err := n.Compile(ops, 26)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := Prepare(ops, 26)
	if err != nil {
		t.Fatal(err)
	}
	for _, cond := range []uint64{0, 1, 2, 1 << 63, ^uint64(0)} {
		state := make([]uint64, 26)
		state[0] = cond
		var want uint64
		for i := 0; i < 24; i++ {
			state[i+1] = uint64(i*7919) ^ 0xfedcba9876543210
		}
		for i := 0; i < 24; i++ {
			j := i
			if cond == 0 {
				j = 23 - i
			}
			want += state[j+1] ^ uint64(j+37)
		}
		native := append([]uint64(nil), state...)
		if err = n.Call(entry, native); err != nil {
			t.Fatal(err)
		}
		var machine IRMachine
		if err = machine.Run(prepared, state); err != nil {
			t.Fatal(err)
		}
		if native[25] != want || state[25] != want {
			t.Fatalf("condition %x: native %x IR %x want %x", cond, native[25], state[25], want)
		}
	}
	// Direct clients need not normalize the condition to 0/1.
	direct := []Op{{Kind: LoadState, Imm: 0}, {Kind: LoadState, Imm: 1}, {Kind: LoadState, Imm: 2}, {Kind: SelectValue, A: 0, B: 1, Imm: 2}, {Kind: StoreState, A: 3, Imm: 3}}
	entry, err = n.Compile(direct, 4)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []uint64{0, 2, 1 << 63, ^uint64(0)} {
		state := []uint64{c, 0xdeadbeef01234567, 0xfedcba9876543210, 0}
		want := state[2]
		if c != 0 {
			want = state[1]
		}
		if err = n.Call(entry, state); err != nil {
			t.Fatal(err)
		}
		if state[3] != want {
			t.Fatal(state, want)
		}
	}
	for _, bad := range []uint64{3, 4, ^uint64(0)} {
		direct[3].Imm = bad
		if Validate(direct, 4) == nil || ValidateMemory(direct, 4) == nil {
			t.Fatal("accepted invalid false operand", bad)
		}
	}
	direct[3].Imm = 2
	direct[2] = Op{Kind: StoreState, A: 1, Imm: 2}
	if Validate(direct, 4) == nil || ValidateMemory(direct, 4) == nil {
		t.Fatal("accepted effect-only false operand")
	}
}

func TestNativeImmediateSigned32Boundaries(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	for _, k := range []int{Add, Sub, And, Or, Xor, Mul} {
		for _, c := range []uint64{0x7fffffff, 0x80000000, 0xffffffff, 0xffffffff80000000, 0xffffffff7fffffff, ^uint64(0)} {
			var b Builder
			b.Store(1, b.Binary(k, b.Load(0), b.Constant(c)))
			entry, err := n.Compile(b.Finish(2), 2)
			if err != nil {
				t.Fatal(err)
			}
			for _, x := range []uint64{0, 1, 0x123456789abcdef0, ^uint64(0)} {
				var want uint64
				switch k {
				case Add:
					want = x + c
				case Sub:
					want = x - c
				case And:
					want = x & c
				case Or:
					want = x | c
				case Xor:
					want = x ^ c
				case Mul:
					want = x * c
				}
				state := []uint64{x, 0}
				if err = n.Call(entry, state); err != nil {
					t.Fatal(err)
				}
				if state[1] != want {
					t.Fatalf("op %d x %x c %x got %x want %x", k, x, c, state[1], want)
				}
			}
		}
	}
}

func TestNativeSelectDoesNotSuppressMemoryFault(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	for _, cond := range []uint64{0, 1} {
		var b Builder
		b.Binary(Add, b.Load(0), b.Constant(99))
		b.Checkpoint(2, 0)
		loaded := b.MemoryLoad(b.Constant(8192), 8)
		b.Store(0, b.Choose(b.Load(1), b.Constant(7), loaded))
		b.Checkpoint(2, 1)
		ops := b.FinishMemory(2)
		entry, err := n.CompileMemory(ops, 2)
		if err != nil {
			t.Fatal(err)
		}
		state := []uint64{42, cond}
		m := &MemoryContext{}
		if err = n.CallMemory(entry, state, m); err != nil {
			t.Fatal(err)
		}
		if m.Status != 1 || m.Address != 8192 || m.Retired != 0 || state[0] != 42 {
			t.Fatal("select suppressed or reordered a fault", state, m)
		}
	}
}
