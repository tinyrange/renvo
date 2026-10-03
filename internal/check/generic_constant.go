package check

// Exact rationals keep declaration checking independent of the first concrete
// instantiation, including constants that fit one constraint term but not all.
type genericRational struct{ numerator, denominator wideConstant }
type genericConstant struct {
	real, imaginary genericRational
	text            *string
	boolean         *bool
}

func genericBooleanConstant(value bool) *genericConstant {
	return &genericConstant{boolean: &value}
}

func genericConstantComparison(a *genericConstant, b *genericConstant, op string) *genericConstant {
	if a == nil || b == nil {
		return nil
	}
	compare, equal := 0, false
	if a.boolean != nil && b.boolean != nil {
		equal = *a.boolean == *b.boolean
	} else if a.text != nil && b.text != nil {
		equal = *a.text == *b.text
		if *a.text < *b.text {
			compare = -1
		} else if *a.text > *b.text {
			compare = 1
		}
	} else if a.boolean == nil && b.boolean == nil && a.text == nil && b.text == nil {
		real := genericRationalAdd(a.real, genericRationalNegate(b.real))
		imaginary := genericRationalAdd(a.imaginary, genericRationalNegate(b.imaginary))
		if !real.numerator.ok || !imaginary.numerator.ok {
			return nil
		}
		equal = len(real.numerator.words) == 0 && len(imaginary.numerator.words) == 0
		if len(real.numerator.words) != 0 {
			compare = 1
			if real.numerator.negative {
				compare = -1
			}
		}
	} else {
		return nil
	}
	result := equal
	switch op {
	case "!=":
		result = !equal
	case "<":
		result = compare < 0
	case "<=":
		result = compare <= 0
	case ">":
		result = compare > 0
	case ">=":
		result = compare >= 0
	}
	return genericBooleanConstant(result)
}

func genericRationalInteger(n wideConstant) genericRational {
	return genericRational{numerator: n, denominator: wideSmall(1)}
}

func genericRationalAdd(a genericRational, b genericRational) genericRational {
	return genericRational{numerator: wideAdd(wideMultiply(a.numerator, b.denominator), wideMultiply(b.numerator, a.denominator)), denominator: wideMultiply(a.denominator, b.denominator)}
}

func genericRationalNegate(a genericRational) genericRational {
	a.numerator = wideNegate(a.numerator)
	return a
}

func genericRationalMultiply(a genericRational, b genericRational) genericRational {
	return genericRational{numerator: wideMultiply(a.numerator, b.numerator), denominator: wideMultiply(a.denominator, b.denominator)}
}

func genericRationalDivide(a genericRational, b genericRational) genericRational {
	if len(b.numerator.words) == 0 {
		return genericRational{}
	}
	n := wideMultiply(a.numerator, b.denominator)
	d := wideMultiply(a.denominator, b.numerator)
	if d.negative {
		n = wideNegate(n)
		d = wideNegate(d)
	}
	return genericRational{numerator: n, denominator: d}
}

func genericConstantLiteral(text string) *genericConstant {
	imaginary := false
	if len(text) > 0 && text[len(text)-1] == 'i' {
		imaginary = true
		text = text[:len(text)-1]
	}
	clean := ""
	for i := 0; i < len(text); i++ {
		if text[i] != '_' {
			clean += text[i : i+1]
		}
	}
	text = clean
	base, expBase := 10, 10
	start := 0
	if len(text) > 2 && text[0] == '0' && (text[1] == 'x' || text[1] == 'X') {
		base, expBase, start = 16, 2, 2
	}
	point, exponent := -1, -1
	for i := start; i < len(text); i++ {
		if text[i] == '.' {
			point = i
		}
		if base == 16 && (text[i] == 'p' || text[i] == 'P') || base == 10 && (text[i] == 'e' || text[i] == 'E') {
			exponent = i
			break
		}
	}
	n := wideConstant{}
	d := wideSmall(1)
	if point < 0 && exponent < 0 {
		if imaginary && len(text) > 1 && text[0] == '0' && text[1] >= '0' && text[1] <= '9' {
			for len(text) > 1 && text[0] == '0' {
				text = text[1:]
			}
		}
		n = wideIntegerLiteral(text)
	} else {
		end := len(text)
		if exponent >= 0 {
			end = exponent
		}
		digits := ""
		for i := start; i < end; i++ {
			if text[i] != '.' {
				digits += text[i : i+1]
			}
		}
		if base == 16 {
			digits = "0x" + digits
		}
		if base == 10 {
			for len(digits) > 1 && digits[0] == '0' {
				digits = digits[1:]
			}
		}
		n = wideIntegerLiteral(digits)
		power := 0
		if exponent >= 0 {
			pos := exponent + 1
			negative := false
			if pos < len(text) && (text[pos] == '+' || text[pos] == '-') {
				negative = text[pos] == '-'
				pos++
			}
			for ; pos < len(text); pos++ {
				if text[pos] < '0' || text[pos] > '9' || power > 20000 {
					return nil
				}
				power = power*10 + int(text[pos]-'0')
			}
			if negative {
				power = -power
			}
		}
		if point >= 0 {
			fraction := end - point - 1
			if base == 16 {
				fraction *= 4
			}
			power -= fraction
		}
		negative := power < 0
		if negative {
			power = -power
		}
		factor := wideSmall(1)
		step := wideSmall(expBase)
		for power > 0 {
			if power&1 != 0 {
				factor = wideMultiply(factor, step)
			}
			power /= 2
			if power > 0 {
				step = wideMultiply(step, step)
			}
			if len(factor.words) > 16384 || len(step.words) > 16384 {
				return nil
			}
		}
		if negative {
			d = factor
		} else {
			n = wideMultiply(n, factor)
		}
	}
	if !n.ok {
		return nil
	}
	value := &genericConstant{real: genericRationalInteger(wideSmall(0)), imaginary: genericRationalInteger(wideSmall(0))}
	rational := genericRational{numerator: n, denominator: d}
	if imaginary {
		value.imaginary = rational
	} else {
		value.real = rational
	}
	return value
}

func genericConstantInteger(value *genericConstant) wideConstant {
	if value == nil || value.text != nil || value.boolean != nil || len(value.imaginary.numerator.words) != 0 {
		return wideConstant{}
	}
	n, remainder := wideDivide(value.real.numerator, value.real.denominator)
	if len(remainder.words) != 0 {
		return wideConstant{}
	}
	return n
}

func genericConstantBinary(a *genericConstant, b *genericConstant, op string, integer bool) *genericConstant {
	if a == nil || b == nil {
		return nil
	}
	if a.boolean != nil || b.boolean != nil {
		if a.boolean == nil || b.boolean == nil {
			return nil
		}
		if op == "&&" {
			return genericBooleanConstant(*a.boolean && *b.boolean)
		}
		if op == "||" {
			return genericBooleanConstant(*a.boolean || *b.boolean)
		}
		return nil
	}
	if a.text != nil || b.text != nil {
		if a.text != nil && b.text != nil && op == "+" {
			text := *a.text + *b.text
			return &genericConstant{text: &text}
		}
		return nil
	}
	out := &genericConstant{}
	switch op {
	case "+":
		out.real = genericRationalAdd(a.real, b.real)
		out.imaginary = genericRationalAdd(a.imaginary, b.imaginary)
	case "-":
		out.real = genericRationalAdd(a.real, genericRationalNegate(b.real))
		out.imaginary = genericRationalAdd(a.imaginary, genericRationalNegate(b.imaginary))
	case "*":
		out.real = genericRationalAdd(genericRationalMultiply(a.real, b.real), genericRationalNegate(genericRationalMultiply(a.imaginary, b.imaginary)))
		out.imaginary = genericRationalAdd(genericRationalMultiply(a.real, b.imaginary), genericRationalMultiply(a.imaginary, b.real))
	case "/":
		den := genericRationalAdd(genericRationalMultiply(b.real, b.real), genericRationalMultiply(b.imaginary, b.imaginary))
		out.real = genericRationalDivide(genericRationalAdd(genericRationalMultiply(a.real, b.real), genericRationalMultiply(a.imaginary, b.imaginary)), den)
		out.imaginary = genericRationalDivide(genericRationalAdd(genericRationalMultiply(a.imaginary, b.real), genericRationalNegate(genericRationalMultiply(a.real, b.imaginary))), den)
		if integer {
			q, _ := wideDivide(out.real.numerator, out.real.denominator)
			out.real = genericRationalInteger(q)
		}
	default:
		left, right := genericConstantInteger(a), genericConstantInteger(b)
		if !left.ok || !right.ok {
			return nil
		}
		n := wideConstant{}
		if op == "%" {
			_, n = wideDivide(left, right)
		}
		if op == "&" || op == "|" || op == "^" || op == "&^" {
			n = wideBitwise(left, right, op)
		}
		if op == "<<" || op == ">>" {
			if shift, ok := wideInt(right); ok && shift >= 0 && shift < 262144 {
				n = wideShift(left, shift, op == "<<")
			}
		}
		out.real = genericRationalInteger(n)
		out.imaginary = genericRationalInteger(wideSmall(0))
	}
	if !out.real.numerator.ok || !out.imaginary.numerator.ok {
		return nil
	}
	return out
}

func genericIntegerBits(name string, wordBits int, pointerBits int) int {
	if wordBits == 0 {
		wordBits = 64
	}
	if pointerBits == 0 {
		pointerBits = wordBits
	}
	if name == "int" || name == "uint" {
		return wordBits
	}
	if name == "uintptr" {
		return pointerBits
	}
	if name == "int8" || name == "uint8" {
		return 8
	}
	if name == "int16" || name == "uint16" {
		return 16
	}
	if name == "int32" || name == "uint32" {
		return 32
	}
	return 64
}

func genericRationalFits(value genericRational, name string, wordBits int, pointerBits int) bool {
	if !value.numerator.ok || !value.denominator.ok {
		return false
	}
	if genericInteger(name) {
		integer, remainder := wideDivide(value.numerator, value.denominator)
		if !integer.ok || len(remainder.words) != 0 {
			return false
		}
		bits := genericIntegerBits(name, wordBits, pointerBits)
		unsigned := name[0] == 'u'
		if unsigned && integer.negative {
			return false
		}
		if !unsigned {
			bits--
		}
		limit := wideShift(wideSmall(1), bits, true)
		cmp := wideMagnitudeCompare(integer, limit)
		return cmp < 0 || !unsigned && integer.negative && cmp == 0
	}
	precision, exponent := 53, 1024
	if name == "float32" || name == "complex64" {
		precision, exponent = 24, 128
	}
	threshold := wideShift(wideAdd(wideShift(wideSmall(1), precision+1, true), wideSmall(-1)), exponent-precision-1, true)
	return wideMagnitudeCompare(value.numerator, wideMultiply(threshold, value.denominator)) < 0
}

func (e *genericEnvironment) constantFits(value genericArgument, target int) bool {
	if value.constant == nil {
		return true
	}
	v := e.types.get(e.types.underlying(target))
	if v.kind == genericInterface {
		return e.constantFits(value, e.types.defaultType(value.untyped))
	}
	if value.constant.text != nil {
		return v.kind == genericBasic && v.name == "string"
	}
	if value.constant.boolean != nil {
		return v.kind == genericBasic && v.name == "bool"
	}
	if v.kind != genericBasic || !genericNumeric(v.name) {
		return false
	}
	if v.name != "complex64" && v.name != "complex128" && len(value.constant.imaginary.numerator.words) != 0 {
		return false
	}
	return genericRationalFits(value.constant.real, v.name, e.wordBits, e.pointerBits) && genericRationalFits(value.constant.imaginary, v.name, e.wordBits, e.pointerBits)
}
