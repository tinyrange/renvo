package io

type Reader interface {
	Read(p []byte) (n int, err error)
}

type Writer interface {
	Write(p []byte) (n int, err error)
}

type Closer interface {
	Close() error
}

type ReadCloser interface {
	Reader
	Closer
}

type StringWriter interface {
	WriteString(s string) (n int, err error)
}

type eofError struct{}
type shortWriteError struct{}

func (eofError) Error() string        { return "EOF" }
func (shortWriteError) Error() string { return "short write" }

var EOF error = eofError{}
var ErrShortWrite error = shortWriteError{}

// Discard succeeds without retaining written data.
var Discard Writer = discard{}

type discard struct{}

func (discard) Write(p []byte) (int, error) { return len(p), nil }

type limitedReader struct {
	r Reader
	n int64
}

// LimitReader returns a Reader that reads at most n bytes from r.
func LimitReader(r Reader, n int64) Reader { return &limitedReader{r: r, n: n} }

func (l *limitedReader) Read(p []byte) (int, error) {
	if l.n <= 0 {
		return 0, EOF
	}
	if int64(len(p)) > l.n {
		p = p[:int(l.n)]
	}
	n, err := l.r.Read(p)
	l.n -= int64(n)
	return n, err
}

type nopCloser struct{ Reader }

func (nopCloser) Close() error { return nil }

// NopCloser wraps a Reader with a no-op Close method.
func NopCloser(r Reader) ReadCloser { return nopCloser{Reader: r} }

func ReadAll(r Reader) ([]byte, error) {
	var out []byte
	buf := make([]byte, 512)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			out = append(out, buf[:n]...)
		}
		if err != nil {
			if err == EOF {
				return out, nil
			}
			return out, err
		}
		if n == 0 {
			return out, nil
		}
	}
}

func Copy(dst Writer, src Reader) (int64, error) {
	var total int64
	buf := make([]byte, 32768)
	for {
		n, readErr := src.Read(buf)
		if n > 0 {
			written, writeErr := dst.Write(buf[:n])
			total += int64(written)
			if writeErr != nil {
				return total, writeErr
			}
			if written != n {
				return total, ErrShortWrite
			}
		}
		if readErr != nil {
			if readErr == EOF {
				return total, nil
			}
			return total, readErr
		}
		if n == 0 {
			return total, nil
		}
	}
}

func WriteString(w Writer, s string) (int, error) {
	if sw, ok := w.(StringWriter); ok {
		return sw.WriteString(s)
	}
	data := []byte(s)
	return w.Write(data)
}
