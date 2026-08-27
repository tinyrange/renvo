//go:build !renvo || renvo_wasi_c_object

package c11

func ppParseChar(text []byte) (int64, bool) {
	start := 1
	if len(text) > 2 && (text[0] == 'L' || text[0] == 'u' || text[0] == 'U') {
		start++
	}
	if start >= len(text)-1 {
		return 0, false
	}
	if text[start] != '\\' {
		return int64(text[start]), start+2 == len(text)
	}
	start++
	if start >= len(text)-1 {
		return 0, false
	}
	switch text[start] {
	case 'n':
		return 10, true
	case 'r':
		return 13, true
	case 't':
		return 9, true
	case 'v':
		return 11, true
	case 'f':
		return 12, true
	case 'a':
		return 7, true
	case 'b':
		return 8, true
	case 'e':
		return 27, true
	case '\\', '\'', '"', '?':
		return int64(text[start]), true
	}
	if text[start] >= '0' && text[start] <= '7' {
		value, end := int64(0), start
		for end < len(text)-1 && end < start+3 && text[end] >= '0' && text[end] <= '7' {
			value, end = value*8+int64(text[end]-'0'), end+1
		}
		return value, end == len(text)-1
	}
	if text[start] == 'x' {
		value, digits := int64(0), 0
		for start++; start < len(text)-1; start++ {
			ch, digit := text[start], int64(-1)
			if ch >= '0' && ch <= '9' {
				digit = int64(ch - '0')
			} else if ch >= 'a' && ch <= 'f' {
				digit = int64(ch-'a') + 10
			} else if ch >= 'A' && ch <= 'F' {
				digit = int64(ch-'A') + 10
			}
			if digit < 0 {
				return 0, false
			}
			value, digits = value*16+digit, digits+1
		}
		return value, digits > 0
	}
	return 0, false
}
