//go:build !renvo

package runtime

import "testing"

func TestNativeConstantVariableShiftBoundaries(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	var b Builder
	input := b.Load(0)
	slot := 1
	for _, width := range []int{32, 64} {
		for kind := VariableShl; kind <= RotateRight; kind++ {
			for _, count := range []uint64{0, 1, 31, 32, 63, 64, 65, ^uint64(0)} {
				b.Store(slot, b.VariableShift(kind, input, b.Constant(count), width))
				slot++
			}
		}
	}
	ops := b.Finish(slot)
	entry, err := n.Compile(ops, slot)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []uint64{0, 1, 0x80000000, 0xffffffff00000001, 1 << 63, ^uint64(0), 0x123456789abcdef0} {
		state := make([]uint64, slot)
		state[0] = value
		want := append([]uint64(nil), state...)
		if err = Interpret(ops, want); err != nil {
			t.Fatal(err)
		}
		if err = n.Call(entry, state); err != nil {
			t.Fatal(err)
		}
		for i := range state {
			if state[i] != want[i] {
				t.Fatalf("value=%x slot=%d got=%x want=%x", value, i, state[i], want[i])
			}
		}
	}
}

func TestLowWordSignExtensionPreservesOtherUses(t *testing.T) {
	for _, sign := range []uint64{1 << 7, 1 << 15, 1 << 31, 1 << 63} {
		for _, mask := range []uint64{15, 255, 65535, 0xffffffff, ^uint64(0), 0xff00, 0x8000000000000080} {
			for _, kind := range []int{Add, Sub, Mul} {
				var b Builder
				x, y := b.Load(0), b.Load(1)
				extended := b.Binary(Sub, b.Binary(Xor, x, b.Constant(sign)), b.Constant(sign))
				b.Store(2, extended) // Observable full-width use must not be rewritten.
				b.Store(3, b.Binary(And, b.Binary(kind, extended, y), b.Constant(mask)))
				b.Store(4, b.Binary(And, b.Binary(kind, y, extended), b.Constant(mask)))
				ops := b.Finish(5)
				for _, value := range []uint64{0, 1, sign - 1, sign, sign + 1, ^uint64(0), 0x123456789abcdef0} {
					state := []uint64{value, 37, 0, 0, 0}
					if err := Interpret(ops, state); err != nil {
						t.Fatal(err)
					}
					full := (value ^ sign) - sign
					left, right := full+37, full+37
					if kind == Sub {
						left, right = full-37, 37-full
					}
					if kind == Mul {
						left, right = full*37, full*37
					}
					if state[2] != full || state[3] != left&mask || state[4] != right&mask {
						t.Fatalf("kind=%d sign=%x mask=%x value=%x: %x", kind, sign, mask, value, state)
					}
				}
			}
		}
	}
}
