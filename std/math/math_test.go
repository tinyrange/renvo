package math

import "testing"

func almostEqual(a, b float64) bool {
	if a == b {
		return true
	}
	d := a - b
	if d < 0 {
		d = -d
	}
	scale := Abs(a)
	if Abs(b) > scale {
		scale = Abs(b)
	}
	if scale < 1e-300 {
		return d < 1e-300
	}
	return d/scale < 1e-13
}

func TestConstants(t *testing.T) {
	if !almostEqual(Pi, 3.141592653589793) || !almostEqual(E, 2.718281828459045) {
		t.Fatalf("Pi=%v E=%v", Pi, E)
	}
	if !almostEqual(Sqrt2, 1.4142135623730951) || !almostEqual(Ln2, 0.6931471805599453) {
		t.Fatalf("Sqrt2=%v Ln2=%v", Sqrt2, Ln2)
	}
	if MaxInt32 != 2147483647 || MinInt32 != -2147483648 {
		t.Fatalf("int32 bounds %d %d", MaxInt32, MinInt32)
	}
}

func TestAbsSignAndCopy(t *testing.T) {
	negZero := Copysign(0, -1)
	if Abs(-3.5) != 3.5 || Abs(3.5) != 3.5 {
		t.Fatal("abs")
	}
	if Abs(negZero) != 0 || Signbit(Abs(negZero)) {
		t.Fatal("abs of negative zero must be positive zero")
	}
	if !Signbit(negZero) || Signbit(0.0) || !Signbit(-2) || Signbit(2) {
		t.Fatal("signbit")
	}
	if Copysign(3, -1) != -3 || Copysign(-3, 1) != 3 {
		t.Fatal("copysign")
	}
	if !IsNaN(NaN()) || IsNaN(1.5) {
		t.Fatal("isnan")
	}
	if !IsInf(Inf(1), 1) || !IsInf(Inf(-1), -1) || IsInf(1.5, 0) {
		t.Fatal("isinf")
	}
}

func TestRounding(t *testing.T) {
	cases := []struct {
		x                  float64
		trunc, floor, ceil float64
		round, rte         float64
	}{
		{2.7, 2, 2, 3, 3, 3},
		{2.5, 2, 2, 3, 3, 2},
		{3.5, 3, 3, 4, 4, 4},
		{-2.5, -2, -3, -2, -3, -2},
		{-2.7, -2, -3, -2, -3, -3},
		{-0.5, 0, -1, 0, -1, 0},
		{0.5, 0, 0, 1, 1, 0},
		{7.0, 7, 7, 7, 7, 7},
	}
	for _, tc := range cases {
		if Trunc(tc.x) != tc.trunc {
			t.Errorf("Trunc(%v)=%v want %v", tc.x, Trunc(tc.x), tc.trunc)
		}
		if Floor(tc.x) != tc.floor {
			t.Errorf("Floor(%v)=%v want %v", tc.x, Floor(tc.x), tc.floor)
		}
		if Ceil(tc.x) != tc.ceil {
			t.Errorf("Ceil(%v)=%v want %v", tc.x, Ceil(tc.x), tc.ceil)
		}
		if Round(tc.x) != tc.round {
			t.Errorf("Round(%v)=%v want %v", tc.x, Round(tc.x), tc.round)
		}
		if RoundToEven(tc.x) != tc.rte {
			t.Errorf("RoundToEven(%v)=%v want %v", tc.x, RoundToEven(tc.x), tc.rte)
		}
	}
	if !Signbit(Floor(-0.5)) {
		t.Error("Floor(-0.5) should be negative zero")
	}
	if Round(1e300) != 1e300 {
		t.Error("large values are already integral")
	}
}

func TestMod(t *testing.T) {
	cases := []struct {
		x, y, want float64
	}{
		{7, 3, 1},
		{-7, 3, -1},
		{7, -3, 1},
		{-7, -3, -1},
		{6, 3, 0},
		{0.5, 1, 0.5},
		{1e18, 1, 0},
		{2.5, 2, 0.5},
	}
	for _, tc := range cases {
		if got := Mod(tc.x, tc.y); got != tc.want {
			t.Errorf("Mod(%v,%v)=%v want %v", tc.x, tc.y, got, tc.want)
		}
	}
	if !Signbit(Mod(-4, 2)) {
		t.Error("Mod(-4,2) should keep the sign of x as negative zero")
	}
	if !IsNaN(Mod(1, 0)) || !IsNaN(Mod(Inf(1), 2)) {
		t.Error("Mod domain errors")
	}
	if Mod(5, Inf(1)) != 5 {
		t.Error("Mod with infinite divisor")
	}
}

func TestSqrtCbrtHypot(t *testing.T) {
	if !almostEqual(Sqrt(2), 1.4142135623730951) || Sqrt(9) != 3 || Sqrt(0) != 0 {
		t.Fatalf("sqrt: %v %v", Sqrt(2), Sqrt(9))
	}
	if !almostEqual(Sqrt(1e308), 1e154) {
		t.Fatalf("sqrt large = %v", Sqrt(1e308))
	}
	if !almostEqual(Sqrt(1e-300), 1e-150) {
		t.Fatalf("sqrt small = %v", Sqrt(1e-300))
	}
	if !IsNaN(Sqrt(-1)) {
		t.Fatal("sqrt of negative should be NaN")
	}
	if Cbrt(27) != 3 || Cbrt(-8) != -2 || Cbrt(0) != 0 {
		t.Fatalf("cbrt exact: %v %v", Cbrt(27), Cbrt(-8))
	}
	if !almostEqual(Cbrt(10), 2.154434690031884) || !almostEqual(Cbrt(1e-18), 1e-6) {
		t.Fatalf("cbrt approx: %v %v", Cbrt(10), Cbrt(1e-18))
	}
	if Hypot(3, 4) != 5 || Hypot(-3, -4) != 5 || Hypot(0, 0) != 0 {
		t.Fatal("hypot")
	}
	if !almostEqual(Hypot(1e300, 1e300), 1.4142135623730951e300) {
		t.Fatalf("hypot overflow safety = %v", Hypot(1e300, 1e300))
	}
}

func TestExpLog(t *testing.T) {
	if !almostEqual(Exp(1), E) || Exp(0) != 1 {
		t.Fatalf("exp basic: %v", Exp(1))
	}
	if !almostEqual(Exp(-1), 0.36787944117144233) {
		t.Fatalf("exp(-1) = %v", Exp(-1))
	}
	if !almostEqual(Exp(709), 8.218407461554972e+307) {
		t.Fatalf("exp(709) = %v", Exp(709))
	}
	if !IsInf(Exp(710), 1) || Exp(-750) != 0 {
		t.Fatal("exp limits")
	}
	if Log(1) != 0 || !almostEqual(Log(E), 1) {
		t.Fatalf("log basic: %v", Log(E))
	}
	if !almostEqual(Log(8)/Log(2), 3) {
		t.Fatalf("log(8)/log(2) = %v", Log(8)/Log(2))
	}
	if !IsInf(Log(0), -1) || !IsNaN(Log(-1)) {
		t.Fatal("log domain")
	}
	if !almostEqual(Log2(1024), 10) || !almostEqual(Log10(1000), 3) {
		t.Fatalf("log2/log10: %v %v", Log2(1024), Log10(1000))
	}
	if !almostEqual(Exp2(10), 1024) || !almostEqual(Exp2(-1), 0.5) {
		t.Fatalf("exp2: %v %v", Exp2(10), Exp2(-1))
	}
	if !almostEqual(Log(1e300), 690.7755278982137) {
		t.Fatalf("log(1e300) = %v", Log(1e300))
	}
	if !almostEqual(Log(1e-300), -690.7755278982137) {
		t.Fatalf("log(1e-300) = %v", Log(1e-300))
	}
}

func TestPow(t *testing.T) {
	cases := []struct {
		x, y, want float64
	}{
		{2, 10, 1024},
		{2, -2, 0.25},
		{-2, 3, -8},
		{-2, 2, 4},
		{9, 0.5, 3},
		{1, 100, 1},
		{5, 0, 1},
		{0.5, 3, 0.125},
		{10, 308, 1e308},
	}
	for _, tc := range cases {
		if got := Pow(tc.x, tc.y); !almostEqual(got, tc.want) {
			t.Errorf("Pow(%v,%v)=%v want %v", tc.x, tc.y, got, tc.want)
		}
	}
	if !IsNaN(Pow(-8, 0.5)) {
		t.Error("fractional power of negative base should be NaN")
	}
	if !IsInf(Pow(0, -1), 1) || Pow(0, 2) != 0 {
		t.Error("zero base rules")
	}
	if !almostEqual(Pow(2, 0.5), Sqrt(2)) {
		t.Errorf("Pow(2,0.5)=%v", Pow(2, 0.5))
	}
}

func TestMaxMinDim(t *testing.T) {
	negZero := Copysign(0, -1)
	if Max(2, 3) != 3 || Min(2, 3) != 2 || Max(-1, -5) != -1 {
		t.Fatal("max/min basics")
	}
	if Max(2, NaN()) == Max(2, NaN()) {
		t.Error("Max with NaN should be NaN")
	}
	if Min(NaN(), 2) == Min(NaN(), 2) {
		t.Error("Min with NaN should be NaN")
	}
	if Signbit(Max(0.0, negZero)) {
		t.Error("Max(+0,-0) is +0")
	}
	if !Signbit(Min(0.0, negZero)) {
		t.Error("Min(+0,-0) is -0")
	}
	if Dim(5, 3) != 2 || Dim(3, 5) != 0 || Dim(-1, -3) != 2 {
		t.Fatal("dim")
	}
}

func TestFrexpLdexp(t *testing.T) {
	frac, exp := Frexp(8)
	if frac != 0.5 || exp != 4 {
		t.Fatalf("Frexp(8) = %v, %d", frac, exp)
	}
	frac, exp = Frexp(0)
	if frac != 0 || exp != 0 {
		t.Fatalf("Frexp(0) = %v, %d", frac, exp)
	}
	if Ldexp(0.5, 4) != 8 || Ldexp(0, 100) != 0 {
		t.Fatal("ldexp")
	}
	if Ldexp(1, 1100) != Inf(1) {
		t.Fatal("ldexp overflow")
	}
	bigFrac, bigExp := Frexp(1e300)
	if !almostEqual(Ldexp(bigFrac, bigExp), 1e300) {
		t.Fatal("frexp/ldexp round trip")
	}
}
