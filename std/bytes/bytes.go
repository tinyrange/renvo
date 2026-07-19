//go:build !renvo

package bytes

type eofError struct{}

func (eofError) Error() string { return "EOF" }

var errEOF error = eofError{}

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

func Contains(b []byte, subslice []byte) bool { return Index(b, subslice) >= 0 }

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

func Split(s []byte, sep []byte) [][]byte {
	if len(sep) == 0 {
		out := make([][]byte, 0, len(s))
		for i := 0; i < len(s); i++ {
			out = append(out, s[i:i+1])
		}
		return out
	}
	var out [][]byte
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

func Join(items [][]byte, sep []byte) []byte {
	var out []byte
	for i := 0; i < len(items); i++ {
		if i > 0 {
			out = append(out, sep...)
		}
		out = append(out, items[i]...)
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

type Buffer struct {
	buf []byte
	off int
}

func NewBuffer(buf []byte) *Buffer {
	return &Buffer{buf: buf}
}

func NewBufferString(s string) *Buffer {
	return NewBuffer([]byte(s))
}

func (b *Buffer) Write(p []byte) (int, error) {
	b.buf = append(b.buf, p...)
	return len(p), nil
}

func (b *Buffer) WriteString(s string) (int, error) {
	for i := 0; i < len(s); i++ {
		b.buf = append(b.buf, s[i])
	}
	return len(s), nil
}

func (b *Buffer) Read(p []byte) (int, error) {
	if b.off >= len(b.buf) {
		return 0, errEOF
	}
	n := copy(p, b.buf[b.off:])
	b.off += n
	return n, nil
}

func (b *Buffer) Bytes() []byte  { return b.buf[b.off:] }
func (b *Buffer) String() string { return string(b.Bytes()) }
func (b *Buffer) Len() int       { return len(b.buf) - b.off }

func (b *Buffer) Reset() {
	b.buf = nil
	b.off = 0
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\n' || c == '\t' || c == '\r' || c == '\v' || c == '\f'
}
