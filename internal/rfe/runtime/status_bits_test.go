//go:build !renvo

package runtime

import (
	"math/bits"
	"testing"
)

func statusOracle(a, c uint64, width int, sub bool) uint64 {
	mask := ^uint64(0) >> (64 - width)
	a &= mask
	c &= mask
	sum, carry := bits.Add64(a, c, 0)
	aux := (a&15)+(c&15) > 15
	if width < 64 {
		carry = sum >> width
	}
	result := sum & mask
	if sub {
		result = (a - c) & mask
		carry = 0
		if a < c {
			carry = 1
		}
		aux = a&15 < c&15
	}
	sign := uint64(1) << (width - 1)
	// Signed addition/subtraction overflows exactly when the result changes
	// sign against equal-sign addends or opposite-sign subtrahends.
	overflow := (a&sign == c&sign) && (result&sign != a&sign)
	if sub {
		overflow = (a&sign != c&sign) && (result&sign != a&sign)
	}
	f := uint64(0)
	if carry != 0 {
		f |= 1
	}
	if bits.OnesCount8(uint8(result))%2 == 0 {
		f |= 4
	}
	if aux {
		f |= 16
	}
	if result == 0 {
		f |= 64
	}
	if result&sign != 0 {
		f |= 128
	}
	if overflow {
		f |= 2048
	}
	return f
}
func TestNativeStatusBitsAndArithmeticStatus(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	for _, width := range []int{8, 16, 32, 64} {
		for _, sub := range []bool{false, true} {
			var b Builder
			flags := b.ArithmeticStatus(b.Load(0), b.Load(1), width, sub)
			b.Store(2, flags)
			b.Store(3, b.StatusBits(b.Load(0), width))
			masks := []uint64{1, 4, 16, 64, 128, 2048, 65, 2176, 0x8d5, ^uint64(0)}
			for i, m := range masks {
				b.Store(4+i, b.Binary(And, flags, b.Constant(m)))
			}
			ops := b.Finish(4 + len(masks))
			entry, err := n.Compile(ops, 4+len(masks))
			if err != nil {
				t.Fatal(err)
			}
			check := func(a, c uint64) {
				state := make([]uint64, 4+len(masks))
				state[0], state[1] = a, c
				want := statusOracle(a, c, width, sub)
				wantStatus := statusOracle(a, 0, width, false) & 0xc4
				for tier := 0; tier < 2; tier++ {
					if tier == 0 {
						err = Interpret(ops, state)
					} else {
						err = n.Call(entry, state)
					}
					if err != nil {
						t.Fatal(err)
					}
					if state[2] != want || state[3] != wantStatus {
						t.Fatalf("tier=%d width=%d sub=%v a=%x b=%x flags=%x/%x want=%x/%x", tier, width, sub, a, c, state[2], state[3], want, wantStatus)
					}
					for i, m := range masks {
						if state[4+i] != want&m {
							t.Fatal("field extraction", width, sub, a, c, m, state[4+i], want&m)
						}
					}
				}
			}
			if width == 8 {
				for a := uint64(0); a < 256; a++ {
					for c := uint64(0); c < 256; c++ {
						check(a|0xdeadbeef12340000, c|0x123456789abc0000)
					}
				}
			} else {
				sign := uint64(1) << (width - 1)
				inputs := []uint64{0, 1, 15, 16, sign - 1, sign, sign + 1, sign | sign - 1, ^uint64(0), 0x123456789abcdef0}
				for _, a := range inputs {
					for _, c := range inputs {
						check(a, c)
					}
				}
			}
		}
	}
	for _, kind := range []int{StatusBits, ArithmeticStatus} {
		for _, width := range []uint64{0, 7, 17, 63, 65, 255} {
			if Validate([]Op{{Kind: LoadState}, {Kind: kind, A: 0, B: 0, Imm: width}}, 1) == nil {
				t.Fatal("invalid status width admitted", kind, width)
			}
		}
	}
}
