//go:build renvo && !renvo_wasi_c_object

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
	return 0, false
}
