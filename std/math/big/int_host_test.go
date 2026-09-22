//go:build !renvo

package big

import (
	stdbig "math/big"
	"testing"
)

func TestIntegerAgainstGo(t *testing.T) {
	seed := uint64(321)
	for iteration := 0; iteration < 160; iteration++ {
		a, b := new(stdbig.Int), new(stdbig.Int)
		for limb := 0; limb < 1+iteration%8; limb++ {
			seed = seed*6364136223846793005 + 1
			a.Lsh(a, 64).Add(a, new(stdbig.Int).SetUint64(seed))
			seed = seed*6364136223846793005 + 1
			b.Lsh(b, 64).Add(b, new(stdbig.Int).SetUint64(seed))
		}
		if iteration%2 == 0 {
			a.Neg(a)
		}
		if iteration%3 == 0 {
			b.Neg(b)
		}
		x, y := integer(t, a.String()), integer(t, b.String())
		for op := 0; op < 10; op++ {
			want := new(stdbig.Int)
			got := new(Int)
			switch op {
			case 0:
				want.Add(a, b)
				got.Add(x, y)
			case 1:
				want.Sub(a, b)
				got.Sub(x, y)
			case 2:
				want.Mul(a, b)
				got.Mul(x, y)
			case 3:
				want.Quo(a, b)
				got.Quo(x, y)
			case 4:
				want.Rem(a, b)
				got.Rem(x, y)
			case 5:
				want.And(a, b)
				got.And(x, y)
			case 6:
				want.Or(a, b)
				got.Or(x, y)
			case 7:
				want.Xor(a, b)
				got.Xor(x, y)
			case 8:
				want.Lsh(a, uint(iteration))
				got.Lsh(x, uint(iteration))
			case 9:
				want.Rsh(a, uint(iteration))
				got.Rsh(x, uint(iteration))
			}
			if got.String() != want.String() {
				t.Fatalf("iteration %d op %d: got %s want %s", iteration, op, got, want)
			}
		}
		for base := 2; base <= 62; base++ {
			if x.Text(base) != a.Text(base) {
				t.Fatalf("base %d", base)
			}
		}
	}
}
