package bytes

import "testing"

func TestByteHelpers(t *testing.T) {
	if !Equal([]byte("abc"), []byte("abc")) || Compare([]byte("abc"), []byte("abd")) >= 0 {
		t.Fatalf("compare failed")
	}
	if !Contains([]byte("alpha"), []byte("ph")) || !HasPrefix([]byte("alpha"), []byte("al")) || !HasSuffix([]byte("alpha"), []byte("ha")) {
		t.Fatalf("search failed")
	}
	if string(TrimSpace([]byte(" \tgo\n"))) != "go" {
		t.Fatalf("trim failed")
	}
	parts := Split([]byte("a,b,c"), []byte(","))
	if len(parts) != 3 || string(Join(parts, []byte("|"))) != "a|b|c" || string(Repeat([]byte("ab"), 2)) != "abab" {
		t.Fatalf("split/join/repeat failed")
	}
}

func TestBuffer(t *testing.T) {
	var b Buffer
	n, err := b.WriteString("hello")
	if n != 5 || err != nil || b.Len() != 5 || b.String() != "hello" {
		t.Fatalf("write string failed")
	}
	buf := make([]byte, 2)
	n, err = b.Read(buf)
	if n != 2 || err != nil || string(buf) != "he" || b.String() != "llo" {
		t.Fatalf("read failed: n=%d err=%v buf=%q rest=%q", n, err, string(buf), b.String())
	}
	b.Reset()
	if b.Len() != 0 || b.String() != "" {
		t.Fatalf("reset failed")
	}
}
