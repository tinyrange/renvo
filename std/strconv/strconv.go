package strconv

var ErrSyntax = syntaxError{marker: 1}
var ErrRange = rangeError{marker: 2}

type syntaxError struct{ marker int }
type rangeError struct{ marker int }

type NumError struct {
	Func string
	Num  string
	Err  error
}

func (e *NumError) Error() string {
	return "strconv." + e.Func + ": parsing " + Quote(e.Num) + ": " + e.Err.Error()
}
func (e *NumError) Unwrap() error { return e.Err }

func (syntaxError) Error() string              { return "invalid syntax" }
func (rangeError) Error() string               { return "value out of range" }
func numError(fn, num string, err error) error { return &NumError{Func: fn, Num: num, Err: err} }

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
		magnitude := uint64(-(i + 1)) + 1
		out = appendString(out, FormatUint(magnitude, base))
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
	var reversed []byte
	b := uint64(base)
	for i > 0 {
		d := i % b
		if d < 10 {
			reversed = append(reversed, byte('0'+d))
		} else {
			reversed = append(reversed, byte('a'+d-10))
		}
		i = i / b
	}
	var out []byte
	for i := len(reversed) - 1; i >= 0; i-- {
		out = append(out, reversed[i])
	}
	return string(out)
}

func ParseInt(s string, base int, bitSize int) (int64, error) {
	original := s
	if s == "" {
		return 0, numError("ParseInt", original, ErrSyntax)
	}
	neg := false
	if s[0] == '+' || s[0] == '-' {
		neg = s[0] == '-'
		s = s[1:]
	}
	if s == "" {
		return 0, numError("ParseInt", original, ErrSyntax)
	}
	if bitSize == 0 {
		bitSize = 64
	}
	if bitSize < 1 || bitSize > 64 {
		return 0, numError("ParseInt", original, ErrSyntax)
	}
	limit := uint64(1) << uint(bitSize-1)
	value, syntax, overflow := parseUintMagnitude(s, base, limit)
	if syntax {
		return 0, numError("ParseInt", original, ErrSyntax)
	}
	if overflow || (!neg && value >= limit) {
		if neg {
			return -int64(limit-1) - 1, numError("ParseInt", original, ErrRange)
		}
		return int64(limit - 1), numError("ParseInt", original, ErrRange)
	}
	if neg {
		if value == limit {
			return -int64(limit-1) - 1, nil
		}
		return -int64(value), nil
	}
	return int64(value), nil
}

func ParseUint(s string, base int, bitSize int) (uint64, error) {
	original := s
	if bitSize == 0 {
		bitSize = 64
	}
	if bitSize < 1 || bitSize > 64 {
		return 0, numError("ParseUint", original, ErrSyntax)
	}
	limit := ^uint64(0)
	if bitSize < 64 {
		limit = uint64(1)<<uint(bitSize) - 1
	}
	value, syntax, overflow := parseUintMagnitude(s, base, limit)
	if syntax {
		return 0, numError("ParseUint", original, ErrSyntax)
	}
	if overflow {
		return limit, numError("ParseUint", original, ErrRange)
	}
	return value, nil
}

func parseUintMagnitude(s string, base int, limit uint64) (uint64, bool, bool) {
	if s == "" {
		return 0, true, false
	}
	allowUnderscore := base == 0
	if base == 0 {
		base = 10
		if len(s) > 1 && s[0] == '0' {
			base = 8
			if len(s) > 2 && (s[1] == 'x' || s[1] == 'X') {
				base = 16
				s = s[2:]
			} else if len(s) > 2 && (s[1] == 'b' || s[1] == 'B') {
				base = 2
				s = s[2:]
			} else if len(s) > 2 && (s[1] == 'o' || s[1] == 'O') {
				base = 8
				s = s[2:]
			}
		}
	}
	if base < 2 || base > 36 || s == "" {
		return 0, true, false
	}
	value := uint64(0)
	digits := 0
	underscore := false
	overflow := false
	for i := 0; i < len(s); i++ {
		if s[i] == '_' && allowUnderscore {
			if digits == 0 || underscore || i+1 == len(s) {
				return 0, true, false
			}
			underscore = true
			continue
		}
		d, ok := digitValue(s[i])
		if !ok || d >= base {
			return 0, true, false
		}
		digits++
		underscore = false
		if uint64(d) > limit || value > (limit-uint64(d))/uint64(base) {
			overflow = true
			value = limit
		} else if !overflow {
			value = value*uint64(base) + uint64(d)
		}
	}
	return value, digits == 0 || underscore, overflow
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
		} else if c == '\b' {
			out = append(out, '\\', 'b')
		} else if c == '\f' {
			out = append(out, '\\', 'f')
		} else if c < 0x20 || c == 0x7f {
			out = append(out, '\\', 'x', hexEscapeDigit(c>>4), hexEscapeDigit(c&15))
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
		} else if e == 'b' {
			out = append(out, '\b')
		} else if e == 'f' {
			out = append(out, '\f')
		} else if e == 'u' || e == 'U' {
			count := 4
			if e == 'U' {
				count = 8
			}
			if i+count >= len(s)-1 {
				return "", ErrSyntax
			}
			value, ok := parseHexEscape(s[i+1 : i+1+count])
			if !ok || value > 0x10ffff || value >= 0xd800 && value <= 0xdfff {
				return "", ErrSyntax
			}
			out = appendUTF8(out, value)
			i += count
		} else if e == 'x' {
			if i+2 >= len(s)-1 {
				return "", ErrSyntax
			}
			value, ok := parseHexEscape(s[i+1 : i+3])
			if !ok {
				return "", ErrSyntax
			}
			out = append(out, byte(value))
			i += 2
		} else if e == '\\' || e == '"' {
			out = append(out, e)
		} else {
			return "", ErrSyntax
		}
	}
	return string(out), nil
}

func hexEscapeDigit(value byte) byte {
	if value < 10 {
		return '0' + value
	}
	return 'a' + value - 10
}

func parseHexEscape(s string) (int, bool) {
	value := 0
	for i := 0; i < len(s); i++ {
		digit, ok := digitValue(s[i])
		if !ok || digit >= 16 {
			return 0, false
		}
		value = value*16 + digit
	}
	return value, true
}
func appendUTF8(out []byte, r int) []byte {
	if r <= 0x7f {
		return append(out, byte(r))
	}
	if r <= 0x7ff {
		return append(out, byte(0xc0|r>>6), byte(0x80|r&63))
	}
	if r <= 0xffff {
		return append(out, byte(0xe0|r>>12), byte(0x80|r>>6&63), byte(0x80|r&63))
	}
	return append(out, byte(0xf0|r>>18), byte(0x80|r>>12&63), byte(0x80|r>>6&63), byte(0x80|r&63))
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
