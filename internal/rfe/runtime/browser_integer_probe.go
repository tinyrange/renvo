//go:build renvo && browser

package runtime

import "fmt"

// Exercise emitted instructions with runtime operands, never constant-folded
// answers. Boundaries include zero divisors, signed overflow and carry chains.
func browserIntegerProbe(n *Native) error {
	ops := []Op{{Kind: LoadState, Imm: 0}, {Kind: LoadState, Imm: 1}, {Kind: LoadState, Imm: 2}}
	slot := 3
	add := func(kind int, encoding uint64) {
		index := Value(len(ops))
		ops = append(ops, Op{Kind: kind, A: 0, B: 1, Imm: encoding}, Op{Kind: StoreState, A: index, Imm: uint64(slot)})
		slot++
	}
	for _, kind := range []int{Add, Sub, Mul, And, Or, Xor, Equal, Less} {
		add(kind, 0)
	}
	for _, amount := range []uint64{0, 1, 12, 31, 32, 63, 64} {
		add(Shl, amount)
		add(Shr, amount)
	}
	for _, width := range []uint64{32, 64} {
		for _, kind := range []int{VariableShl, VariableShr, ArithmeticShr, RotateRight, UnsignedDivide, SignedDivide, UnsignedRemainder, SignedRemainder} {
			add(kind, width)
		}
		for _, kind := range []int{LeadingZeros, LeadingSigns, ReverseBits} {
			add(kind, width)
		}
		for _, group := range []uint64{16, 32, 64} {
			if group <= width {
				add(ReverseBytes, width|group<<8)
			}
		}
		for _, sub := range []uint64{0, 128} {
			add(ArithmeticFlags, width|sub)
			add(CarryArithmetic, width|sub|2<<8)
			add(CarryFlags, width|sub|2<<8)
			for condition := uint64(0); condition < 16; condition++ {
				add(ArithmeticCondition, width|sub|condition<<8)
			}
		}
		for condition := uint64(0); condition < 16; condition++ {
			add(LogicalCondition, width|condition<<8)
		}
	}
	for _, width := range []uint64{8, 16, 32, 64} {
		add(StatusBits, width)
		add(ArithmeticStatus, width)
		add(ArithmeticStatus, width|128)
	}
	add(ByteParity, 0)
	add(UnsignedMulHigh, 0)
	add(SignedMulHigh, 0)
	entry, err := n.Compile(ops, slot)
	if err != nil {
		return err
	}
	seeds := []uint64{0, 1, 2, 31, 32, 63, 64, 255, 0xffffffff, 0x100000000, 0x7fffffff, 0x80000000, 0x7fffffffffffffff, 0x8000000000000000, 0xffff800080000000, 0xffffffffff5fe000, ^uint64(0)}
	for _, a := range seeds {
		for _, b := range seeds {
			for carry := uint64(0); carry < 2; carry++ {
				state := make([]uint64, slot)
				state[0], state[1], state[2] = a, b, carry
				want := append([]uint64(nil), state...)
				if err = Interpret(ops, want); err != nil {
					return err
				}
				if err = n.Call(entry, state); err != nil {
					return err
				}
				for i := 3; i < slot; i++ {
					if state[i] != want[i] {
						return fmt.Errorf("WASM integer mismatch slot=%d a=%d b=%d carry=%d got=%d want=%d", i, a, b, carry, state[i], want[i])
					}
				}
			}
		}
	}
	return nil
}
