package strconv

func ParseFloat(s string, bitSize int) (float64, error) {
	original := s
	if s == "" {
		return 0, numError("ParseFloat", original, ErrSyntax)
	}
	neg := false
	if s[0] == '-' || s[0] == '+' {
		neg = s[0] == '-'
		s = s[1:]
	}
	if s == "NaN" && !neg {
		return floatNaN(), nil
	}
	if s == "Inf" || s == "Infinity" {
		value := floatInf()
		if neg {
			value = -value
		}
		return value, nil
	}
	if len(s) > 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		value, err := parseHexFloat(s)
		if err != nil {
			return 0, numError("ParseFloat", original, ErrSyntax)
		}
		if neg {
			value = -value
		}
		if bitSize == 32 {
			value = float64(float32(value))
		}
		return value, nil
	}
	value := float64(0)
	digits := 0
	for len(s) > 0 && s[0] >= '0' && s[0] <= '9' {
		value = value*10 + float64(s[0]-'0')
		s = s[1:]
		digits++
	}
	if len(s) > 0 && s[0] == '.' {
		s = s[1:]
		scale := float64(1)
		for len(s) > 0 && s[0] >= '0' && s[0] <= '9' {
			scale *= 10
			value += float64(s[0]-'0') / scale
			s = s[1:]
			digits++
		}
	}
	if digits == 0 {
		return 0, numError("ParseFloat", original, ErrSyntax)
	}
	exp := 0
	expNeg := false
	if len(s) > 0 && (s[0] == 'e' || s[0] == 'E') {
		s = s[1:]
		if len(s) > 0 && (s[0] == '-' || s[0] == '+') {
			expNeg = s[0] == '-'
			s = s[1:]
		}
		if s == "" {
			return 0, numError("ParseFloat", original, ErrSyntax)
		}
		count := 0
		for len(s) > 0 && s[0] >= '0' && s[0] <= '9' {
			if exp < 10000 {
				exp = exp*10 + int(s[0]-'0')
			}
			s = s[1:]
			count++
		}
		if count == 0 {
			return 0, numError("ParseFloat", original, ErrSyntax)
		}
	}
	if s != "" {
		return 0, numError("ParseFloat", original, ErrSyntax)
	}
	for exp > 0 {
		if expNeg {
			value /= 10
		} else {
			value *= 10
		}
		exp--
	}
	if neg {
		value = -value
	}
	if bitSize == 32 {
		value = float64(float32(value))
	}
	return value, nil
}

func floatInf() float64 { zero := float64(0); return 1 / zero }
func floatNaN() float64 { zero := float64(0); return zero / zero }
func parseHexFloat(s string) (float64, error) {
	s = s[2:]
	value := float64(0)
	fractionDigits := 0
	seenDot := false
	digits := 0
	for len(s) > 0 && s[0] != 'p' && s[0] != 'P' {
		if s[0] == '.' {
			if seenDot {
				return 0, ErrSyntax
			}
			seenDot = true
			s = s[1:]
			continue
		}
		digit, ok := digitValue(s[0])
		if !ok || digit >= 16 {
			return 0, ErrSyntax
		}
		value = value*16 + float64(digit)
		if seenDot {
			fractionDigits++
		}
		digits++
		s = s[1:]
	}
	if digits == 0 || len(s) == 0 {
		return 0, ErrSyntax
	}
	s = s[1:]
	neg := false
	if len(s) > 0 && (s[0] == '+' || s[0] == '-') {
		neg = s[0] == '-'
		s = s[1:]
	}
	if s == "" {
		return 0, ErrSyntax
	}
	exp := 0
	for len(s) > 0 {
		if s[0] < '0' || s[0] > '9' {
			return 0, ErrSyntax
		}
		exp = exp*10 + int(s[0]-'0')
		s = s[1:]
	}
	if neg {
		exp = -exp
	}
	exp -= fractionDigits * 4
	for exp > 0 {
		value *= 2
		exp--
	}
	for exp < 0 {
		value /= 2
		exp++
	}
	return value, nil
}

func FormatFloat(value float64, format byte, precision int, bitSize int) string {
	if value != value {
		return "NaN"
	}
	inf := floatInf()
	if value == inf {
		return "+Inf"
	}
	if value == -inf {
		return "-Inf"
	}
	if bitSize == 32 {
		value = float64(float32(value))
	}
	neg := value < 0
	if neg {
		value = -value
	}
	if precision < 0 {
		precision = 9
		if bitSize == 64 {
			precision = 17
		}
	}
	if precision > 18 {
		precision = 18
	}
	if format == 'e' || format == 'E' {
		return formatFloatExponent(value, format, precision, neg)
	}
	if format != 'f' && format != 'g' && format != 'G' {
		format = 'g'
	}
	text := formatFloatFixed(value, precision)
	if format == 'g' || format == 'G' {
		text = trimFloatZeros(text)
	}
	if neg {
		return "-" + text
	}
	return text
}
func formatFloatFixed(value float64, precision int) string {
	whole := uint64(value)
	fraction := value - float64(whole)
	scale := uint64(1)
	for i := 0; i < precision; i++ {
		scale *= 10
	}
	rounded := uint64(fraction*float64(scale) + 0.5)
	if rounded >= scale && scale > 0 {
		whole++
		rounded -= scale
	}
	text := FormatUint(whole, 10)
	if precision == 0 {
		return text
	}
	digits := FormatUint(rounded, 10)
	for len(digits) < precision {
		digits = "0" + digits
	}
	return text + "." + digits
}
func formatFloatExponent(value float64, format byte, precision int, neg bool) string {
	exp := 0
	if value != 0 {
		for value >= 10 {
			value /= 10
			exp++
		}
		for value < 1 {
			value *= 10
			exp--
		}
	}
	text := formatFloatFixed(value, precision)
	sign := "+"
	if exp < 0 {
		sign = "-"
		exp = -exp
	}
	exponent := Itoa(exp)
	if len(exponent) < 2 {
		exponent = "0" + exponent
	}
	if neg {
		text = "-" + text
	}
	return text + string([]byte{format}) + sign + exponent
}
func trimFloatZeros(text string) string {
	dot := -1
	for i := 0; i < len(text); i++ {
		if text[i] == '.' {
			dot = i
			break
		}
	}
	if dot < 0 {
		return text
	}
	for len(text) > dot+1 && text[len(text)-1] == '0' {
		text = text[:len(text)-1]
	}
	if len(text) == dot+1 {
		text = text[:dot]
	}
	return text
}

func AppendBool(dst []byte, value bool) []byte { return appendString(dst, FormatBool(value)) }
func AppendInt(dst []byte, value int64, base int) []byte {
	return appendString(dst, FormatInt(value, base))
}
func AppendUint(dst []byte, value uint64, base int) []byte {
	return appendString(dst, FormatUint(value, base))
}
func AppendFloat(dst []byte, value float64, format byte, precision int, bitSize int) []byte {
	return appendString(dst, FormatFloat(value, format, precision, bitSize))
}
func AppendQuote(dst []byte, value string) []byte { return appendString(dst, Quote(value)) }
