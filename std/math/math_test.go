package math

import "testing"

func TestIEEEHelpers(t *testing.T) {
	if Float32bits(Float32frombits(0x7fc00042)) != 0x7fc00042 {
		t.Fatal("float32 payload did not round trip")
	}
	if Float64bits(Float64frombits(0x7ff8000000000042)) != 0x7ff8000000000042 {
		t.Fatal("float64 payload did not round trip")
	}
	if !IsNaN(NaN()) || !IsInf(Inf(1), 1) || !IsInf(Inf(-1), -1) {
		t.Fatal("special-value classification failed")
	}
	if Signbit(Abs(Inf(-1))) || !Signbit(Copysign(1, Inf(-1))) {
		t.Fatal("sign operations failed")
	}
}
