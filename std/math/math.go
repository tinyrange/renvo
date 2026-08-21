// Package math provides basic constants and mathematical functions for
// float64 values. It is a portable pure-Go subset of the standard library's
// math package: every function is implemented without platform intrinsics,
// so results are identical on every target. Functions agree with the host
// math package to within a few ULPs; they do not guarantee bit-exact
// equality for transcendental functions.
package math

// Mathematical constants.
const (
	E      = 2.71828182845904523536028747135266249775724709369995957496696763
	Pi     = 3.14159265358979323846264338327950288419716939937510582097494459
	Phi    = 1.61803398874989484820458683436563811772030917980576286213544862
	Sqrt2  = 1.41421356237309504880168872420969807856967187537694807317667974
	SqrtE  = 1.64872127070012814684865078781416357316561887741369087575252939
	SqrtPi = 1.77245385090551602729816748334114518279754945612238712821380779

	Ln2    = 0.693147180559945309417232121458176568075500134360255254120680009
	Log2E  = 1 / Ln2
	Ln10   = 2.30258509299404568401799145468436420760110148862877297603332790
	Log10E = 1 / Ln10

	MaxFloat64             = 1.797693134862315708145274237317043567981e+308
	SmallestNonzeroFloat64 = 4.940656458412465441765687928682213723651e-324

	MaxInt8   = 1<<7 - 1
	MinInt8   = -1 << 7
	MaxInt16  = 1<<15 - 1
	MinInt16  = -1 << 15
	MaxInt32  = 1<<31 - 1
	MinInt32  = -1 << 31
	MaxInt64  = 1<<63 - 1
	MinInt64  = -1 << 63
	MaxUint8  = 1<<8 - 1
	MaxUint16 = 1<<16 - 1
	MaxUint32 = 1<<32 - 1
	MaxUint64 = 1<<64 - 1

	MaxInt  = MaxInt64
	MinInt  = -MaxInt - 1
	MaxUint = MaxUint64
)

// Inf returns positive infinity if sign >= 0, negative infinity otherwise.
func Inf(sign int) float64 {
	var big float64 = MaxFloat64
	if sign >= 0 {
		return big * 10
	}
	return -big * 10
}

// NaN returns an IEEE 754 not-a-number value.
func NaN() float64 {
	var zero float64
	return zero / zero
}

// IsNaN reports whether f is a NaN value.
func IsNaN(f float64) bool { return f != f }

// IsInf reports whether f is an infinity. If sign > 0 it reports whether f
// is positive infinity; if sign < 0, negative infinity; if sign == 0,
// either infinity.
func IsInf(f float64, sign int) bool {
	if sign > 0 {
		return f > MaxFloat64
	}
	if sign < 0 {
		return f < -MaxFloat64
	}
	return f > MaxFloat64 || f < -MaxFloat64
}

// Signbit reports whether x is negative or negative zero. The sign of a NaN
// payload is not representable without bit access, so Signbit reports false
// for any NaN.
func Signbit(x float64) bool {
	if x != x {
		return false
	}
	if x != 0 && !IsInf(x, 0) {
		return x < 0
	}
	var one float64 = 1
	return one/x < 0
}

// Abs returns the absolute value of x.
func Abs(x float64) float64 {
	if x < 0 || Signbit(x) {
		return -x
	}
	return x
}

// Copysign returns a value with the magnitude of x and the sign of y.
func Copysign(x, y float64) float64 {
	if Signbit(y) {
		return -Abs(x)
	}
	return Abs(x)
}

// Mod returns the floating-point remainder of x/y. The result has the sign
// of x and magnitude less than Abs(y).
func Mod(x, y float64) float64 {
	if y == 0 || IsNaN(x) || IsNaN(y) || IsInf(x, 0) {
		return NaN()
	}
	if IsInf(y, 0) {
		return x
	}
	sign := 1.0
	if x < 0 {
		sign = -1
	}
	ax := Abs(x)
	ay := Abs(y)
	if ax < ay {
		return x
	}
	if ax == ay {
		return sign * 0
	}
	// Scale ay up by powers of two until it exceeds ax, then walk back down
	// subtracting the largest aligned multiple at each scale. This is O(log
	// of the quotient) instead of long division by repeated subtraction.
	scale := 1.0
	for ay*scale <= ax {
		scale *= 2
	}
	r := ax
	for scale > 1 {
		scale /= 2
		if r >= ay*scale {
			r -= ay * scale
		}
	}
	if r == 0 {
		return sign * 0
	}
	return sign * r
}

// Trunc returns the integer part of x. Zero results keep the sign of x.
func Trunc(x float64) float64 {
	if IsNaN(x) || IsInf(x, 0) {
		return x
	}
	if x < 1 && x > -1 {
		return 0 * x
	}
	return x - Mod(x, 1)
}

// Floor returns the greatest integer value less than or equal to x.
func Floor(x float64) float64 {
	t := Trunc(x)
	if t > x {
		t -= 1
	}
	return t
}

// Ceil returns the least integer value greater than or equal to x.
func Ceil(x float64) float64 {
	t := Trunc(x)
	if t < x {
		t += 1
	}
	return t
}

// Round rounds x to the nearest integer, with halfway cases away from zero.
func Round(x float64) float64 {
	if IsNaN(x) || IsInf(x, 0) {
		return x
	}
	t := Trunc(x)
	frac := x - t
	if frac >= 0.5 {
		return t + 1
	}
	if frac <= -0.5 {
		return t - 1
	}
	return t
}

// RoundToEven rounds x to the nearest integer, with halfway cases to the
// even integer.
func RoundToEven(x float64) float64 {
	if IsNaN(x) || IsInf(x, 0) {
		return x
	}
	t := Trunc(x)
	frac := x - t
	if frac > 0.5 {
		return t + 1
	}
	if frac < -0.5 {
		return t - 1
	}
	if frac == 0.5 {
		if Mod(t, 2) == 0 {
			return t
		}
		return t + 1
	}
	if frac == -0.5 {
		if Mod(t, 2) == 0 {
			return t
		}
		return t - 1
	}
	return t
}

// Dim returns the maximum of x-y and 0.
func Dim(x, y float64) float64 {
	d := x - y
	if d <= 0 {
		return 0
	}
	return d
}

// Max returns the larger of x or y. NaN arguments make the result NaN. Max
// of two zeros is +0.
func Max(x, y float64) float64 {
	if IsNaN(x) || IsNaN(y) {
		return NaN()
	}
	if x > y {
		return x
	}
	if y > x {
		return y
	}
	if x == 0 && y == 0 && Signbit(x) {
		return y
	}
	return x
}

// Min returns the smaller of x or y. NaN arguments make the result NaN. Min
// of two zeros is -0.
func Min(x, y float64) float64 {
	if IsNaN(x) || IsNaN(y) {
		return NaN()
	}
	if x < y {
		return x
	}
	if y < x {
		return y
	}
	if x == 0 && y == 0 && !Signbit(x) {
		return y
	}
	return x
}

// Frexp splits f into a normalized fraction in [0.5, 1) and an exponent so
// that f == frac * 2**exp.
func Frexp(f float64) (float64, int) {
	if f == 0 || IsNaN(f) || IsInf(f, 0) {
		return f, 0
	}
	exp := 0
	frac := f
	for frac >= 1 {
		frac /= 2
		exp++
	}
	for frac < 0.5 {
		frac *= 2
		exp--
	}
	return frac, exp
}

// Ldexp returns frac * 2**exp.
func Ldexp(frac float64, exp int) float64 {
	if frac == 0 || IsNaN(frac) || IsInf(frac, 0) {
		return frac
	}
	for exp > 0 {
		frac *= 2
		exp--
		if IsInf(frac, 0) {
			return frac
		}
	}
	for exp < 0 {
		frac /= 2
		exp++
	}
	return frac
}

// Sqrt returns the square root of x. Negative inputs are NaN.
func Sqrt(x float64) float64 {
	if IsNaN(x) || x < 0 {
		return NaN()
	}
	if x == 0 || IsInf(x, 1) {
		return x
	}
	// Normalize to m in [1, 4) so x == m * 4**e, then refine a Newton
	// iteration. Multiplication and division by four are exact.
	e := 0
	m := x
	for m >= 4 {
		m /= 4
		e++
	}
	for m < 1 {
		m *= 4
		e--
	}
	g := m*0.5 + 0.5
	for i := 0; i < 9; i++ {
		g = (g + m/g) / 2
	}
	for i := 0; i < e; i++ {
		g *= 2
	}
	for i := 0; i < -e; i++ {
		g /= 2
	}
	return g
}

// cbrtTwoToThird[r+2] is 2**(r/3) for r in [-2, 2].
var cbrtTwoToThird = [5]float64{
	0.6299605249474365823836053036391141752849279638408548493400,
	0.7937005259840997373758528196361541301957466638995353302832,
	1.0,
	1.2599210498948731647672106072782283505701814684539803202333,
	1.587401051,
}

// Cbrt returns the cube root of x; negative inputs get a negative result.
func Cbrt(x float64) float64 {
	if x == 0 || IsNaN(x) || IsInf(x, 0) {
		return x
	}
	neg := x < 0
	if neg {
		x = -x
	}
	// Normalize to m in [1, 2) with x == m * 2**e, take the cube root of m
	// by Newton iteration, then reapply 2**(e/3).
	e := 0
	m := x
	for m >= 2 {
		m /= 2
		e++
	}
	for m < 1 {
		m *= 2
		e--
	}
	g := 1 + (m-1)*0.35
	for i := 0; i < 12; i++ {
		g = (2*g + m/(g*g)) / 3
	}
	k := e / 3
	g *= cbrtTwoToThird[e-3*k+2]
	for i := 0; i < k; i++ {
		g *= 2
	}
	for i := 0; i < -k; i++ {
		g /= 2
	}
	if neg {
		return -g
	}
	return g
}

// Hypot returns Sqrt(p*p + q*q) without unnecessary overflow or underflow.
func Hypot(p, q float64) float64 {
	if IsInf(p, 0) || IsInf(q, 0) {
		return Inf(1)
	}
	if p < 0 {
		p = -p
	}
	if q < 0 {
		q = -q
	}
	if p < q {
		p, q = q, p
	}
	if p == 0 {
		return 0
	}
	r := q / p
	return p * Sqrt(1+r*r)
}

const ln2Half = 0.34657359027997264

// Exp returns e**x. Results overflow to Inf past about 709.78 and underflow
// to zero below about -745.13, matching IEEE 754 double limits.
func Exp(x float64) float64 {
	if IsNaN(x) {
		return x
	}
	if x > 709.782712893384 {
		return Inf(1)
	}
	if x < -745.1332191019411 {
		return 0
	}
	if x == 0 {
		return 1
	}
	if IsInf(x, -1) {
		return 0
	}
	// Range-reduce so r = x - k*Ln2 with |r| <= Ln2/2, evaluate the Taylor
	// series for e**r, then rescale by 2**k.
	k := int(x / Ln2)
	if x > 0 && x-float64(k)*Ln2 > ln2Half {
		k++
	}
	if x < 0 && x-float64(k)*Ln2 < -ln2Half {
		k--
	}
	r := x - float64(k)*Ln2
	sum := 1.0
	term := 1.0
	for i := 1; i <= 16; i++ {
		term *= r / float64(i)
		sum += term
	}
	for i := 0; i < k; i++ {
		sum *= 2
	}
	for i := 0; i < -k; i++ {
		sum /= 2
	}
	return sum
}

// Exp2 returns 2**x.
func Exp2(x float64) float64 {
	if IsNaN(x) {
		return x
	}
	if x >= 1024 {
		return Inf(1)
	}
	if x <= -1075 {
		return 0
	}
	return Exp(x * Ln2)
}

// Log returns the natural logarithm of x. Negative inputs and NaN are NaN;
// zero approaches negative infinity.
func Log(x float64) float64 {
	if IsNaN(x) || x < 0 {
		return NaN()
	}
	if x == 0 {
		return Inf(-1)
	}
	if IsInf(x, 1) {
		return Inf(1)
	}
	// Normalize to m in [1, 2) with x == m * 2**e, then use the atanh
	// series ln(m) = 2*atanh((m-1)/(m+1)), which converges quickly because
	// the argument is at most 1/3.
	e := 0
	m := x
	for m >= 2 {
		m /= 2
		e++
	}
	for m < 1 {
		m *= 2
		e--
	}
	s := (m - 1) / (m + 1)
	s2 := s * s
	sum := 0.0
	pow := s
	for i := 1; i <= 24; i++ {
		sum += pow / float64(2*i-1)
		pow *= s2
	}
	return 2*sum + float64(e)*Ln2
}

// Log2 returns the binary logarithm of x.
func Log2(x float64) float64 { return Log(x) / Ln2 }

// Log10 returns the decimal logarithm of x.
func Log10(x float64) float64 { return Log(x) / Ln10 }

// oddIntegral reports whether y is an integral value congruent to 1 mod 2.
func oddIntegral(y float64) bool {
	if y != Trunc(y) {
		return false
	}
	return Mod(Abs(y), 2) == 1
}

func ipow(base float64, n int) float64 {
	result := 1.0
	for n > 0 {
		if n&1 == 1 {
			result *= base
		}
		base *= base
		n >>= 1
	}
	return result
}

// powSpecial handles the IEEE special cases of Pow. It reports whether the
// result is final.
func powSpecial(x, y float64) (float64, bool) {
	if y == 0 || x == 1 {
		return 1, true
	}
	if IsNaN(x) || IsNaN(y) {
		return NaN(), true
	}
	if x == 0 {
		if y < 0 {
			if oddIntegral(y) && Signbit(x) {
				return Inf(-1), true
			}
			return Inf(1), true
		}
		if oddIntegral(y) {
			return x, true
		}
		return 0, true
	}
	if IsInf(y, 1) {
		if x > 1 || x < -1 {
			return Inf(1), true
		}
		if x > -1 && x < 1 {
			return 0, true
		}
		return NaN(), true
	}
	if IsInf(y, -1) {
		if x > 1 || x < -1 {
			return 0, true
		}
		if x > -1 && x < 1 {
			return Inf(1), true
		}
		return NaN(), true
	}
	if IsInf(x, 1) {
		if y > 0 {
			return Inf(1), true
		}
		return 0, true
	}
	if IsInf(x, -1) {
		if y > 0 {
			if oddIntegral(y) {
				return Inf(-1), true
			}
			return Inf(1), true
		}
		if oddIntegral(y) {
			return Copysign(0, -1), true
		}
		return 0, true
	}
	return 0, false
}

// powFinite computes x**y for finite non-zero x and y.
func powFinite(x, y float64) float64 {
	neg := false
	base := x
	if x < 0 {
		if y != Trunc(y) {
			return NaN()
		}
		base = -x
		neg = oddIntegral(y)
	}
	var result float64
	if y == Trunc(y) && Abs(y) <= 1024 {
		result = ipow(base, int(Abs(y)))
		if y < 0 {
			result = 1 / result
		}
	} else {
		result = Exp(y * Log(base))
	}
	if neg {
		result = -result
	}
	return result
}

// Pow returns x**y following Go's special-case rules: Pow(x, 0) and
// Pow(1, y) are 1; Pow(0, negative) is an infinity; fractional exponents on
// negative bases are NaN.
func Pow(x, y float64) float64 {
	special, ok := powSpecial(x, y)
	if ok {
		return special
	}
	return powFinite(x, y)
}
