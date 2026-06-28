package strings

type Builder struct {
	buf []byte
}

func (b *Builder) WriteString(s string) int {
	i := 0
	for i < len(s) {
		b.buf = append(b.buf, s[i])
		i = i + 1
	}
	return len(s)
}

func (b *Builder) Write(p []byte) int {
	i := 0
	for i < len(p) {
		b.buf = append(b.buf, p[i])
		i = i + 1
	}
	return len(p)
}

func (b *Builder) WriteByte(c byte) int {
	b.buf = append(b.buf, c)
	return 0
}

func (b *Builder) Len() int {
	return len(b.buf)
}

func (b *Builder) String() string {
	return string(b.buf)
}

func (b *Builder) Reset() {
	var empty []byte
	b.buf = empty
}

func HasPrefix(s string, prefix string) bool {
	n := len(prefix)
	if n > len(s) {
		return false
	}
	for i := 0; i < n; i++ {
		if s[i] != prefix[i] {
			return false
		}
	}
	return true
}

func HasSuffix(s string, suffix string) bool {
	n := len(suffix)
	if n > len(s) {
		return false
	}
	start := len(s) - n
	for i := 0; i < n; i++ {
		if s[start+i] != suffix[i] {
			return false
		}
	}
	return true
}

func Contains(s string, substr string) bool {
	return Index(s, substr) >= 0
}

func Index(s string, substr string) int {
	n := len(substr)
	if n == 0 {
		return 0
	}
	if n > len(s) {
		return -1
	}
	limit := len(s) - n
	i := 0
	for i <= limit {
		if equalAt(s, substr, i) {
			return i
		}
		i = i + 1
	}
	return -1
}

func LastIndex(s string, substr string) int {
	n := len(substr)
	if n == 0 {
		return len(s)
	}
	if n > len(s) {
		return -1
	}
	i := len(s) - n
	for i >= 0 {
		if equalAt(s, substr, i) {
			return i
		}
		i = i - 1
	}
	return -1
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

func LastIndexByte(s string, c byte) int {
	i := len(s) - 1
	for i >= 0 {
		if s[i] == c {
			return i
		}
		i = i - 1
	}
	return -1
}

func Count(s string, substr string) int {
	n := len(substr)
	if n == 0 {
		return len(s) + 1
	}
	count := 0
	i := 0
	for i <= len(s)-n {
		if equalAt(s, substr, i) {
			count = count + 1
			i = i + n
		} else {
			i = i + 1
		}
	}
	return count
}

func Repeat(s string, count int) string {
	var out []byte
	i := 0
	for i < count {
		j := 0
		for j < len(s) {
			out = append(out, s[j])
			j = j + 1
		}
		i = i + 1
	}
	return string(out)
}

func equalAt(s string, substr string, start int) bool {
	for i := 0; i < len(substr); i++ {
		if s[start+i] != substr[i] {
			return false
		}
	}
	return true
}

func TrimPrefix(s string, prefix string) string {
	if HasPrefix(s, prefix) {
		return s[len(prefix):len(s)]
	}
	return s
}

func TrimSuffix(s string, suffix string) string {
	if HasSuffix(s, suffix) {
		return s[0 : len(s)-len(suffix)]
	}
	return s
}

func TrimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && isSpace(s[start]) {
		start = start + 1
	}
	for end > start && isSpace(s[end-1]) {
		end = end - 1
	}
	return s[start:end]
}

func Join(elems []string, sep string) string {
	var out []byte
	i := 0
	for i < len(elems) {
		if i > 0 {
			j := 0
			for j < len(sep) {
				out = append(out, sep[j])
				j = j + 1
			}
		}
		elem := elems[i]
		j := 0
		for j < len(elem) {
			out = append(out, elem[j])
			j = j + 1
		}
		i = i + 1
	}
	return string(out)
}

func Split(s string, sep string) []string {
	var out []string
	if len(sep) == 0 {
		i := 0
		for i < len(s) {
			out = append(out, s[i:i+1])
			i = i + 1
		}
		return out
	}
	start := 0
	for start <= len(s) {
		next := Index(s[start:len(s)], sep)
		if next < 0 {
			out = append(out, s[start:len(s)])
			return out
		}
		end := start + next
		out = append(out, s[start:end])
		start = end + len(sep)
	}
	return out
}

func Fields(s string) []string {
	var out []string
	i := 0
	for i < len(s) {
		for i < len(s) && isSpace(s[i]) {
			i = i + 1
		}
		start := i
		for i < len(s) && !isSpace(s[i]) {
			i = i + 1
		}
		if start < i {
			out = append(out, s[start:i])
		}
	}
	return out
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}
