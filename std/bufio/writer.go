package bufio

import (
	"io"
	"unicode/utf8"
)

type Writer struct {
	err error
	buf []byte
	n   int
	wr  io.Writer
}

func NewWriter(w io.Writer) *Writer { return NewWriterSize(w, defaultBufSize) }
func NewWriterSize(w io.Writer, size int) *Writer {
	if size <= 0 {
		size = defaultBufSize
	}
	return &Writer{buf: make([]byte, size), wr: w}
}
func (b *Writer) Size() int               { return len(b.buf) }
func (b *Writer) Available() int          { return len(b.buf) - b.n }
func (b *Writer) AvailableBuffer() []byte { return b.buf[b.n:b.n] }
func (b *Writer) Buffered() int           { return b.n }
func (b *Writer) Reset(w io.Writer)       { b.err, b.n, b.wr = nil, 0, w }

func (b *Writer) Flush() error {
	if b.err != nil {
		return b.err
	}
	if b.n == 0 {
		return nil
	}
	n, err := b.wr.Write(b.buf[:b.n])
	if n < 0 || n > b.n {
		n = 0
		err = ErrBadReadCount
	}
	if n < b.n && err == nil {
		err = io.ErrShortWrite
	}
	if err != nil {
		if n > 0 {
			copy(b.buf, b.buf[n:b.n])
			b.n -= n
		}
		b.err = err
		return err
	}
	b.n = 0
	return nil
}

func (b *Writer) Write(p []byte) (int, error) {
	written := 0
	for len(p) > b.Available() && b.err == nil {
		if b.n == 0 {
			n, err := b.wr.Write(p)
			written += n
			p = p[n:]
			if err != nil {
				b.err = err
			}
		} else {
			n := copy(b.buf[b.n:], p)
			b.n += n
			written += n
			p = p[n:]
			b.Flush()
		}
	}
	if b.err != nil {
		return written, b.err
	}
	n := copy(b.buf[b.n:], p)
	b.n += n
	written += n
	return written, nil
}
func (b *Writer) WriteByte(c byte) error {
	if b.Available() <= 0 && b.Flush() != nil {
		return b.err
	}
	b.buf[b.n] = c
	b.n++
	return nil
}
func (b *Writer) WriteRune(r rune) (int, error) {
	if r < utf8.RuneSelf {
		return 1, b.WriteByte(byte(r))
	}
	var data [utf8.UTFMax]byte
	if r < 0 || r > utf8.MaxRune || (r >= 0xd800 && r <= 0xdfff) {
		r = utf8.RuneError
	}
	value := uint32(r)
	n := 2
	if r < 0x800 {
		data[0] = byte(0xc0 + value/64)
		data[1] = byte(0x80 + value%64)
	} else if r < 0x10000 {
		n = 3
		data[0] = byte(0xe0 + value/4096)
		data[1] = byte(0x80 + value/64&0x3f)
		data[2] = byte(0x80 + value%64)
	} else {
		n = 4
		data[0] = byte(0xf0 + value/262144)
		data[1] = byte(0x80 + value/4096&0x3f)
		data[2] = byte(0x80 + value/64&0x3f)
		data[3] = byte(0x80 + value%64)
	}
	written, err := b.Write(data[:n])
	return written, err
}
func (b *Writer) WriteString(s string) (int, error) {
	return b.Write([]byte(s))
}
func (b *Writer) ReadFrom(r io.Reader) (int64, error) {
	var total int64
	for {
		if b.Available() == 0 {
			if err := b.Flush(); err != nil {
				return total, err
			}
		}
		n, err := r.Read(b.buf[b.n:])
		if n < 0 || n > b.Available() {
			b.err = ErrBadReadCount
			return total, b.err
		}
		b.n += n
		total += int64(n)
		if err != nil {
			if err == io.EOF {
				return total, nil
			}
			return total, err
		}
		if n == 0 {
			return total, nil
		}
	}
}
