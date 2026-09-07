package big

import "math"

// Rat is a rational number in reduced form with a positive denominator.
// Its zero value represents zero.
type Rat struct {
	numerator   Int
	denominator Int
}

func (z *Rat) SetInt(x *Int) *Rat    { z.numerator.Set(x); z.denominator.SetInt64(1); return z }
func (z *Rat) SetInt64(x int64) *Rat { return z.SetInt(NewInt(x)) }
func (z *Rat) SetFrac(a, b *Int) *Rat {
	if b.Sign() == 0 {
		panic("division by zero")
	}
	var n, d Int
	n.Set(a)
	d.Set(b)
	if d.Sign() < 0 {
		n.Neg(&n)
		d.Neg(&d)
	}
	var x, y Int
	x.Abs(&n)
	y.Set(&d)
	for y.Sign() != 0 {
		var remainder Int
		remainder.Rem(&x, &y)
		x.Set(&y)
		y.Set(&remainder)
	}
	z.numerator.Quo(&n, &x)
	z.denominator.Quo(&d, &x)
	return z
}
func (z *Rat) SetFloat64(value float64) *Rat {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return nil
	}
	bits := math.Float64bits(value)
	fraction := bits & ((uint64(1) << 52) - 1)
	exponent := int((bits >> 52) & 2047)
	if exponent != 0 {
		fraction |= uint64(1) << 52
		exponent -= 1023 + 52
	} else {
		exponent = -1074
	}
	var n, d Int
	n.SetUint64(fraction)
	d.SetInt64(1)
	if exponent >= 0 {
		n.Lsh(&n, uint(exponent))
	} else {
		d.Lsh(&d, uint(-exponent))
	}
	if bits>>63 != 0 {
		n.Neg(&n)
	}
	return z.SetFrac(&n, &d)
}
func (x *Rat) Num() *Int { return &x.numerator }
func (x *Rat) Denom() *Int {
	if x.denominator.Sign() == 0 {
		x.denominator.SetInt64(1)
	}
	return &x.denominator
}
func (x *Rat) Sign() int { return x.numerator.Sign() }
func (x *Rat) Cmp(y *Rat) int {
	var a, b Int
	a.Mul(&x.numerator, y.Denom())
	b.Mul(&y.numerator, x.Denom())
	return a.Cmp(&b)
}
func (x *Rat) RatString() string {
	denominator := x.Denom()
	if denominator.Cmp(NewInt(1)) == 0 {
		return x.numerator.String()
	}
	return x.numerator.String() + "/" + denominator.String()
}
func (x *Rat) String() string { return x.numerator.String() + "/" + x.Denom().String() }
