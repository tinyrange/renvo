//go:build !renvo

package runtime

import (
	"math/big"
	"testing"
)

func TestNativeCarryExplicitInput(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	for _, width := range []int{32, 64} {
		for _, sub := range []bool{false, true} {
			var b Builder
			b.Constant(17) // dead input exercises third-operand remapping
			a, rhs, c := b.Load(0), b.Load(1), b.Load(2)
			b.Store(3, b.Carry(CarryArithmetic, a, rhs, c, width, sub))
			b.Store(4, b.Carry(CarryFlags, a, rhs, c, width, sub))
			ops := b.Finish(5)
			entry, err := n.Compile(ops, 5)
			if err != nil {
				t.Fatal(err)
			}
			inputs := []uint64{0, 1, 2, ^uint64(0), 0x7fffffff, 0x80000000, 0xffffffff, 0x7fffffffffffffff, 0x8000000000000000, 0x123456789abcdef0}
			random := uint64(101)
			for i := 0; i < 96; i++ {
				random ^= random << 13
				random ^= random >> 7
				random ^= random << 17
				inputs = append(inputs, random)
			}
			modulus := new(big.Int).Lsh(big.NewInt(1), uint(width))
			mask := new(big.Int).Sub(new(big.Int).Set(modulus), big.NewInt(1))
			sign := new(big.Int).Rsh(new(big.Int).Set(modulus), 1)
			for _, x := range inputs {
				for _, y := range inputs {
					for c := uint64(0); c < 4; c++ {
						ax := new(big.Int).And(new(big.Int).SetUint64(x), mask)
						by := new(big.Int).And(new(big.Int).SetUint64(y), mask)
						exact := new(big.Int).Set(ax)
						if sub {
							exact.Sub(exact, by)
							exact.Sub(exact, new(big.Int).SetUint64(1-(c&1)))
						} else {
							exact.Add(exact, by)
							exact.Add(exact, new(big.Int).SetUint64(c&1))
						}
						result := new(big.Int).And(new(big.Int).Set(exact), mask).Uint64()
						flags := uint64(0)
						if result == 0 {
							flags |= 1 << 30
						}
						if result>>(width-1) != 0 {
							flags |= 1 << 31
						}
						if sub && exact.Sign() >= 0 || !sub && exact.Cmp(modulus) >= 0 {
							flags |= 1 << 29
						}
						signedA, signedB := new(big.Int).Set(ax), new(big.Int).Set(by)
						if signedA.Cmp(sign) >= 0 {
							signedA.Sub(signedA, modulus)
						}
						if signedB.Cmp(sign) >= 0 {
							signedB.Sub(signedB, modulus)
						}
						signed := new(big.Int).Set(signedA)
						if sub {
							signed.Sub(signed, signedB)
							signed.Sub(signed, new(big.Int).SetUint64(1-(c&1)))
						} else {
							signed.Add(signed, signedB)
							signed.Add(signed, new(big.Int).SetUint64(c&1))
						}
						min := new(big.Int).Neg(new(big.Int).Set(sign))
						if signed.Cmp(min) < 0 || signed.Cmp(sign) >= 0 {
							flags |= 1 << 28
						}
						state := []uint64{x, y, c, 0, 0}
						portable := append([]uint64(nil), state...)
						if err = Interpret(ops, portable); err != nil {
							t.Fatal(err)
						}
						if err = n.Call(entry, state); err != nil {
							t.Fatal(err)
						}
						if state[3] != result || state[4] != flags || portable[3] != result || portable[4] != flags {
							t.Fatalf("width=%d sub=%v a=%x b=%x c=%d native=%x portable=%x want=%x,%x", width, sub, x, y, c, state, portable, result, flags)
						}
					}
				}
			}
		}
	}
	if Validate([]Op{{Kind: LoadState}, {Kind: CarryArithmetic, A: 0, B: 0, Imm: 64 | 2<<8}}, 1) == nil {
		t.Fatal("forward carry operand accepted")
	}
}
