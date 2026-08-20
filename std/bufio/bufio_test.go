package bufio

import (
	"bytes"
	"io"
	"testing"
)

func TestReaderLinesPeekAndDiscard(t *testing.T) {
	r := NewReaderSize(bytes.NewBufferString("alpha\r\nbeta\ngamma"), 16)
	peek, err := r.Peek(5)
	if err != nil || string(peek) != "alpha" {
		t.Fatalf("Peek=%q,%v", peek, err)
	}
	if n, err := r.Discard(2); n != 2 || err != nil {
		t.Fatalf("Discard=%d,%v", n, err)
	}
	line, prefix, err := r.ReadLine()
	if string(line) != "pha" || prefix || err != nil {
		t.Fatalf("ReadLine=%q,%v,%v", line, prefix, err)
	}
	line, prefix, err = r.ReadLine()
	if string(line) != "beta" || prefix || err != nil {
		t.Fatalf("ReadLine=%q,%v,%v", line, prefix, err)
	}
	line, prefix, err = r.ReadLine()
	if string(line) != "gamma" || prefix || err != io.EOF {
		t.Fatalf("ReadLine=%q,%v,%v", line, prefix, err)
	}
}

func TestReaderByteRuneAndUnread(t *testing.T) {
	r := NewReader(bytes.NewBufferString("a€z"))
	b, err := r.ReadByte()
	if b != 'a' || err != nil {
		t.Fatal(b, err)
	}
	if err := r.UnreadByte(); err != nil {
		t.Fatal(err)
	}
	b, _ = r.ReadByte()
	if b != 'a' {
		t.Fatal(b)
	}
	rn, size, err := r.ReadRune()
	if rn != '€' || size != 3 || err != nil {
		t.Fatal(rn, size, err)
	}
	if err := r.UnreadRune(); err != nil {
		t.Fatal(err)
	}
	rn, _, _ = r.ReadRune()
	if rn != '€' {
		t.Fatal(rn)
	}
}

func TestReaderLongReadString(t *testing.T) {
	r := NewReaderSize(bytes.NewBufferString("abcdefghijklmnop-tail!next"), 16)
	got, err := r.ReadString('!')
	if got != "abcdefghijklmnop-tail!" || err != nil {
		t.Fatalf("ReadString=%q,%v", got, err)
	}
	got, err = r.ReadString('!')
	if got != "next" || err != io.EOF {
		t.Fatalf("ReadString=%q,%v", got, err)
	}
}
