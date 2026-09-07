//go:build !renvo

package big

import (
	"math"
	stdbig "math/big"
	"testing"
)

func TestConversionsAgainstGo(t *testing.T) {
	seed := uint64(789)
	for i := 0; i < 250; i++ {
		seed = seed*6364136223846793005 + 1
		value := math.Float64frombits(seed)
		if math.IsNaN(value) || math.IsInf(value, 0) {
			continue
		}
		gotRat := new(Rat).SetFloat64(value).RatString()
		wantRat := new(stdbig.Rat).SetFloat64(value).RatString()
		if gotRat != wantRat {
			t.Fatalf("rational bits %x: %s want %s", seed, gotRat, wantRat)
		}
		got, acc := NewFloat(value).Int(nil)
		want, wantAcc := new(stdbig.Float).SetFloat64(value).Int(nil)
		if got.String() != want.String() || int(acc) != int(wantAcc) {
			t.Fatalf("int bits %x", seed)
		}
		large := new(stdbig.Int).Lsh(new(stdbig.Int).SetUint64(seed), uint(i*5))
		large.Add(large, stdbig.NewInt(int64(i)))
		if i%2 == 0 {
			large.Neg(large)
		}
		x := integer(t, large.String())
		f, accuracy := new(Float).SetInt(x).Float64()
		expected, expectedAccuracy := new(stdbig.Float).SetInt(large).Float64()
		if math.Float64bits(f) != math.Float64bits(expected) || int(accuracy) != int(expectedAccuracy) {
			t.Fatalf("integer Float64 %d: %g %v, want %g %v", i, f, accuracy, expected, expectedAccuracy)
		}
	}
}
