package big

import (
	"math"
	"testing"
)

func TestRationalConversion(t *testing.T) {
	if new(Rat).SetFloat64(0.1).RatString() != "3602879701896397/36028797018963968" {
		t.Fatal("exact binary fraction")
	}
	if new(Rat).SetFrac(NewInt(6), NewInt(-8)).RatString() != "-3/4" {
		t.Fatal("reduced fraction")
	}
	var zero Rat
	if zero.RatString() != "0" || zero.String() != "0/1" {
		t.Fatal("zero rational")
	}
	if new(Rat).SetInt(NewInt(7)).Cmp(new(Rat).SetFloat64(7)) != 0 {
		t.Fatal("integer equality")
	}
	if new(Rat).SetInt(integer(t, "9007199254740993")).Cmp(new(Rat).SetFloat64(9007199254740992)) <= 0 {
		t.Fatal("precision loss in comparison")
	}
	if new(Rat).SetFloat64(math.Inf(1)) != nil || new(Rat).SetFloat64(math.NaN()) != nil {
		t.Fatal("nonfinite rational")
	}
}

func TestFloatConversion(t *testing.T) {
	for _, bits := range []uint64{0, uint64(1) << 63, 1, 4503599627370495, 4503599627370496, 4607182418800017408, 13830554455654793216, 9218868437227405311} {
		value := math.Float64frombits(bits)
		got, accuracy := NewFloat(value).Float64()
		if math.Float64bits(got) != bits || accuracy != Exact {
			t.Fatal("binary64 round trip")
		}
	}
	for _, value := range []float64{-1.75, -0.5, 0, 0.5, 1.75, 4503599627370496} {
		got, accuracy := NewFloat(value).Int(nil)
		if got.Int64() != int64(value) {
			t.Fatal("truncate float")
		}
		want := Exact
		if float64(int64(value)) < value {
			want = Below
		}
		if float64(int64(value)) > value {
			want = Above
		}
		if accuracy != want {
			t.Fatal("truncate accuracy")
		}
	}
	for _, negative := range []bool{false, true} {
		value := integer(t, "9007199254740993")
		if negative {
			value.Neg(value)
		}
		got, accuracy := new(Float).SetInt(value).Float64()
		want := float64(9007199254740992)
		direction := Below
		if negative {
			want = -want
			direction = Above
		}
		if got != want || accuracy != direction {
			t.Fatal("ties to even")
		}
	}
	x := new(Int).Lsh(NewInt(1), 1024)
	got, accuracy := new(Float).SetInt(x).Float64()
	if !math.IsInf(got, 1) || accuracy != Above {
		t.Fatal("float overflow")
	}
}
