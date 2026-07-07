//go:build rtg

package strconv

type NumError struct {
	text string
}

func Itoa(i int) string {
	return FormatInt(i, 10)
}

func Atoi(s string) (int, *NumError) {
	if len(s) == 0 {
		return 0, nil
	}
	neg := false
	if s[0] == '-' {
		neg = true
		s = s[1:]
	}
	out := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return 0, nil
		}
		out = out*10 + int(c-'0')
	}
	if neg {
		out = -out
	}
	return out, nil
}

func FormatInt(i int, base int) string {
	if base == 16 {
		return formatBase(i, 16)
	}
	return formatBase(i, 10)
}

func formatBase(i int, base int) string {
	if i == 0 {
		return "0"
	}
	var out []byte
	if i < 0 {
		out = append(out, '-')
		i = -i
	}
	var digits []byte
	for i > 0 {
		d := i % base
		if d < 10 {
			digits = append(digits, byte('0'+d))
		} else {
			digits = append(digits, byte('a'+d-10))
		}
		i = i / base
	}
	for i := len(digits) - 1; i >= 0; i-- {
		out = append(out, digits[i])
	}
	return string(out)
}
