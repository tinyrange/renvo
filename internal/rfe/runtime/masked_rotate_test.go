package runtime

import (
	"math/bits"
	"testing"
)

func TestMaskedRotateKeepsSelectedWrapAndWidth(t *testing.T) {
	for _, width := range []int{32, 64} {
		for _, count := range []uint64{0, 1, 2, 31, 32, 33, 63, 64, 65, 127, ^uint64(0)} {
			for _, mask := range []uint64{1, 255, 0xffffffff, 0x1fffffffe, 1 << 63, 0x8888888888888888, ^uint64(0)} {
				var b Builder
				rotated := b.VariableShift(RotateRight, b.Load(0), b.Constant(count), width)
				b.Store(1, b.Binary(And, rotated, b.Constant(mask)))
				// A second full-width observer must still see wrapped bits.
				b.Store(2, rotated)
				ops := b.Finish(3)
				for _, x := range []uint64{0, 1, 1 << 31, 1 << 63, 0x123456789abcdef0, ^uint64(0)} {
					want := bits.RotateLeft64(x, -int(count&63))
					if width == 32 {
						want = uint64(bits.RotateLeft32(uint32(x), -int(count&31)))
					}
					state := []uint64{x, 99, 99}
					if err := Interpret(ops, state); err != nil {
						t.Fatal(err)
					}
					if state[1] != want&mask || state[2] != want {
						t.Fatalf("width=%d count=%d mask=%x x=%x state=%x want=%x", width, count, mask, x, state, want)
					}
				}
			}
		}
	}
	var b Builder
	v := b.Binary(And, b.VariableShift(RotateRight, b.Load(0), b.Constant(63), 64), b.Constant(0x1fffffffe))
	o := b.Ops[v]
	if o.Kind != Shl || o.Imm != 1 || b.Ops[o.A].Kind != And || b.Ops[b.Ops[o.A].B].Imm != 0xffffffff {
		t.Fatal("low-word scaled shape remained hidden", b.Ops)
	}
}

func TestMaskedSignExtensionKeepsInteriorFields(t *testing.T) {
	for _, width := range []uint{8, 16, 32, 64} {
		for _, mask := range []uint64{1, 0x3c, 0x55555555, 0xffffffff, 1 << 63, ^uint64(0)} {
			var b Builder
			sign := b.Constant(uint64(1) << (width - 1))
			low := b.Binary(And, b.Load(0), b.Constant(^uint64(0)>>(64-width)))
			extended := b.Binary(Sub, b.Binary(Xor, low, sign), sign)
			b.Store(1, b.Binary(And, extended, b.Constant(mask)))
			b.Store(2, extended)
			ops := b.Finish(3)
			for _, x := range []uint64{0, 1, 127, 128, 65535, 0x80000000, 1 << 63, 0x987654321abcdef0, ^uint64(0)} {
				want := uint64(int64(x<<(64-width)) >> (64 - width))
				state := []uint64{x, 99, 99}
				if e := Interpret(ops, state); e != nil {
					t.Fatal(e)
				}
				if state[1] != want&mask || state[2] != want {
					t.Fatalf("width=%d mask=%x x=%x got=%x want=%x", width, mask, x, state, want)
				}
			}
		}
	}
}
