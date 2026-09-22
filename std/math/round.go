package math

// Trunc returns the integer part of x, preserving signed zero and infinities.
func Trunc(x float64) float64 {
	b := Float64bits(x)
	e := int(b>>52&0x7ff) - 1023
	if e < 0 {
		return Copysign(0, x)
	}
	if e >= 52 {
		return x
	}
	mask := uint64(1)<<uint(52-e) - 1
	return Float64frombits(b &^ mask)
}

// Floor returns the greatest integer value no larger than x.
func Floor(x float64) float64 {
	y := Trunc(x)
	if y > x {
		return y - 1
	}
	return y
}

// Mod returns the remainder of x/y, with the sign of x. Integer significand
// division avoids overflowing a floating-point quotient or losing low bits.
func Mod(x, y float64) float64 {
	if y == 0 || IsInf(x, 0) || IsNaN(x) || IsNaN(y) {
		return NaN()
	}
	ax, ay := Abs(x), Abs(y)
	if ax < ay || x == 0 || IsInf(y, 0) {
		return x
	}
	if ax == ay {
		return Copysign(0, x)
	}
	xm, xe := modSignificand(ax)
	ym, ye := modSignificand(ay)
	for xe > ye {
		if xm >= ym {
			xm -= ym
		}
		if xm == 0 {
			return Copysign(0, x)
		}
		xm <<= 1
		xe--
	}
	if xm >= ym {
		xm -= ym
	}
	if xm == 0 {
		return Copysign(0, x)
	}
	for xm < uint64(1)<<52 {
		xm <<= 1
		xe--
	}
	var bits uint64
	if xe >= -1022 {
		bits = uint64(xe+1023)<<52 | (xm & (uint64(1)<<52 - 1))
	} else {
		bits = xm >> uint(-1022-xe)
	}
	return Copysign(Float64frombits(bits), x)
}

// modSignificand decomposes a positive finite nonzero value as m*2^(e-52),
// normalizing subnormal inputs so that bit 52 of m is always set.
func modSignificand(x float64) (uint64, int) {
	b := Float64bits(x)
	e := int(b>>52) - 1023
	m := b & (uint64(1)<<52 - 1)
	if e == -1023 {
		e = -1022
		for m < uint64(1)<<52 {
			m <<= 1
			e--
		}
	} else {
		m |= uint64(1) << 52
	}
	return m, e
}
