package utf8

const RuneError = '\uFFFD'
const RuneSelf = 0x80
const MaxRune = '\U0010FFFF'
const UTFMax = 4

func Valid(p []byte) bool {
	for len(p) > 0 {
		r, n := DecodeRune(p)
		if r == RuneError && n == 1 {
			return false
		}
		p = p[n:]
	}
	return true
}

func DecodeLastRuneInString(s string) (rune, int) {
	end := len(s)
	if end == 0 {
		return RuneError, 0
	}
	if s[end-1] < RuneSelf {
		return rune(s[end-1]), 1
	}
	start := end - 1
	limit := end - UTFMax
	if limit < 0 {
		limit = 0
	}
	for start > limit && s[start]&0xc0 == 0x80 {
		start--
	}
	r, size := DecodeRuneInString(s[start:])
	if start+size != end {
		return RuneError, 1
	}
	return r, size
}
func DecodeLastRune(p []byte) (rune, int) {
	if len(p) > UTFMax {
		p = p[len(p)-UTFMax:]
	}
	return DecodeLastRuneInString(string(p))
}

// DecodeRune decodes the first rune, returning RuneError and width 1 for an
// invalid encoding, or width 0 for an empty input.
func DecodeRune(p []byte) (rune, int) {
	if len(p) > UTFMax {
		p = p[:UTFMax]
	}
	return DecodeRuneInString(string(p))
}

func RuneLen(r rune) int {
	if r < 0 {
		return -1
	}
	if r < 0x80 {
		return 1
	}
	if r < 0x800 {
		return 2
	}
	if r < 0x10000 {
		if r >= 0xD800 && r <= 0xDFFF {
			return -1
		}
		return 3
	}
	if r <= MaxRune {
		return 4
	}
	return -1
}

func DecodeRuneInString(s string) (r rune, size int) {
	if len(s) == 0 {
		return RuneError, 0
	}
	c0 := s[0]
	if c0 < RuneSelf {
		return rune(c0), 1
	}
	if c0 < 0xC2 {
		return RuneError, 1
	}
	if c0 < 0xE0 {
		if len(s) < 2 || !isCont(s[1]) {
			return RuneError, 1
		}
		return rune(c0&0x1F)<<6 | rune(s[1]&0x3F), 2
	}
	if c0 < 0xF0 {
		if len(s) < 3 || !isCont(s[1]) || !isCont(s[2]) {
			return RuneError, 1
		}
		r := rune(c0&0x0F)<<12 | rune(s[1]&0x3F)<<6 | rune(s[2]&0x3F)
		if r < 0x800 || (r >= 0xD800 && r <= 0xDFFF) {
			return RuneError, 1
		}
		return r, 3
	}
	if c0 < 0xF5 {
		if len(s) < 4 || !isCont(s[1]) || !isCont(s[2]) || !isCont(s[3]) {
			return RuneError, 1
		}
		r := rune(c0&0x07)<<18 | rune(s[1]&0x3F)<<12 | rune(s[2]&0x3F)<<6 | rune(s[3]&0x3F)
		if r < 0x10000 || r > MaxRune {
			return RuneError, 1
		}
		return r, 4
	}
	return RuneError, 1
}

func RuneCountInString(s string) int {
	count := 0
	for len(s) > 0 {
		_, size := DecodeRuneInString(s)
		if size == 0 {
			break
		}
		s = s[size:]
		count++
	}
	return count
}

func EncodeRune(p []byte, r rune) int {
	n := RuneLen(r)
	if n < 0 {
		r = RuneError
		n = 3
	}
	if len(p) < n {
		panic("utf8.EncodeRune: buffer too short")
	}
	if n == 1 {
		p[0] = byte(r)
	} else if n == 2 {
		p[0] = byte(0xC0 | r>>6)
		p[1] = byte(0x80 | r&0x3F)
	} else if n == 3 {
		p[0] = byte(0xE0 | r>>12)
		p[1] = byte(0x80 | r>>6&0x3F)
		p[2] = byte(0x80 | r&0x3F)
	} else {
		p[0] = byte(0xF0 | r>>18)
		p[1] = byte(0x80 | r>>12&0x3F)
		p[2] = byte(0x80 | r>>6&0x3F)
		p[3] = byte(0x80 | r&0x3F)
	}
	return n
}

func ValidString(s string) bool {
	for len(s) > 0 {
		r, size := DecodeRuneInString(s)
		if r == RuneError && size == 1 && s[0] >= RuneSelf {
			return false
		}
		s = s[size:]
	}
	return true
}

func isCont(c byte) bool {
	return c >= 0x80 && c < 0xC0
}
