package strings

import (
	"unicode"
	"unicode/utf8"
)

func Compare(a, b string) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}
func IndexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

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
		return utf8.RuneCountInString(s) + 1
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
	return TrimFunc(s, unicode.IsSpace)
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
	return SplitN(s, sep, -1)
}

func SplitN(s, sep string, n int) []string {
	if n == 0 {
		return nil
	}
	if sep == "" {
		count := utf8.RuneCountInString(s)
		if n < 0 || n > count {
			n = count
		}
		out := make([]string, 0, n)
		for i := 0; i < n; i++ {
			if i == n-1 {
				out = append(out, s)
				break
			}
			_, size := utf8.DecodeRuneInString(s)
			out = append(out, s[:size])
			s = s[size:]
		}
		return out
	}
	var out []string
	start := 0
	for {
		i := Index(s[start:], sep)
		if i < 0 || n > 0 && len(out) == n-1 {
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
	return FieldsFunc(s, unicode.IsSpace)
}

func Repeat(s string, count int) string {
	if count < 0 {
		panic("strings: negative Repeat count")
	}
	if count == 0 || len(s) == 0 {
		return ""
	}
	if uint(count) > (^uint(0)>>1)/uint(len(s)) {
		panic("strings: Repeat output length overflow")
	}
	var out []byte
	for i := 0; i < count; i++ {
		out = appendString(out, s)
	}
	return string(out)
}

func Replace(s string, old string, new string, n int) string {
	if old == new || n == 0 {
		return s
	}
	if old == "" {
		var out []byte
		at, done := 0, 0
		for {
			if n < 0 || done < n {
				out = appendString(out, new)
				done++
			} else {
				out = appendString(out, s[at:])
				break
			}
			if at == len(s) {
				break
			}
			_, width := utf8.DecodeRuneInString(s[at:])
			out = appendString(out, s[at:at+width])
			at += width
		}
		return string(out)
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
