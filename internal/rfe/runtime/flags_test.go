//go:build !renvo

package runtime

import (
	"math/big"
	"math/bits"
	"testing"
)

func TestNativeFlagsWidthsAndLiveValues(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	for _, width := range []int{32, 64} {
		for _, sub := range []bool{false, true} {
			var b Builder
			// Keep unrelated values live across flag capture and stress register reuse.
			live := make([]Value, 12)
			for i := range live {
				live[i] = b.Binary(Xor, b.Load(i+2), b.Constant(uint64(i+17)))
			}
			arithmetic := b.ArithmeticFlags(b.Load(0), b.Load(1), width, sub)
			logical := b.LogicalFlags(b.Load(0), width)
			b.Store(14, arithmetic)
			b.Store(15, logical)
			for condition := 0; condition < 16; condition++ {
				a, ok := b.FlagCondition(arithmetic, condition)
				if !ok {
					t.Fatal("missing arithmetic predicate")
				}
				b.Store(17+condition, a)
				l, ok := b.FlagCondition(logical, condition)
				if !ok {
					t.Fatal("missing logical predicate")
				}
				b.Store(33+condition, l)
			}
			sum := b.Constant(0)
			for _, v := range live {
				sum = b.Binary(Add, sum, v)
			}
			b.Store(16, sum)
			ops := b.Finish(49)
			entry, err := n.Compile(ops, 49)
			if err != nil {
				t.Fatal(err)
			}
			prepared, err := Prepare(ops, 49)
			if err != nil {
				t.Fatal(err)
			}
			words := []uint64{0, 1, 0x7fffffff, 0x80000000, 0xffffffff, 0x7fffffffffffffff, 0x8000000000000000, ^uint64(0)}
			for trial := 0; trial < 256; trial++ {
				a, rhs := words[trial%len(words)], words[(trial/len(words))%len(words)]
				if trial >= 64 {
					a = bits.RotateLeft64(uint64(trial)*0x9e3779b97f4a7c15, trial)
					rhs = bits.RotateLeft64(^a, trial/3)
				}
				state := make([]uint64, 49)
				state[0], state[1] = a, rhs
				var wantSum uint64
				for i := 2; i < 14; i++ {
					state[i] = uint64(i*7919 + trial)
					wantSum += state[i] ^ uint64(i-2+17)
				}
				m := ^uint64(0)
				if width == 32 {
					m = 0xffffffff
				}
				a &= m
				rhs &= m
				result := a + rhs
				carry := false
				if sub {
					result = a - rhs
					carry = a >= rhs
				} else if width == 32 {
					carry = a+rhs > m
				} else {
					_, c := bits.Add64(a, rhs, 0)
					carry = c != 0
				}
				result &= m
				signed := func(x uint64) *big.Int {
					v := new(big.Int).SetUint64(x)
					if x>>(width-1) != 0 {
						v.Sub(v, new(big.Int).Lsh(big.NewInt(1), uint(width)))
					}
					return v
				}
				exact := signed(a)
				if sub {
					exact.Sub(exact, signed(rhs))
				} else {
					exact.Add(exact, signed(rhs))
				}
				limit := new(big.Int).Lsh(big.NewInt(1), uint(width-1))
				min := new(big.Int).Neg(new(big.Int).Set(limit))
				max := new(big.Int).Sub(limit, big.NewInt(1))
				var wantFlags, wantLogical uint64
				if result>>(width-1) != 0 {
					wantFlags |= 1 << 31
				}
				if result == 0 {
					wantFlags |= 1 << 30
				}
				if carry {
					wantFlags |= 1 << 29
				}
				if exact.Cmp(min) < 0 || exact.Cmp(max) > 0 {
					wantFlags |= 1 << 28
				}
				if a>>(width-1) != 0 {
					wantLogical |= 1 << 31
				}
				if a == 0 {
					wantLogical |= 1 << 30
				}
				native := append([]uint64(nil), state...)
				if err = n.Call(entry, native); err != nil {
					t.Fatal(err)
				}
				var machine IRMachine
				if err = machine.Run(prepared, state); err != nil {
					t.Fatal(err)
				}

				for _, got := range [][]uint64{native, state} {
					for which, flags := range []uint64{wantFlags, wantLogical} {
						n, z, c, v := flags&(1<<31) != 0, flags&(1<<30) != 0, flags&(1<<29) != 0, flags&(1<<28) != 0
						truth := []bool{z, !z, c, !c, n, !n, v, !v, c && !z, !c || z, n == v, n != v, n == v && !z, n != v || z, true, true}
						for condition, want := range truth {
							actual := got[17+which*16+condition]
							if actual > 1 || (actual == 1) != want {
								t.Fatalf("width %d sub %v trial %d flags %x cond %d: got %d want %v", width, sub, trial, flags, condition, actual, want)
							}
						}
					}

					if got[14] != wantFlags || got[15] != wantLogical || got[16] != wantSum {
						t.Fatalf("width %d sub %v trial %d: flags %x logical %x sum %x want %x %x %x", width, sub, trial, got[14], got[15], got[16], wantFlags, wantLogical, wantSum)
					}
				}
			}
		}
	}
	for _, k := range []int{ArithmeticFlags, LogicalFlags} {
		for _, width := range []uint64{0, 16, 31, 65, 128, 224, ^uint64(0)} {
			ops := []Op{{Kind: LoadState}, {Kind: k, A: 0, B: 0, Imm: width}, {Kind: StoreState, A: 1}}
			if Validate(ops, 1) == nil || ValidateMemory(ops, 1) == nil {
				t.Fatal("accepted invalid flags width", k, width)
			}
		}
	}
}
