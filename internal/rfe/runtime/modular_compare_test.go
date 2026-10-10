package runtime

import "testing"

func TestModularDifferenceZeroComparison(t *testing.T) {
	for _, mask := range []uint64{0, 1, 255, 65535, 0xffffffff, ^uint64(0), 0x80, 0xff00} {
		var b Builder
		difference := b.Binary(Sub, b.Load(0), b.Load(1))
		b.Store(2, b.Binary(Equal, b.Binary(And, difference, b.Constant(mask)), b.Constant(0)))
		ops := b.Finish(3)
		if mask&(mask+1) == 0 {
			for _, op := range ops {
				if op.Kind == Sub {
					t.Fatal("unnecessary subtraction", mask, ops)
				}
			}
		}
		for _, a := range []uint64{0, 1, 127, 128, 255, 256, 65535, 65536, 0xffffffff, 0x100000000, 1 << 63, ^uint64(0)} {
			for _, c := range []uint64{0, 1, 128, 255, 256, 65535, 65536, 0xffffffff, 0x100000000, 1 << 63, ^uint64(0)} {
				state := []uint64{a, c, 99}
				if err := Interpret(ops, state); err != nil {
					t.Fatal(err)
				}
				want := uint64(0)
				if (a-c)&mask == 0 {
					want = 1
				}
				if state[2] != want {
					t.Fatalf("a=%x b=%x mask=%x: %v want %d", a, c, mask, state, want)
				}
			}
		}
	}
}
