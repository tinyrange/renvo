package math

import "testing"

func TestRounding(t *testing.T) {
	xs := []float64{0, Copysign(0, -1), 1.75, -1.75, SmallestNonzeroFloat64, -SmallestNonzeroFloat64, MaxFloat64, -MaxFloat64, Inf(1), Inf(-1)}
	floors := []float64{0, Copysign(0, -1), 1, -2, 0, -1, MaxFloat64, -MaxFloat64, Inf(1), Inf(-1)}
	truncs := []float64{0, Copysign(0, -1), 1, -1, 0, Copysign(0, -1), MaxFloat64, -MaxFloat64, Inf(1), Inf(-1)}
	for i, x := range xs {
		if Float64bits(Floor(x)) != Float64bits(floors[i]) || Float64bits(Trunc(x)) != Float64bits(truncs[i]) {
			t.Fatal("rounding", i)
		}
	}
	if !IsNaN(Floor(NaN())) || !IsNaN(Trunc(NaN())) {
		t.Fatal("NaN rounding")
	}
}

func TestMod(t *testing.T) {
	xs := []float64{5.5, -5.5, 5.5, -4, MaxFloat64, SmallestNonzeroFloat64 * 7, SmallestNonzeroFloat64 * 7, 1, Copysign(0, -1)}
	ys := []float64{2, 2, -2, 2, 3, SmallestNonzeroFloat64 * 3, SmallestNonzeroFloat64 * 8, Inf(1), 2}
	wants := []float64{1.5, -1.5, 1.5, Copysign(0, -1), 2, SmallestNonzeroFloat64, SmallestNonzeroFloat64 * 7, 1, Copysign(0, -1)}
	for i, x := range xs {
		if Float64bits(Mod(x, ys[i])) != Float64bits(wants[i]) {
			t.Fatal("remainder", i, Mod(x, ys[i]), wants[i])
		}
	}
	if !IsNaN(Mod(1, 0)) || !IsNaN(Mod(Inf(1), 1)) || !IsNaN(Mod(NaN(), 1)) || !IsNaN(Mod(1, NaN())) {
		t.Fatal("NaN remainder")
	}
}
