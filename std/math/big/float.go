package big

import "math"

type Accuracy int8

const (
	Below Accuracy = -1
	Exact Accuracy = 0
	Above Accuracy = 1
)

func (a Accuracy) String() string {
	if a < 0 {
		return "Below"
	}
	if a > 0 {
		return "Above"
	}
	return "Exact"
}

// Float stores an exact binary mantissa and exponent for integer/binary64
// conversion. Arithmetic and configurable precision are not yet implemented.
type Float struct {
	magnitude Int
	exponent  int
	negative  bool
	infinite  bool
}

func NewFloat(value float64) *Float { return new(Float).SetFloat64(value) }
func (z *Float) SetFloat64(value float64) *Float {
	if math.IsNaN(value) {
		panic("Float.SetFloat64(NaN)")
	}
	bits := math.Float64bits(value)
	fraction := bits & ((uint64(1) << 52) - 1)
	exponent := int((bits >> 52) & 2047)
	z.negative = bits>>63 != 0
	z.infinite = exponent == 2047
	if exponent != 0 {
		fraction |= uint64(1) << 52
		exponent -= 1023 + 52
	} else {
		exponent = -1074
	}
	z.magnitude.SetUint64(fraction)
	z.exponent = exponent
	return z
}
func (z *Float) SetInt(value *Int) *Float {
	z.magnitude.Abs(value)
	z.exponent = 0
	z.negative = value.Sign() < 0
	z.infinite = false
	return z
}
func (x *Float) Int(z *Int) (*Int, Accuracy) {
	if x.infinite {
		if x.negative {
			return nil, Above
		}
		return nil, Below
	}
	if z == nil {
		z = new(Int)
	}
	accuracy := Exact
	if x.exponent >= 0 {
		z.Lsh(&x.magnitude, uint(x.exponent))
	} else {
		z.Rsh(&x.magnitude, uint(-x.exponent))
		var restored Int
		restored.Lsh(z, uint(-x.exponent))
		if restored.Cmp(&x.magnitude) != 0 {
			accuracy = Below
		}
	}
	if x.negative {
		z.Neg(z)
		accuracy = -accuracy
	}
	return z, accuracy
}

// roundedMagnitude returns an integer rounded to nearest, ties to even.
// Callers choose a shift which leaves no more than 53 significant bits.
func roundedMagnitude(magnitude *Int, right int) (uint64, Accuracy) {
	var rounded Int
	if right <= 0 {
		rounded.Lsh(magnitude, uint(-right))
		return rounded.Uint64(), Exact
	}
	rounded.Rsh(magnitude, uint(right))
	var restored, remainder, half Int
	restored.Lsh(&rounded, uint(right))
	remainder.Sub(magnitude, &restored)
	if remainder.Sign() == 0 {
		return rounded.Uint64(), Exact
	}
	half.Lsh(NewInt(1), uint(right-1))
	cmp := remainder.Cmp(&half)
	if cmp > 0 || cmp == 0 && rounded.Uint64()&1 != 0 {
		return rounded.Uint64() + 1, Above
	}
	return rounded.Uint64(), Below
}
func (x *Float) Float64() (float64, Accuracy) {
	sign := uint64(0)
	if x.negative {
		sign = uint64(1) << 63
	}
	if x.infinite {
		return math.Float64frombits(sign | uint64(2047)<<52), Exact
	}
	length := x.magnitude.BitLen()
	if length == 0 {
		return math.Float64frombits(sign), Exact
	}
	exponent := length - 1 + x.exponent
	accuracy := Exact
	var bits uint64
	if exponent > 1023 {
		bits = uint64(2047) << 52
		accuracy = Above
	} else if exponent >= -1022 {
		var significand uint64
		significand, accuracy = roundedMagnitude(&x.magnitude, length-53)
		if significand == uint64(1)<<53 {
			significand >>= 1
			exponent++
		}
		if exponent > 1023 {
			bits = uint64(2047) << 52
			accuracy = Above
		} else {
			bits = uint64(exponent+1023)<<52 | (significand & ((uint64(1) << 52) - 1))
		}
	} else {
		bits, accuracy = roundedMagnitude(&x.magnitude, -1074-x.exponent)
	}
	if x.negative {
		accuracy = -accuracy
	}
	return math.Float64frombits(sign | bits), accuracy
}
