package bytes

import (
	"io"
	"testing"
)

func TestReaderNavigation(t *testing.T) {
	var zero Reader
	if zero.UnreadRune() == nil {
		t.Fatal("zero reader unread rune")
	}
	r := NewReader([]byte("aéZ"))
	if r.Len() != 4 || r.Size() != 4 {
		t.Fatal("initial size")
	}
	b, err := r.ReadByte()
	if err != nil || b != 'a' {
		t.Fatal("byte")
	}
	ch, size, err := r.ReadRune()
	if err != nil || ch != 'é' || size != 2 {
		t.Fatal("rune")
	}
	if r.UnreadRune() != nil || r.Len() != 3 {
		t.Fatal("unread rune")
	}
	var out [2]byte
	n, err := r.ReadAt(out[:], 2)
	if n != 2 || err != nil || out[1] != 'Z' || r.Len() != 3 {
		t.Fatal("read at")
	}
	if pos, err := r.Seek(-1, io.SeekEnd); err != nil || pos != 3 {
		t.Fatal("seek")
	}
	if r.UnreadRune() == nil {
		t.Fatal("unread after seek")
	}
	n, err = r.Read(out[:])
	if n != 1 || err != nil || out[0] != 'Z' {
		t.Fatal("read tail")
	}
	if n, err := r.Read(out[:]); n != 0 || err != io.EOF {
		t.Fatal("EOF")
	}
	if _, err := r.Seek(-1, io.SeekStart); err == nil {
		t.Fatal("negative seek")
	}
	r.Reset([]byte("reset"))
	if r.Len() != 5 || r.Size() != 5 {
		t.Fatal("reset")
	}
	if r.UnreadByte() == nil {
		t.Fatal("unread at start")
	}
}

type readerTestWriter struct {
	data  []byte
	limit int
}

func (w *readerTestWriter) Write(data []byte) (int, error) {
	n := len(data)
	if n > w.limit {
		n = w.limit
	}
	w.data = append(w.data, data[:n]...)
	return n, nil
}
func TestReaderWriteTo(t *testing.T) {
	r := NewReader([]byte("abc"))
	w := &readerTestWriter{limit: 2}
	if n, err := r.WriteTo(w); n != 2 || err != io.ErrShortWrite || r.Len() != 1 {
		t.Fatal("short write")
	}
	if n, err := r.WriteTo(w); n != 1 || err != nil || string(w.data) != "abc" {
		t.Fatal("remaining write")
	}
	if n, err := r.WriteTo(w); n != 0 || err != nil {
		t.Fatal("empty write")
	}
}
