//go:build rtg

package fmt

func Sprintf(format string, value int) string {
	var out []byte
	used := false
	for i := 0; i < len(format); i++ {
		c := format[i]
		if c != '%' {
			out = append(out, c)
			continue
		}
		if i+1 >= len(format) {
			out = append(out, '%')
			continue
		}
		i++
		verb := format[i]
		if verb == '%' {
			out = append(out, '%')
			continue
		}
		if !used && (verb == 'd' || verb == 'v') {
			out = appendInt(out, value)
			used = true
			continue
		}
		out = append(out, '%')
		out = append(out, '!')
		out = append(out, verb)
	}
	return string(out)
}

func appendInt(out []byte, value int) []byte {
	if value == 0 {
		return append(out, '0')
	}
	if value < 0 {
		out = append(out, '-')
		value = -value
	}
	var digits []byte
	for value > 0 {
		digits = append(digits, byte('0'+value%10))
		value = value / 10
	}
	for i := len(digits) - 1; i >= 0; i-- {
		out = append(out, digits[i])
	}
	return out
}
