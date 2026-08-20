package bufio

import (
	"io"
	"unicode/utf8"
)

const defaultBufSize = 4096

type Reader struct {
	buf          []byte
	rd           io.Reader
	r            int
	w            int
	lastByte     int
	lastRuneSize int
}

func NewReader(rd io.Reader) *Reader { return NewReaderSize(rd, defaultBufSize) }
func NewReaderSize(rd io.Reader, size int) *Reader {
	if size < 16 {
		size = 16
	}
	return &Reader{buf: make([]byte, size), rd: rd, lastByte: -1, lastRuneSize: -1}
}
func (b *Reader) Size() int     { return len(b.buf) }
func (b *Reader) Buffered() int { return b.w - b.r }
func (b *Reader) Reset(r io.Reader) {
	b.rd, b.r, b.w = r, 0, 0
	b.lastByte, b.lastRuneSize = -1, -1
}

func (b *Reader) fill() error {
	if b.r > 0 {
		copy(b.buf, b.buf[b.r:b.w])
		b.w -= b.r
		b.r = 0
	}
	if b.w == len(b.buf) {
		return ErrBufferFull
	}
	n, err := b.rd.Read(b.buf[b.w:])
	if n < 0 || b.w+n > len(b.buf) {
		return ErrBadReadCount
	}
	b.w += n
	if n == 0 && err == nil {
		return io.EOF
	}
	return err
}

func (b *Reader) Read(p []byte) (int, error) {
	b.lastByte, b.lastRuneSize = -1, -1
	if len(p) == 0 {
		return 0, nil
	}
	if b.r == b.w {
		if len(p) >= len(b.buf) {
			n, err := b.rd.Read(p)
			if n > 0 {
				b.lastByte = int(p[n-1])
			}
			return n, err
		}
		if err := b.fill(); err != nil && b.r == b.w {
			return 0, err
		}
	}
	n := copy(p, b.buf[b.r:b.w])
	b.r += n
	if n > 0 {
		b.lastByte = int(p[n-1])
	}
	return n, nil
}

func (b *Reader) ReadByte() (byte, error) {
	b.lastRuneSize = -1
	if b.r == b.w {
		if err := b.fill(); err != nil && b.r == b.w {
			return 0, err
		}
	}
	c := b.buf[b.r]
	b.r++
	b.lastByte = int(c)
	return c, nil
}
func (b *Reader) UnreadByte() error {
	if b.lastByte < 0 || b.r == 0 {
		return ErrInvalidUnreadByte
	}
	b.r--
	b.lastByte, b.lastRuneSize = -1, -1
	return nil
}

func (b *Reader) ReadRune() (rune, int, error) {
	for b.r == b.w || (b.buf[b.r] >= utf8.RuneSelf && b.w-b.r < 4) {
		err := b.fill()
		if err != nil && b.r == b.w {
			return 0, 0, err
		}
		if err != nil {
			break
		}
	}
	r, size := utf8.DecodeRuneInString(string(b.buf[b.r:b.w]))
	b.r += size
	b.lastByte = int(b.buf[b.r-1])
	b.lastRuneSize = size
	return r, size, nil
}
func (b *Reader) UnreadRune() error {
	if b.lastRuneSize < 0 || b.r < b.lastRuneSize {
		return ErrInvalidUnreadRune
	}
	b.r -= b.lastRuneSize
	b.lastByte, b.lastRuneSize = -1, -1
	return nil
}

func (b *Reader) Peek(n int) ([]byte, error) {
	if n < 0 {
		return nil, ErrNegativeCount
	}
	b.lastByte, b.lastRuneSize = -1, -1
	for b.w-b.r < n && b.w-b.r < len(b.buf) {
		if err := b.fill(); err != nil {
			break
		}
	}
	if n > len(b.buf) {
		return b.buf[b.r:b.w], ErrBufferFull
	}
	if b.w-b.r < n {
		return b.buf[b.r:b.w], io.EOF
	}
	return b.buf[b.r : b.r+n], nil
}
func (b *Reader) Discard(n int) (int, error) {
	if n < 0 {
		return 0, ErrNegativeCount
	}
	b.lastByte, b.lastRuneSize = -1, -1
	discarded := 0
	for discarded < n {
		if b.r == b.w {
			if err := b.fill(); err != nil && b.r == b.w {
				return discarded, err
			}
		}
		step := n - discarded
		if step > b.w-b.r {
			step = b.w - b.r
		}
		b.r += step
		discarded += step
	}
	return discarded, nil
}

func (b *Reader) ReadSlice(delim byte) ([]byte, error) {
	b.lastByte, b.lastRuneSize = -1, -1
	for {
		for i := b.r; i < b.w; i++ {
			if b.buf[i] == delim {
				line := b.buf[b.r : i+1]
				b.r = i + 1
				b.lastByte = int(delim)
				return line, nil
			}
		}
		if b.Buffered() >= len(b.buf) {
			line := b.buf[b.r:b.w]
			b.r = b.w
			return line, ErrBufferFull
		}
		if err := b.fill(); err != nil {
			line := b.buf[b.r:b.w]
			b.r = b.w
			if len(line) > 0 {
				b.lastByte = int(line[len(line)-1])
			}
			return line, err
		}
	}
}
func (b *Reader) ReadBytes(delim byte) ([]byte, error) {
	var out []byte
	for {
		part, err := b.ReadSlice(delim)
		out = append(out, part...)
		if err == nil || err != ErrBufferFull {
			return out, err
		}
	}
}
func (b *Reader) ReadString(delim byte) (string, error) {
	data, err := b.ReadBytes(delim)
	return string(data), err
}
func (b *Reader) ReadLine() (line []byte, isPrefix bool, err error) {
	line, err = b.ReadSlice(10)
	if err != nil && err == ErrBufferFull {
		return line, true, nil
	}
	if len(line) == 0 {
		return line, false, err
	}
	if line[len(line)-1] == 10 {
		line = line[:len(line)-1]
		if len(line) > 0 && line[len(line)-1] == 13 {
			line = line[:len(line)-1]
		}
	}
	return line, false, err
}
