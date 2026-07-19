package io

type Reader interface {
	Read(p []byte) (n int, err error)
}

type Writer interface {
	Write(p []byte) (n int, err error)
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
