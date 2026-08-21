package io

import "testing"

type sliceReader struct {
	data []byte
}

func (r *sliceReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, EOF
	}
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, nil
}

type sliceWriter struct {
	data []byte
}

func (w *sliceWriter) Write(p []byte) (int, error) {
	w.data = append(w.data, p...)
	return len(p), nil
}

func (w *sliceWriter) WriteString(s string) (int, error) {
	w.data = append(w.data, []byte(s)...)
	return len(s), nil
}

func TestReadAllCopyWriteString(t *testing.T) {
	all, err := ReadAll(&sliceReader{data: []byte("abcdef")})
	if err != nil || string(all) != "abcdef" {
		t.Fatalf("ReadAll = %q, %v", string(all), err)
	}
	var w sliceWriter
	n, err := Copy(&w, &sliceReader{data: []byte("xyz")})
	if n != 3 || err != nil || string(w.data) != "xyz" {
		t.Fatalf("Copy = %d, %v, %q", n, err, string(w.data))
	}
	n2, err := WriteString(&w, "ok")
	if n2 != 2 || err != nil || string(w.data) != "xyzok" {
		t.Fatalf("WriteString = %d, %v, %q", n2, err, string(w.data))
	}
}

func TestLimitReaderNopCloserAndDiscard(t *testing.T) {
	source := &sliceReader{data: []byte("abcdef")}
	got, err := ReadAll(LimitReader(source, 3))
	if err != nil || string(got) != "abc" {
		t.Fatalf("LimitReader ReadAll = %q, %v", got, err)
	}
	closer := NopCloser(&sliceReader{data: []byte("x")})
	if closer.Close() != nil {
		t.Fatal("NopCloser.Close returned an error")
	}
	if n, err := Discard.Write([]byte("discarded")); n != 9 || err != nil {
		t.Fatalf("Discard.Write = %d, %v", n, err)
	}
}
