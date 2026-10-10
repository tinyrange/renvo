package runtime

import "testing"

func TestBitfieldExtractEliminatesUnobservedWork(t *testing.T) {
	var b Builder
	a, rhs, old := b.Load(0), b.Load(1), b.Load(2)
	overflow := b.Binary(Less, a, rhs)
	flagProduct := b.Binary(Mul, overflow, b.Constant(0x801))
	z := b.Binary(Equal, a, b.Constant(0))
	packed := b.Binary(Or, b.Binary(And, old, b.Constant(^uint64(0x8d5))), b.Binary(Or, flagProduct, b.Shift(Shl, z, 6)))
	extracted := b.Binary(And, b.Shift(Shr, packed, 6), b.Constant(1))
	b.Store(3, extracted)
	b.Store(4, b.Binary(And, flagProduct, b.Constant(^uint64(0x801))))
	ops := b.Finish(5)
	for _, o := range ops {
		if o.Kind == Mul || o.Kind == Less || o.Kind == Shl || o.Kind == Shr {
			t.Fatalf("dead flag computation remains: %+v", ops)
		}
	}
	for _, x := range []uint64{0, 1, 1 << 63, ^uint64(0)} {
		for _, y := range []uint64{0, 1, ^uint64(0)} {
			state := []uint64{x, y, ^uint64(0), 99, 99}
			if err := Interpret(ops, state); err != nil {
				t.Fatal(err)
			}
			want := uint64(0)
			if x == 0 {
				want = 1
			}
			if state[3] != want || state[4] != 0 {
				t.Fatal(state)
			}
		}
	}
}
func TestBitfieldExtractPreservesShiftedOffAndSelectedBits(t *testing.T) {
	for _, n := range []uint64{0, 1, 6, 31, 63, 64, 65} {
		for _, mask := range []uint64{1, 255, 1 << 63, ^uint64(0)} {
			var b Builder
			x := b.Load(0)
			y := b.Load(1)
			packed := b.Binary(Or, b.Shift(Shl, x, n), b.Binary(And, y, b.Constant(0x8888888888888888)))
			b.Store(2, b.Binary(And, b.Shift(Shr, packed, n), b.Constant(mask)))
			b.Store(3, b.Shift(Shr, b.Shift(Shl, x, n), n))
			ops := b.Finish(4)
			for _, value := range []uint64{0, 1, 7, 0xabcdef0123456789, ^uint64(0)} {
				other := ^value
				state := []uint64{value, other, 0, 0}
				if err := Interpret(ops, state); err != nil {
					t.Fatal(err)
				}
				want := ((value << n) | (other & 0x8888888888888888)) >> n & mask
				if state[2] != want || state[3] != (value<<n)>>n {
					t.Fatalf("n=%d mask=%x value=%x: %x want %x", n, mask, value, state, want)
				}
			}
		}
	}
}
