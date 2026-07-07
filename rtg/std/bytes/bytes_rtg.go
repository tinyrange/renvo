//go:build rtg

package bytes

type SplitResult struct {
	data   []byte
	starts []int
	ends   []int
}

func Equal(a []byte, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func Compare(a []byte, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	if len(a) < len(b) {
		return -1
	}
	if len(a) > len(b) {
		return 1
	}
	return 0
}

func Contains(b []byte, subslice []byte) bool {
	return Index(b, subslice) >= 0
}

func HasPrefix(s []byte, prefix []byte) bool {
	return len(prefix) <= len(s) && Equal(s[:len(prefix)], prefix)
}

func HasSuffix(s []byte, suffix []byte) bool {
	return len(suffix) <= len(s) && Equal(s[len(s)-len(suffix):], suffix)
}

func Index(s []byte, sep []byte) int {
	if len(sep) == 0 {
		return 0
	}
	if len(sep) > len(s) {
		return -1
	}
	for i := 0; i+len(sep) <= len(s); i++ {
		if Equal(s[i:i+len(sep)], sep) {
			return i
		}
	}
	return -1
}

func TrimSpace(s []byte) []byte {
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

func Split(s []byte, sep []byte) SplitResult {
	var out SplitResult
	out.data = s
	if len(sep) == 0 {
		for i := 0; i < len(s); i++ {
			out.starts = append(out.starts, i)
			out.ends = append(out.ends, i+1)
		}
		return out
	}
	start := 0
	for {
		i := Index(s[start:], sep)
		if i < 0 {
			out.starts = append(out.starts, start)
			out.ends = append(out.ends, len(s))
			return out
		}
		out.starts = append(out.starts, start)
		out.ends = append(out.ends, start+i)
		start = start + i + len(sep)
	}
}

func Join(items SplitResult, sep []byte) []byte {
	var out []byte
	for i := 0; i < len(items.starts); i++ {
		if i > 0 {
			out = append(out, sep...)
		}
		out = append(out, items.data[items.starts[i]:items.ends[i]]...)
	}
	return out
}

func Repeat(b []byte, count int) []byte {
	var out []byte
	for i := 0; i < count; i++ {
		out = append(out, b...)
	}
	return out
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\n' || c == '\t' || c == '\r' || c == '\v' || c == '\f'
}
