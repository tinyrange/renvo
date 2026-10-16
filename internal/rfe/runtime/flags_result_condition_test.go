//go:build !renvo

package runtime

import "testing"

// The oracle uses the result's architectural sign/zero and independently
// enumerated conditions, not flagsOperation or the builder's simplifier.
func TestResultConditionsKnownRanges(t *testing.T) {
	n, err := NewNative(4 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	for _, width := range []int{32, 64} {
		wordMask := ^uint64(0) >> (64 - uint(width))
		for _, mask := range []uint64{1, 255, 0x7fffffff, 0xffffffff, 0x7fffffffffffffff, ^uint64(0)} {
			for mode := 0; mode < 3; mode++ {
				var b Builder
				value := b.Binary(And, b.Load(0), b.Constant(mask))
				// This is zero in the flag width, but may have live high bits.
				rhs := b.Binary(And, b.Load(1), b.Constant(^wordMask))
				flags := b.LogicalFlags(value, width)
				if mode != 0 {
					flags = b.ArithmeticFlags(value, rhs, width, mode == 2)
				}
				b.Store(2, flags)
				for condition := 0; condition < 16; condition++ {
					predicate, ok := b.FlagCondition(flags, condition)
					if !ok {
						t.Fatal("missing condition")
					}
					if mask&(uint64(1)<<(width-1)) == 0 && (condition == 4 || condition == 11) {
						if b.Ops[predicate].Kind != Const || b.Ops[predicate].Imm != 0 {
							t.Fatal("lost known nonnegative range")
						}
					}
					b.Store(3+condition, predicate)
				}
				ops := b.Finish(19)
				entry, err := n.Compile(ops, 19)
				if err != nil {
					t.Fatal(err)
				}
				prepared, err := Prepare(ops, 19)
				if err != nil {
					t.Fatal(err)
				}
				for _, input := range []uint64{0, 1, 2, 255, 256, 0x7fffffff, 0x80000000, 0xffffffff, 0x100000000, 0x7fffffffffffffff, 0x8000000000000000, ^uint64(0)} {
					initial := make([]uint64, 19)
					initial[0], initial[1] = input, ^uint64(0)
					result := input & mask & wordMask
					negative, zero, carry := result>>(width-1) != 0, result == 0, mode == 2
					var packed uint64
					if negative {
						packed |= 1 << 31
					}
					if zero {
						packed |= 1 << 30
					}
					if carry {
						packed |= 1 << 29
					}
					conditions := []bool{zero, !zero, carry, !carry, negative, !negative, false, true, carry && !zero, !carry || zero, !negative, negative, !negative && !zero, negative || zero, true, true}
					for _, native := range []bool{false, true} {
						state := append([]uint64(nil), initial...)
						if native {
							err = n.Call(entry, state)
						} else {
							var machine IRMachine
							err = machine.Run(prepared, state)
						}
						if err != nil {
							t.Fatal(err)
						}
						if state[2] != packed {
							t.Fatalf("width=%d mask=%x mode=%d input=%x native=%v: packed %x want %x", width, mask, mode, input, native, state[2], packed)
						}
						for condition, want := range conditions {
							got := state[3+condition]
							if got > 1 || (got == 1) != want {
								t.Fatalf("width=%d mask=%x mode=%d input=%x native=%v condition=%d: %d want %v", width, mask, mode, input, native, condition, got, want)
							}
						}
					}
				}
			}
		}
	}
}
