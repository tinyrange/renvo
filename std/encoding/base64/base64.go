package base64

type CorruptInputError int64

func (e CorruptInputError) Error() string { return "illegal base64 data at input byte" }

type Encoding struct {
	alphabet string
	pad      byte
}

func NewEncoding(a string) *Encoding             { return &Encoding{a, '='} }
func (e *Encoding) WithPadding(p rune) *Encoding { return &Encoding{e.alphabet, byte(p)} }

var StdEncoding = NewEncoding("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/")
var URLEncoding = NewEncoding("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_")
var RawStdEncoding = StdEncoding.WithPadding(-1)
var RawURLEncoding = URLEncoding.WithPadding(-1)

func (e *Encoding) EncodedLen(n int) int {
	if e.pad == 255 {
		return (n*8 + 5) / 6
	}
	return (n + 2) / 3 * 4
}
func (e *Encoding) Encode(dst, src []byte) {
	d := 0
	for len(src) >= 3 {
		v := uint(src[0])<<16 | uint(src[1])<<8 | uint(src[2])
		dst[d] = e.alphabet[v>>18]
		dst[d+1] = e.alphabet[v>>12&63]
		dst[d+2] = e.alphabet[v>>6&63]
		dst[d+3] = e.alphabet[v&63]
		d += 4
		src = src[3:]
	}
	if len(src) > 0 {
		v := uint(src[0]) << 16
		if len(src) > 1 {
			v |= uint(src[1]) << 8
		}
		dst[d] = e.alphabet[v>>18]
		dst[d+1] = e.alphabet[v>>12&63]
		if len(src) > 1 {
			dst[d+2] = e.alphabet[v>>6&63]
		} else if e.pad != 255 {
			dst[d+2] = e.pad
		}
		if e.pad != 255 {
			dst[d+3] = e.pad
		}
	}
}
func (e *Encoding) EncodeToString(src []byte) string {
	b := make([]byte, e.EncodedLen(len(src)))
	e.Encode(b, src)
	return string(b)
}
func (e *Encoding) DecodedLen(n int) int {
	if e.pad == 255 {
		return n * 6 / 8
	}
	return n / 4 * 3
}
func (e *Encoding) v(c byte) int {
	for i := 0; i < 64; i++ {
		if e.alphabet[i] == c {
			return i
		}
	}
	return -1
}
func (e *Encoding) Decode(dst, src []byte) (int, error) {
	n := 0
	for i := 0; i < len(src); {
		rem := len(src) - i
		if rem < 2 {
			return n, CorruptInputError(i)
		}
		cnt := 4
		if rem < 4 {
			if e.pad != 255 {
				return n, CorruptInputError(i)
			}
			cnt = rem
		}
		var v [4]int
		used := cnt
		if e.pad == 255 {
			for j := 0; j < cnt; j++ {
				if src[i+j] == '=' {
					return n, CorruptInputError(i + j)
				}
			}
		}
		for j := 0; j < cnt; j++ {
			if src[i+j] == '=' {
				used = j
				break
			}
			v[j] = e.v(src[i+j])
			if v[j] < 0 {
				return n, CorruptInputError(i + j)
			}
		}
		if used < 2 {
			return n, CorruptInputError(i)
		}
		x := uint(v[0])<<18 | uint(v[1])<<12 | uint(v[2])<<6 | uint(v[3])
		dst[n] = byte(x >> 16)
		n++
		if used > 2 {
			dst[n] = byte(x >> 8)
			n++
		}
		if used > 3 {
			dst[n] = byte(x)
			n++
		}
		i += cnt
		if used < cnt {
			break
		}
	}
	return n, nil
}
func (e *Encoding) DecodeString(s string) ([]byte, error) {
	b := make([]byte, e.DecodedLen(len(s))+2)
	n, x := e.Decode(b, []byte(s))
	return b[:n], x
}
