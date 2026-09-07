package check

import (
	"math/big"
	"testing"
)

func testWideBig(value wideConstant) *big.Int {
	out := new(big.Int)
	for i := len(value.words) - 1; i >= 0; i-- {
		out.Lsh(out, 15)
		out.Add(out, big.NewInt(int64(value.words[i])))
	}
	if value.negative {
		out.Neg(out)
	}
	return out
}

func TestWideDivide(t *testing.T) {
	for _, leftBits := range []int{0, 1, 14, 15, 16, 31, 63, 64, 100, 255, 1024} {
		for _, rightBits := range []int{0, 1, 14, 15, 16, 63, 64, 100, 511} {
			for _, signs := range []int{0, 1, 2, 3} {
				left := wideAdd(wideShift(wideSmall(1), leftBits, true), wideSmall(31))
				right := wideAdd(wideShift(wideSmall(1), rightBits, true), wideSmall(7))
				left.negative, right.negative = signs&1 != 0, signs&2 != 0
				l, r := testWideBig(left), testWideBig(right)
				wantQ, wantR := new(big.Int), new(big.Int)
				wantQ.QuoRem(l, r, wantR)
				q, rem := wideDivide(left, right)
				if !q.ok || !rem.ok || testWideBig(q).Cmp(wantQ) != 0 || testWideBig(rem).Cmp(wantR) != 0 {
					t.Fatalf("division at %d/%d signs %d", leftBits, rightBits, signs)
				}
				if testWideBig(left).Cmp(l) != 0 || testWideBig(right).Cmp(r) != 0 {
					t.Fatal("mutated operand")
				}
			}
		}
	}
	for _, test := range [][2]int{{0, 1}, {1, 1}, {-1, 1}, {1, -1}, {-1, -1}, {10, 2}, {-10, 2}, {10, -2}, {-10, -2}, {1, 2}, {-1, 2}} {
		q, r := wideDivide(wideSmall(test[0]), wideSmall(test[1]))
		qv, qok := wideInt(q)
		rv, rok := wideInt(r)
		if !qok || !rok || qv != test[0]/test[1] || rv != test[0]%test[1] || qv == 0 && q.negative || rv == 0 && r.negative {
			t.Fatalf("small division %v", test)
		}
	}
	if q, r := wideDivide(wideSmall(1), wideSmall(0)); q.ok || r.ok {
		t.Fatal("accepted division by zero")
	}
	if q, r := wideDivide(wideConstant{}, wideSmall(1)); q.ok || r.ok {
		t.Fatal("accepted unknown dividend")
	}
}
