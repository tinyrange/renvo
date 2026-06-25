package bytes

type Buffer struct {
	buf []byte
}

func (b *Buffer) WriteString(s string) int {
	i := 0
	for i < len(s) {
		b.buf = append(b.buf, s[i])
		i = i + 1
	}
	return len(s)
}

func (b *Buffer) WriteByte(c byte) int {
	b.buf = append(b.buf, c)
	return 0
}

func (b *Buffer) Len() int {
	return len(b.buf)
}

func (b *Buffer) String() string {
	return string(b.buf)
}

func (b *Buffer) Bytes() []byte {
	return b.buf
}

func (b *Buffer) Reset() {
	var empty []byte
	b.buf = empty
}
