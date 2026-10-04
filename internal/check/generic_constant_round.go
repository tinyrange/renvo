package check

// Round a rational to the target floating-point format, including subnormals
// and ties to even. Keep the result exact for subsequent constant operations.
func genericRoundRational(value genericRational, precision int, minimum int) genericRational {
	if len(value.numerator.words) == 0 {
		return value
	}
	n, d := value.numerator, value.denominator
	n.negative = false
	// This limb estimate is strictly above floor(log2(n/d)). Refine it
	// downward only until the normal/subnormal boundary matters.
	exponent := (len(n.words) - len(d.words) + 1) * 15
	for exponent > minimum+precision-1 {
		left, right := n, d
		if exponent >= 0 {
			right = wideShift(right, exponent, true)
		} else {
			left = wideShift(left, -exponent, true)
		}
		if wideMagnitudeCompare(left, right) >= 0 {
			break
		}
		exponent--
	}
	shift := exponent - precision + 1
	if shift < minimum {
		shift = minimum
	}
	if shift >= 0 {
		d = wideShift(d, shift, true)
	} else {
		n = wideShift(n, -shift, true)
	}
	quotient, remainder := wideDivide(n, d)
	compare := wideMagnitudeCompare(wideShift(remainder, 1, true), d)
	if compare > 0 || compare == 0 && len(quotient.words) > 0 && quotient.words[0]&1 != 0 {
		quotient = wideAdd(quotient, wideSmall(1))
	}
	quotient.negative = value.numerator.negative
	out := genericRationalInteger(quotient)
	if shift >= 0 {
		out.numerator = wideShift(out.numerator, shift, true)
	} else {
		out.denominator = wideShift(out.denominator, -shift, true)
	}
	return out
}

func (e *genericEnvironment) roundConstant(value *genericConstant, typ int) *genericConstant {
	if value == nil || value.text != nil || typ == 0 {
		return value
	}
	name := e.types.get(e.types.underlying(typ)).name
	precision, minimum := 53, -1074
	if name == "float32" || name == "complex64" {
		precision, minimum = 24, -149
	} else if name != "float64" && name != "complex128" {
		return value
	}
	return &genericConstant{real: genericRoundRational(value.real, precision, minimum), imaginary: genericRoundRational(value.imaginary, precision, minimum)}
}
