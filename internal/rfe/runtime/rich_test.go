//go:build !renvo

package runtime

import (
	"math/big"
	"testing"
)

func TestNativeRichIntegerSelection(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	var b Builder
	a, rhs := b.Load(0), b.Load(1)
	b.Store(2, b.MultiplyHigh(a, rhs, false))
	b.Store(3, b.MultiplyHigh(a, rhs, true))
	slot := 4
	for _, width := range []int{32, 64} {
		for _, kind := range []int{LeadingZeros, LeadingSigns, ReverseBits} {
			b.Store(slot, b.RichUnary(kind, a, width, 0))
			slot++
		}
		for group := 16; group <= width; group *= 2 {
			b.Store(slot, b.RichUnary(ReverseBytes, a, width, group))
			slot++
		}
	}
	ops := b.Finish(slot)
	entry, err := n.Compile(ops, slot)
	if err != nil {
		t.Fatal(err)
	}
	inputs := []uint64{0, 1, 2, 0xffffffff, 0x80000000, 0x7fffffff, 0x8000000000000000, 0x7fffffffffffffff, ^uint64(0), 0x123456789abcdef0}
	random := uint64(73)
	for i := 0; i < 128; i++ {
		random ^= random << 13
		random ^= random >> 7
		random ^= random << 17
		inputs = append(inputs, random)
	}
	for i, a := range inputs {
		rhs := inputs[(i*37+3)%len(inputs)]
		state := make([]uint64, slot)
		state[0], state[1] = a, rhs
		want := append([]uint64(nil), state...)
		if err = Interpret(ops, want); err != nil {
			t.Fatal(err)
		}
		// Independent full-width signed and unsigned multiplication oracle.
		unsigned := new(big.Int).Mul(new(big.Int).SetUint64(a), new(big.Int).SetUint64(rhs))
		unsigned.Rsh(unsigned, 64)
		signed := new(big.Int).Mul(big.NewInt(int64(a)), big.NewInt(int64(rhs)))
		signed.Rsh(signed, 64)
		signed.And(signed, new(big.Int).SetUint64(^uint64(0)))
		if want[2] != unsigned.Uint64() || want[3] != signed.Uint64() {
			t.Fatal("portable multiply", a, rhs, want)
		}
		if err = n.Call(entry, state); err != nil {
			t.Fatal(err)
		}
		for j := range state {
			if state[j] != want[j] {
				t.Fatalf("trial %d slot %d: %x want %x (a=%x rhs=%x)", i, j, state[j], want[j], a, rhs)
			}
		}
	}
	for _, bad := range []Op{{Kind: LeadingZeros, A: 0, B: 0, Imm: 16}, {Kind: ReverseBytes, A: 0, B: 0, Imm: 32 | 64<<8}, {Kind: ReverseBits, A: 0, B: 0, Imm: 64 | 1<<8}} {
		if Validate([]Op{{Kind: LoadState}, bad}, 1) == nil {
			t.Fatal("invalid rich encoding accepted", bad)
		}
	}
}
