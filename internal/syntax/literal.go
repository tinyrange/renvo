package syntax

func StringLiteralValue(src []byte, tok Token) (string, bool) {
	start := int(tok.Start)
	end := int(tok.End)
	if tok.KindLine&255 != TokenString || start < 0 || end-start < 2 || end > len(src) {
		return "", false
	}
	quote := src[start]
	if quote == '`' {
		if src[end-1] != '`' {
			return "", false
		}
		return string(src[start+1 : end-1]), true
	}
	if quote != '"' || src[end-1] != '"' {
		return "", false
	}
	out := make([]byte, 0, end-start-2)
	i := start + 1
	for i < end-1 {
		c := src[i]
		if c != '\\' {
			out = append(out, c)
			i++
			continue
		}
		i++
		if i >= end-1 {
			return "", false
		}
		next, value, unicode, ok := stringEscapeValue(src, i-1, end-1)
		if !ok {
			return "", false
		}
		if unicode {
			out = appendStringRune(out, value)
		} else {
			out = append(out, byte(value))
		}
		i = next
	}
	return string(out), true
}

func stringEscapeValue(src []byte, slash int, end int) (int, int, bool, bool) {
	if slash < 0 || slash+1 >= end || end > len(src) {
		return 0, 0, false, false
	}
	esc := src[slash+1]
	simple := "abfnrtv\\\""
	values := "\a\b\f\n\r\t\v\\\""
	for i := 0; i < len(simple); i++ {
		if esc == simple[i] {
			return slash + 2, int(values[i]), false, true
		}
	}
	digits := 0
	base := 16
	unicode := false
	start := slash + 2
	if esc == 'x' {
		digits = 2
	} else if esc == 'u' {
		digits = 4
		unicode = true
	} else if esc == 'U' {
		digits = 8
		unicode = true
	} else if esc >= '0' && esc <= '7' {
		digits = 3
		base = 8
		start = slash + 1
	} else {
		return 0, 0, false, false
	}
	if start+digits > end {
		return 0, 0, false, false
	}
	value := 0
	for i := 0; i < digits; i++ {
		digit, ok := hexValue(src[start+i])
		if !ok || digit >= base {
			return 0, 0, false, false
		}
		value = value*base + digit
	}
	if unicode && (value > 0x10ffff || value >= 0xd800 && value <= 0xdfff) || !unicode && value > 255 {
		return 0, 0, false, false
	}
	return start + digits, value, unicode, true
}

func appendStringRune(out []byte, value int) []byte {
	if value < 0x80 {
		return append(out, byte(value))
	}
	if value < 0x800 {
		return append(out, byte(0xc0|value>>6), byte(0x80|value&0x3f))
	}
	if value < 0x10000 {
		return append(out, byte(0xe0|value>>12), byte(0x80|value>>6&0x3f), byte(0x80|value&0x3f))
	}
	return append(out, byte(0xf0|value>>18), byte(0x80|value>>12&0x3f), byte(0x80|value>>6&0x3f), byte(0x80|value&0x3f))
}

func hexValue(c byte) (int, bool) {
	if c >= '0' && c <= '9' {
		return int(c - '0'), true
	}
	if c >= 'a' && c <= 'f' {
		return int(c-'a') + 10, true
	}
	if c >= 'A' && c <= 'F' {
		return int(c-'A') + 10, true
	}
	return 0, false
}
