package bufio

import (
	"bytes"
	"io"
	"testing"
)

func TestWriterBufferFlushAndReset(t *testing.T) {
	var out bytes.Buffer
	w := NewWriterSize(&out, 8)
	if n, err := w.WriteString("hello"); n != 5 || err != nil {
		t.Fatal(n, err)
	}
	if w.Buffered() != 5 || w.Available() != 3 || out.Len() != 0 {
		t.Fatal(w.Buffered(), w.Available(), out.Len())
	}
	if err := w.WriteByte('!'); err != nil {
		t.Fatal(err)
	}
	if n, err := w.WriteRune('€'); n != 3 || err != nil {
		t.Fatal(n, err)
	}
	if err := w.Flush(); err != nil || out.String() != "hello!€" {
		t.Fatalf("Flush=%q,%v", out.String(), err)
	}

	var second bytes.Buffer
	w.Reset(&second)
	if _, err := w.WriteString("next"); err != nil {
		t.Fatal(err)
	}
	if err := w.Flush(); err != nil || second.String() != "next" {
		t.Fatal(second.String(), err)
	}
}

func TestWriterLargeWriteAndReadFrom(t *testing.T) {
	var out bytes.Buffer
	w := NewWriterSize(&out, 4)
	if n, err := w.Write([]byte("a long direct write")); n != 19 || err != nil {
		t.Fatal(n, err)
	}
	if _, err := w.ReadFrom(bytes.NewBufferString(" plus reader")); err != nil {
		t.Fatal(err)
	}
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	if out.String() != "a long direct write plus reader" {
		t.Fatal(out.String())
	}
}

type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) { return len(p) - 1, nil }

func TestWriterShortWriteStickyError(t *testing.T) {
	w := NewWriterSize(shortWriter{}, 4)
	if _, err := w.WriteString("abcd"); err != nil {
		t.Fatal(err)
	}
	if err := w.Flush(); err != io.ErrShortWrite {
		t.Fatalf("Flush=%v", err)
	}
	if _, err := w.WriteString("x"); err != io.ErrShortWrite {
		t.Fatalf("Write=%v", err)
	}
}
