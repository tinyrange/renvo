package strings

func Contains(s string, substr string) bool {
	return Index(s, substr) >= 0
}

func HasPrefix(s string, prefix string) bool {
	if len(prefix) > len(s) {
		return false
	}
	return s[:len(prefix)] == prefix
}

func HasSuffix(s string, suffix string) bool {
	if len(suffix) > len(s) {
		return false
	}
	return s[len(s)-len(suffix):] == suffix
}

func Index(s string, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	if len(substr) > len(s) {
		return -1
	}
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func LastIndex(s string, substr string) int {
	if len(substr) == 0 {
		return len(s)
	}
	if len(substr) > len(s) {
		return -1
	}
	for i := len(s) - len(substr); i >= 0; i-- {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func Count(s string, substr string) int {
	if len(substr) == 0 {
		count := 1
		for i := 0; i < len(s); i++ {
			if s[i]&0xc0 != 0x80 {
				count++
			}
		}
		return count
	}
	count := 0
	start := 0
	for start+len(substr) <= len(s) {
		i := Index(s[start:], substr)
		if i < 0 {
			break
		}
		count++
		start += i + len(substr)
	}
	return count
}

func TrimSpace(s string) string {
	start := 0
	for start < len(s) && isSpace(s[start]) {
		start++
	}
	end := len(s)
	for end > start && isSpace(s[end-1]) {
		end--
	}
	return s[start:end]
}

func TrimPrefix(s string, prefix string) string {
	if HasPrefix(s, prefix) {
		return s[len(prefix):]
	}
	return s
}

func TrimSuffix(s string, suffix string) string {
	if HasSuffix(s, suffix) {
		return s[:len(s)-len(suffix)]
	}
	return s
}

func Split(s string, sep string) []string {
	if sep == "" {
		out := make([]string, 0, len(s))
		for i := 0; i < len(s); i++ {
			out = append(out, s[i:i+1])
		}
		return out
	}
	var out []string
	start := 0
	for {
		i := Index(s[start:], sep)
		if i < 0 {
			out = append(out, s[start:])
			return out
		}
		out = append(out, s[start:start+i])
		start = start + i + len(sep)
	}
}

func Join(items []string, sep string) string {
	if len(items) == 0 {
		return ""
	}
	var out []byte
	for i := 0; i < len(items); i++ {
		if i > 0 {
			out = appendString(out, sep)
		}
		out = appendString(out, items[i])
	}
	return string(out)
}

func Fields(s string) []string {
	var out []string
	i := 0
	for i < len(s) {
		for i < len(s) && isSpace(s[i]) {
			i++
		}
		start := i
		for i < len(s) && !isSpace(s[i]) {
			i++
		}
		if start < i {
			out = append(out, s[start:i])
		}
	}
	return out
}

func Repeat(s string, count int) string {
	if count <= 0 || len(s) == 0 {
		return ""
	}
	var out []byte
	for i := 0; i < count; i++ {
		out = appendString(out, s)
	}
	return string(out)
}

func Replace(s string, old string, new string, n int) string {
	if old == "" || n == 0 {
		return s
	}
	var out []byte
	start := 0
	done := 0
	for start < len(s) {
		i := Index(s[start:], old)
		if i < 0 || (n > 0 && done >= n) {
			out = appendString(out, s[start:])
			return string(out)
		}
		out = appendString(out, s[start:start+i])
		out = appendString(out, new)
		start = start + i + len(old)
		done++
	}
	return string(out)
}

func ReplaceAll(s string, old string, new string) string {
	return Replace(s, old, new, -1)
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\n' || c == '\t' || c == '\r' || c == '\v' || c == '\f'
}

func appendString(out []byte, s string) []byte {
	for i := 0; i < len(s); i++ {
		out = append(out, s[i])
	}
	return out
}
