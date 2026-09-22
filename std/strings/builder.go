package strings

// Builder accumulates strings. A non-zero Builder must not be copied.
type Builder struct {
	addr *Builder
	buf  []byte
}

func (b *Builder) checkCopy() {
	if b.addr == nil {
		b.addr = b
	} else if b.addr != b {
		panic("strings: illegal use of non-zero Builder copied by value")
	}
}

func (b *Builder) String() string { return string(b.buf) }
func (b *Builder) Len() int       { return len(b.buf) }
func (b *Builder) Cap() int       { return cap(b.buf) }
func (b *Builder) Reset()         { b.addr = nil; b.buf = nil }

func (b *Builder) Grow(n int) {
	b.checkCopy()
	if n < 0 || n > int(^uint(0)>>1)-len(b.buf) {
		panic("strings.Builder.Grow: invalid count")
	}
	if n <= cap(b.buf)-len(b.buf) {
		return
	}
	buf := make([]byte, len(b.buf), len(b.buf)+n)
	copy(buf, b.buf)
	b.buf = buf
}

func (b *Builder) Write(p []byte) (int, error) {
	b.checkCopy()
	b.buf = append(b.buf, p...)
	return len(p), nil
}
func (b *Builder) WriteString(s string) (int, error) {
	b.checkCopy()
	b.buf = append(b.buf, s...)
	return len(s), nil
}
func (b *Builder) WriteByte(c byte) error {
	b.checkCopy()
	b.buf = append(b.buf, c)
	return nil
}
func (b *Builder) WriteRune(r rune) (int, error) {
	return b.WriteString(string(r))
}
