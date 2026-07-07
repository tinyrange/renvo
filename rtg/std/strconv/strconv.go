//go:build !rtg

package strconv

var ErrSyntax error = syntaxError{}
var ErrRange error = rangeError{}

type syntaxError struct{}
type rangeError struct{}

func (syntaxError) Error() string { return "invalid syntax" }
func (rangeError) Error() string  { return "value out of range" }

func Itoa(i int) string {
	return FormatInt(int64(i), 10)
}

func Atoi(s string) (int, error) {
	v, err := ParseInt(s, 10, 0)
	return int(v), err
}

func FormatBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func ParseBool(s string) (bool, error) {
	if s == "1" || s == "t" || s == "T" || s == "true" || s == "TRUE" || s == "True" {
		return true, nil
	}
	if s == "0" || s == "f" || s == "F" || s == "false" || s == "FALSE" || s == "False" {
		return false, nil
	}
	return false, ErrSyntax
}

func FormatInt(i int64, base int) string {
	if base < 2 || base > 36 {
		base = 10
	}
	if i < 0 {
		var out []byte
		out = append(out, '-')
		out = appendString(out, FormatUint(uint64(-i), base))
		return string(out)
	}
	return FormatUint(uint64(i), base)
}

func FormatUint(i uint64, base int) string {
	if base < 2 || base > 36 {
		base = 10
	}
	if i == 0 {
		return "0"
	}
	var buf [65]byte
	pos := len(buf)
	b := uint64(base)
	for i > 0 {
		d := i % b
		pos--
		if d < 10 {
			buf[pos] = byte('0' + d)
		} else {
			buf[pos] = byte('a' + d - 10)
		}
		i = i / b
	}
	return string(buf[pos:])
}

func ParseInt(s string, base int, bitSize int) (int64, error) {
	if len(s) == 0 {
		return 0, ErrSyntax
	}
	neg := false
	if s[0] == '+' || s[0] == '-' {
		neg = s[0] == '-'
		s = s[1:]
		if len(s) == 0 {
			return 0, ErrSyntax
		}
	}
	u, err := ParseUint(s, base, bitSize)
	if err != nil {
		return 0, err
	}
	if neg {
		return -int64(u), nil
	}
	return int64(u), nil
}

func ParseUint(s string, base int, bitSize int) (uint64, error) {
	if len(s) == 0 {
		return 0, ErrSyntax
	}
	if base == 0 {
		base = 10
		if len(s) > 1 && s[0] == '0' {
			base = 8
			if len(s) > 2 && (s[1] == 'x' || s[1] == 'X') {
				base = 16
				s = s[2:]
			}
		}
	}
	if base < 2 || base > 36 || len(s) == 0 {
		return 0, ErrSyntax
	}
	var out uint64
	for i := 0; i < len(s); i++ {
		d, ok := digitValue(s[i])
		if !ok || d >= base {
			return 0, ErrSyntax
		}
		next := out*uint64(base) + uint64(d)
		if next < out {
			return 0, ErrRange
		}
		out = next
	}
	return out, nil
}

func Quote(s string) string {
	var out []byte
	out = append(out, '"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\\' || c == '"' {
			out = append(out, '\\')
			out = append(out, c)
		} else if c == '\n' {
			out = append(out, '\\')
			out = append(out, 'n')
		} else if c == '\t' {
			out = append(out, '\\')
			out = append(out, 't')
		} else if c == '\r' {
			out = append(out, '\\')
			out = append(out, 'r')
		} else {
			out = append(out, c)
		}
	}
	out = append(out, '"')
	return string(out)
}

func Unquote(s string) (string, error) {
	if len(s) < 2 || s[0] != '"' || s[len(s)-1] != '"' {
		return "", ErrSyntax
	}
	var out []byte
	for i := 1; i+1 < len(s); i++ {
		c := s[i]
		if c != '\\' {
			out = append(out, c)
			continue
		}
		i++
		if i+1 >= len(s) {
			return "", ErrSyntax
		}
		e := s[i]
		if e == 'n' {
			out = append(out, '\n')
		} else if e == 't' {
			out = append(out, '\t')
		} else if e == 'r' {
			out = append(out, '\r')
		} else if e == '\\' || e == '"' {
			out = append(out, e)
		} else {
			return "", ErrSyntax
		}
	}
	return string(out), nil
}

func digitValue(c byte) (int, bool) {
	if c >= '0' && c <= '9' {
		return int(c - '0'), true
	}
	if c >= 'a' && c <= 'z' {
		return int(c-'a') + 10, true
	}
	if c >= 'A' && c <= 'Z' {
		return int(c-'A') + 10, true
	}
	return 0, false
}

func appendString(out []byte, s string) []byte {
	for i := 0; i < len(s); i++ {
		out = append(out, s[i])
	}
	return out
}
