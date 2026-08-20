package time

import "errors"

type Duration int64

const (
	Nanosecond  Duration = 1
	Microsecond          = 1000 * Nanosecond
	Millisecond          = 1000 * Microsecond
	Second               = 1000 * Millisecond
	Minute               = 60 * Second
	Hour                 = 60 * Minute
)

func ParseDuration(s string) (Duration, error) {
	if s == "" {
		return 0, errors.New("time: invalid duration")
	}
	neg := false
	if s[0] == '-' || s[0] == '+' {
		neg = s[0] == '-'
		s = s[1:]
	}
	var total Duration
	for len(s) > 0 {
		whole := int64(0)
		digits := 0
		for digits < len(s) && s[digits] >= '0' && s[digits] <= '9' {
			whole = whole*10 + int64(s[digits]-'0')
			digits++
		}
		if digits == 0 {
			return 0, errors.New("time: invalid duration")
		}
		s = s[digits:]
		fraction := int64(0)
		scale := int64(1)
		if len(s) > 0 && s[0] == '.' {
			s = s[1:]
			count := 0
			for count < len(s) && s[count] >= '0' && s[count] <= '9' {
				if scale < 1000000000 {
					fraction = fraction*10 + int64(s[count]-'0')
					scale *= 10
				}
				count++
			}
			if count == 0 {
				return 0, errors.New("time: invalid duration")
			}
			s = s[count:]
		}
		unit := Duration(0)
		unitLen := 0
		if len(s) >= 2 && s[:2] == "ns" {
			unit = Nanosecond
			unitLen = 2
		} else if len(s) >= 2 && s[:2] == "us" {
			unit = Microsecond
			unitLen = 2
		} else if len(s) >= 2 && s[:2] == "ms" {
			unit = Millisecond
			unitLen = 2
		} else if len(s) >= 1 && s[0] == 's' {
			unit = Second
			unitLen = 1
		} else if len(s) >= 1 && s[0] == 'm' {
			unit = Minute
			unitLen = 1
		} else if len(s) >= 1 && s[0] == 'h' {
			unit = Hour
			unitLen = 1
		} else {
			return 0, errors.New("time: unknown unit")
		}
		total += Duration(whole)*unit + Duration(fraction)*unit/Duration(scale)
		s = s[unitLen:]
	}
	if neg {
		return -total, nil
	}
	return total, nil
}
func (d Duration) String() string {
	if d == 0 {
		return "0s"
	}
	neg := d < 0
	if neg {
		d = -d
	}
	out := ""
	if d >= Hour {
		out += decimal(int64(d/Hour)) + "h"
		d %= Hour
	}
	if d >= Minute {
		out += decimal(int64(d/Minute)) + "m"
		d %= Minute
	}
	if d >= Second {
		out += decimal(int64(d / Second))
		d %= Second
		if d > 0 {
			out += "." + fraction9(int64(d))
		}
		out += "s"
	} else if d >= Millisecond {
		out += decimal(int64(d/Millisecond)) + "ms"
	} else if d >= Microsecond {
		out += decimal(int64(d/Microsecond)) + "us"
	} else if d > 0 {
		out += decimal(int64(d)) + "ns"
	}
	if neg {
		return "-" + out
	}
	return out
}
func decimal(v int64) string {
	if v == 0 {
		return "0"
	}
	b := make([]byte, 0, 20)
	for v > 0 {
		b = append(b, byte('0'+v%10))
		v /= 10
	}
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	return string(b)
}
func fraction9(v int64) string {
	b := make([]byte, 9)
	for i := 8; i >= 0; i-- {
		b[i] = byte('0' + v%10)
		v /= 10
	}
	for len(b) > 0 && b[len(b)-1] == '0' {
		b = b[:len(b)-1]
	}
	return string(b)
}
