//go:build !renvo

package runtime

import (
	"math/big"
	"testing"
)

// big.Int is an independent arithmetic oracle; the all-ones and overflow cases
// are explicit IR contracts, not host-language exception behavior.
func divisionOracle(kind int, a, b uint64, width int) uint64 {
	mask := ^uint64(0)
	if width == 32 {
		mask = 0xffffffff
	}
	a &= mask
	b &= mask
	if b == 0 {
		if kind == UnsignedRemainder || kind == SignedRemainder {
			return a
		}
		return mask
	}
	x, y := new(big.Int).SetUint64(a), new(big.Int).SetUint64(b)
	modulus := new(big.Int).Lsh(big.NewInt(1), uint(width))
	if kind == SignedDivide || kind == SignedRemainder {
		if a>>(width-1) != 0 {
			x.Sub(x, modulus)
		}
		if b>>(width-1) != 0 {
			y.Sub(y, modulus)
		}
	}
	q, r := new(big.Int), new(big.Int)
	q.QuoRem(x, y, r)
	if kind == UnsignedRemainder || kind == SignedRemainder {
		q = r
	}
	q.Mod(q, modulus)
	return q.Uint64()
}
func TestDivisionAllTiersSpillsAndHalfwordLinks(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(24, 23); err != nil {
		t.Fatal(err)
	}
	values := []uint64{0, 1, 2, 3, 7, 0x7fffffff, 0x80000000, 0xffffffff, 1 << 32, 1 << 63, (1 << 63) - 1, ^uint64(0), ^uint64(0) - 6, 0xfeed123456789abc}
	for _, width := range []int{32, 64} {
		build := func(memory, loop bool) []Op {
			var b Builder
			a, rhs := b.Load(0), b.Load(1)
			live := make([]Value, 20)
			for i := range live {
				live[i] = b.Load(i + 2)
			}
			for k := UnsignedDivide; k <= SignedRemainder; k++ {
				b.Store(k-UnsignedDivide+2, b.Divide(k, a, rhs, width))
			}
			sum := b.Constant(0)
			for _, v := range live {
				sum = b.Binary(Add, sum, v)
			}
			b.Store(22, sum)
			b.Store(23, b.Constant(6))
			if memory {
				b.Checkpoint(24, 1)
				if loop {
					b.LoopContinue(b.Constant(0))
				}
				return b.FinishMemory(24)
			}
			return b.Finish(24)
		}
		ops := build(false, false)
		pure, err := n.Compile(ops, 24)
		if err != nil {
			t.Fatal(err)
		}
		memory, err := n.CompileMemory(build(true, false), 24)
		if err != nil {
			t.Fatal(err)
		}
		loop, err := n.CompileLoop(build(true, true), 24, 1)
		if err != nil {
			t.Fatal(err)
		}
		if err = n.PrepareTargetLink(2, loop, 24, 1); err != nil {
			t.Fatal(err)
		}
		for _, a := range values {
			for _, rhs := range values {
				initial := make([]uint64, 24)
				initial[0], initial[1], initial[23] = a, rhs, 2
				want := append([]uint64(nil), initial...)
				for i := 2; i < 22; i++ {
					initial[i] = uint64(i)*a + rhs
					want[i] = initial[i]
					want[22] += initial[i]
				}
				want[23] = 6
				for k := UnsignedDivide; k <= SignedRemainder; k++ {
					expected := divisionOracle(k, a, rhs, width)
					want[k-UnsignedDivide+2] = expected
					var folded Builder
					v := folded.Divide(k, folded.Constant(a), folded.Constant(rhs), width)
					if folded.Ops[v].Kind != Const || folded.Ops[v].Imm != expected {
						t.Fatal("constant division", width, k, a, rhs)
					}
				}
				for _, mode := range []string{"ir", "native", "memory", "linked", "session"} {
					got := append([]uint64(nil), initial...)
					m := new(MemoryContext)
					switch mode {
					case "ir":
						err = Interpret(ops, got)
					case "native":
						err = n.Call(pure, got)
					case "memory":
						err = n.CallMemory(memory, got, m)
					default:
						m.ClaimLinks(n)
						m.PublishLink(2, loop, 1, [17]uint8{})
						if mode == "session" {
							err = n.RunLinkedSession(got, m, 1)
						} else {
							err = n.CallLinked(got, m, 1)
						}
						if m.Total != 1 || m.Remaining != 0 || m.Status != 0 || m.PreparedTargets != 0 || m.CodeView != [4]uint64{} {
							t.Fatal("halfword linked progress", mode, m.Total, m.Status)
						}
					}
					if err != nil {
						t.Fatal(err)
					}
					for i, v := range want {
						if got[i] != v {
							t.Fatalf("%s width=%d a=%x b=%x slot=%d got=%x want=%x", mode, width, a, rhs, i, got[i], v)
						}
					}
				}
			}
		}
	}
}
func TestDivisionRejectsInvalidWidths(t *testing.T) {
	for k := UnsignedDivide; k <= SignedRemainder; k++ {
		for _, width := range []uint64{0, 1, 16, 33, 65, ^uint64(0)} {
			ops := []Op{{Kind: LoadState}, {Kind: LoadState, Imm: 1}, {Kind: k, A: 0, B: 1, Imm: width}, {Kind: StoreState, A: 2}}
			if Validate(ops, 2) == nil || ValidateMemory(ops, 2) == nil {
				t.Fatal("accepted bad width", k, width)
			}
		}
	}
}
