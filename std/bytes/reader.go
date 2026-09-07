package bytes

import (
	"errors"
	"io"
	"unicode/utf8"
)

// Reader reads a byte slice without copying its backing storage.
type Reader struct {
	data         []byte
	at           int64
	previousRune int64
}

func NewReader(data []byte) *Reader { return &Reader{data: data, previousRune: -1} }
func (r *Reader) Reset(data []byte) { r.data = data; r.at = 0; r.previousRune = -1 }
func (r *Reader) Size() int64       { return int64(len(r.data)) }
func (r *Reader) Len() int {
	if r.at >= int64(len(r.data)) {
		return 0
	}
	return len(r.data) - int(r.at)
}
func (r *Reader) Read(p []byte) (int, error) {
	r.previousRune = -1
	if r.at >= int64(len(r.data)) {
		return 0, io.EOF
	}
	n := copy(p, r.data[int(r.at):])
	r.at += int64(n)
	return n, nil
}
func (r *Reader) ReadAt(p []byte, offset int64) (int, error) {
	if offset < 0 {
		return 0, errors.New("bytes.Reader.ReadAt: negative offset")
	}
	if offset >= int64(len(r.data)) {
		return 0, io.EOF
	}
	n := copy(p, r.data[int(offset):])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}
func (r *Reader) ReadByte() (byte, error) {
	r.previousRune = -1
	if r.at >= int64(len(r.data)) {
		return 0, io.EOF
	}
	b := r.data[int(r.at)]
	r.at++
	return b, nil
}
func (r *Reader) UnreadByte() error {
	if r.at <= 0 {
		return errors.New("bytes.Reader.UnreadByte: at beginning of slice")
	}
	r.previousRune = -1
	r.at--
	return nil
}
func (r *Reader) ReadRune() (rune, int, error) {
	if r.at >= int64(len(r.data)) {
		r.previousRune = -1
		return 0, 0, io.EOF
	}
	r.previousRune = r.at
	value, size := utf8.DecodeRune(r.data[int(r.at):])
	r.at += int64(size)
	return value, size, nil
}
func (r *Reader) UnreadRune() error {
	if r.at <= 0 || r.previousRune < 0 {
		return errors.New("bytes.Reader.UnreadRune: previous operation was not ReadRune")
	}
	r.at = r.previousRune
	r.previousRune = -1
	return nil
}
func (r *Reader) Seek(offset int64, whence int) (int64, error) {
	r.previousRune = -1
	position := offset
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		position += r.at
	case io.SeekEnd:
		position += int64(len(r.data))
	default:
		return 0, errors.New("bytes.Reader.Seek: invalid whence")
	}
	if position < 0 {
		return 0, errors.New("bytes.Reader.Seek: negative position")
	}
	r.at = position
	return position, nil
}
func (r *Reader) WriteTo(w io.Writer) (int64, error) {
	r.previousRune = -1
	if r.at >= int64(len(r.data)) {
		return 0, nil
	}
	remaining := r.data[int(r.at):]
	n, err := w.Write(remaining)
	if n < 0 || n > len(remaining) {
		panic("bytes.Reader.WriteTo: invalid Write count")
	}
	r.at += int64(n)
	if err == nil && n < len(remaining) {
		err = io.ErrShortWrite
	}
	return int64(n), err
}
