package runtime

import "testing"

func TestMaskedShiftExtractionPreservesAllSelectedBits(t *testing.T) {
	for _, left := range []uint64{1, 7, 28, 31, 32, 63} {
		for _, right := range []uint64{0, 1, 7, 29, 32, 63, 64, 65} {
			for _, mask := range []uint64{1, 0xf, 0x80000000, 0x5555555555555555, ^uint64(0)} {
				var b Builder
				v := b.Load(0)
				packed := b.Binary(Or, b.Shift(Shl, v, left), b.Binary(And, b.Load(1), b.Constant(0xa00000000)))
				b.Store(2, b.Binary(And, b.Shift(Shr, packed, right), b.Constant(mask)))
				b.Store(3, b.Binary(And, b.Shift(Shl, v, left), b.Constant(mask)))
				ops := b.Finish(4)
				x := uint64(0x8877665544332211)
				for n := 0; n < 16; n++ {
					x = x*6364136223846793005 + 1442695040888963407
					state := []uint64{x, ^x, 0, 0}
					if e := Interpret(ops, state); e != nil {
						t.Fatal(e)
					}
					want := ((x << left) | (^x & 0xa00000000)) >> right & mask
					if state[2] != want || state[3] != (x<<left)&mask {
						t.Fatal(left, right, mask, x, state, want)
					}
				}
			}
		}
	}
}
