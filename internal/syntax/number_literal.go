package syntax

// Validate the complete numeric token without converting its value to a host
// number. Literal size and target representability are separate checks.
func validNumberLiteral(src []byte, start, end int) bool {
	if start < 0 || start >= end || end > len(src) {
		return false
	}
	imaginary := src[end-1] == 'i'
	if imaginary {
		end--
	}
	if start >= end {
		return false
	}
	base, pos := 10, start
	prefix := false
	if end-start >= 2 && src[start] == '0' {
		if src[start+1] == 'x' || src[start+1] == 'X' {
			base = 16
			prefix = true
		}
		if src[start+1] == 'b' || src[start+1] == 'B' {
			base = 2
			prefix = true
		}
		if src[start+1] == 'o' || src[start+1] == 'O' {
			base = 8
			prefix = true
		}
		if prefix {
			pos += 2
		}
	}
	pos, digits, ok := numberDigitRun(src, pos, end, base, prefix)
	if !ok {
		return false
	}
	dot := pos < end && src[pos] == '.'
	if dot {
		if base != 10 && base != 16 {
			return false
		}
		var after int
		pos, after, ok = numberDigitRun(src, pos+1, end, base, false)
		if !ok {
			return false
		}
		digits += after
	}
	if digits == 0 {
		return false
	}
	exponent := false
	if pos < end && (base == 10 && (src[pos] == 'e' || src[pos] == 'E') || base == 16 && (src[pos] == 'p' || src[pos] == 'P')) {
		exponent = true
		pos++
		if pos < end && (src[pos] == '+' || src[pos] == '-') {
			pos++
		}
		pos, digits, ok = numberDigitRun(src, pos, end, 10, false)
		if !ok || digits == 0 {
			return false
		}
	}
	if pos != end || base == 16 && dot && !exponent {
		return false
	}
	// Legacy leading-zero integers are octal, but decimal imaginary literals
	// (including 09i) and decimal floating literals use decimal digits.
	if !prefix && !dot && !exponent && !imaginary && src[start] == '0' {
		_, _, ok = numberDigitRun(src, start, end, 8, false)
		return ok
	}
	return true
}

func numberDigitRun(src []byte, start, end, base int, prefix bool) (int, int, bool) {
	pos, count := start, 0
	previousDigit := false
	if prefix && pos < end && src[pos] == '_' {
		pos++
	}
	for pos < end {
		c := src[pos]
		if c == '_' {
			if !previousDigit {
				return pos, count, false
			}
			previousDigit = false
			pos++
			continue
		}
		digit, ok := hexValue(c)
		if !ok || base != 16 && (c < '0' || c > '9') {
			break
		}
		if digit >= base {
			return pos, count, false
		}
		previousDigit = true
		count++
		pos++
	}
	if pos > start && !previousDigit {
		return pos, count, false
	}
	return pos, count, true
}
