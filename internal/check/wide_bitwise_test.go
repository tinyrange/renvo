package check

import (
	"math/big"
	"testing"
)

func TestWideBitwise(t *testing.T) {
	for _, leftBits := range []int{0, 1, 14, 15, 16, 31, 63, 64, 100, 255, 1024} {
		for _, rightBits := range []int{0, 1, 14, 15, 16, 63, 64, 100, 511} {
			for signs := 0; signs < 4; signs++ {
				left := wideAdd(wideShift(wideSmall(1), leftBits, true), wideSmall(31))
				right := wideAdd(wideShift(wideSmall(1), rightBits, true), wideSmall(7))
				left.negative, right.negative = signs&1 != 0, signs&2 != 0
				l, r := testWideBig(left), testWideBig(right)
				for _, op := range []string{"&", "|", "^", "&^"} {
					want := new(big.Int)
					switch op {
					case "&":
						want.And(l, r)
					case "|":
						want.Or(l, r)
					case "^":
						want.Xor(l, r)
					case "&^":
						want.AndNot(l, r)
					}
					got := wideBitwise(left, right, op)
					if !got.ok || testWideBig(got).Cmp(want) != 0 {
						t.Fatalf("%d %s %d signs %d", leftBits, op, rightBits, signs)
					}
					if testWideBig(left).Cmp(l) != 0 || testWideBig(right).Cmp(r) != 0 {
						t.Fatal("mutated operand")
					}
				}
			}
		}
	}
	for _, left := range []int{-32769, -32768, -1, 0, 1, 32767, 32768} {
		for _, right := range []int{-32769, -32768, -1, 0, 1, 32767, 32768} {
			for op, want := range map[string]int{"&": left & right, "|": left | right, "^": left ^ right, "&^": left &^ right} {
				got := wideBitwise(wideSmall(left), wideSmall(right), op)
				v, ok := wideInt(got)
				if !ok || v != want || v == 0 && got.negative {
					t.Fatalf("%d %s %d", left, op, right)
				}
			}
		}
	}
	if wideBitwise(wideConstant{}, wideSmall(1), "&").ok || wideBitwise(wideSmall(1), wideSmall(1), "?").ok {
		t.Fatal("accepted unknown input/operator")
	}
}
