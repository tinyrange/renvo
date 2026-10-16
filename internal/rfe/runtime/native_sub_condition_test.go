//go:build !renvo

package runtime

import "testing"

// Constants straddle the signed imm8/imm32 encodings and the 32-bit truncation
// boundary. Compare independently typed orders, including live full operands,
// rather than deriving expected answers from the packed-flags implementation.
func TestNativeSubConditionsImmediateWidths(t *testing.T) {
	n, err := NewNative(4 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	for _, width := range []int{32, 64} {
		for _, rhs := range []uint64{0, 1, 127, 128, 255, 0x7fffffff, 0x80000000, 0xffffffff, 0x100000000, 0x7fffffffffffffff, 0x8000000000000000, ^uint64(0), ^uint64(128)} {
			var b Builder
			a := b.Load(0)
			flags := b.ArithmeticFlags(a, b.Constant(rhs), width, true)
			for c := 0; c < 14; c++ {
				v, ok := b.FlagCondition(flags, c)
				if !ok {
					t.Fatal("predicate")
				}
				b.Store(1+c, v)
			}
			b.Store(15, b.Binary(Xor, a, b.Constant(0x81f047b293ad65ce)))
			entry, e := n.Compile(b.Finish(16), 16)
			if e != nil {
				t.Fatal(e)
			}
			for _, input := range []uint64{0, 1, 127, 128, 0x7fffffff, 0x80000000, 0xffffffff, 0x100000000, 0x7fffffffffffffff, 0x8000000000000000, ^uint64(0)} {
				x, y := input, rhs
				if width == 32 {
					x, y = uint64(uint32(x)), uint64(uint32(y))
				}
				r := x - y
				if width == 32 {
					r = uint64(uint32(r))
				}
				neg, zero, carry := r>>uint(width-1) != 0, x == y, x >= y
				sign := uint64(1) << uint(width-1)
				overflow := ((x ^ y) & (x ^ r) & sign) != 0
				less, greater := int64(x) < int64(y), int64(x) > int64(y)
				if width == 32 {
					less, greater = int32(x) < int32(y), int32(x) > int32(y)
				}
				want := []bool{zero, !zero, carry, !carry, neg, !neg, overflow, !overflow, x > y, x <= y, !less, less, greater, !greater}
				state := make([]uint64, 16)
				state[0] = input
				if e := n.Call(entry, state); e != nil {
					t.Fatal(e)
				}
				if state[0] != input || state[15] != input^0x81f047b293ad65ce {
					t.Fatal("lost live operand", width, rhs, input, state)
				}
				for c, w := range want {
					if state[1+c] > 1 || (state[1+c] == 1) != w {
						t.Fatalf("width=%d rhs=%x input=%x condition=%d got=%d want=%t", width, rhs, input, c, state[1+c], w)
					}
				}
			}
		}
	}
}
