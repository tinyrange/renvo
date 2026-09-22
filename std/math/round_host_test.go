//go:build !renvo

package math

import (
	stdmath "math"
	"testing"
)

func TestRoundingAgainstGo(t *testing.T) {
	seed := uint64(1)
	for i := 0; i < 10000; i++ {
		seed = seed*6364136223846793005 + 1
		x := Float64frombits(seed)
		seed = seed*6364136223846793005 + 1
		y := Float64frombits(seed)
		got := []float64{Floor(x), Trunc(x), Mod(x, y)}
		want := []float64{stdmath.Floor(x), stdmath.Trunc(x), stdmath.Mod(x, y)}
		for j := range got {
			if Float64bits(got[j]) != Float64bits(want[j]) && !(IsNaN(got[j]) && IsNaN(want[j])) {
				t.Fatalf("sample %d operation %d x=%x y=%x: got %x want %x", i, j, Float64bits(x), Float64bits(y), Float64bits(got[j]), Float64bits(want[j]))
			}
		}
	}
}
