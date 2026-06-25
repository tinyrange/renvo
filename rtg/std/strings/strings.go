package strings

func HasPrefix(s string, prefix string) bool {
	n := len(prefix)
	if n > len(s) {
		return false
	}
	return s[0:n] == prefix
}

func HasSuffix(s string, suffix string) bool {
	n := len(suffix)
	if n > len(s) {
		return false
	}
	start := len(s) - n
	return s[start:len(s)] == suffix
}

func Contains(s string, substr string) bool {
	n := len(substr)
	if n == 0 {
		return true
	}
	if n > len(s) {
		return false
	}
	limit := len(s) - n
	i := 0
	for i <= limit {
		if s[i:i+n] == substr {
			return true
		}
		i = i + 1
	}
	return false
}

func IndexByte(s string, c byte) int {
	i := 0
	for i < len(s) {
		if s[i] == c {
			return i
		}
		i = i + 1
	}
	return -1
}
