package hex

const digits = "0123456789abcdef"

type InvalidByteError byte

func (e InvalidByteError) Error() string { return "encoding/hex: invalid byte" }
func EncodedLen(n int) int               { return n * 2 }
func DecodedLen(n int) int               { return n / 2 }
func Encode(dst, src []byte) int {
	for i, b := range src {
		hi := int(b >> 4)
		lo := int(b & 15)
		dst[2*i] = digits[hi]
		dst[2*i+1] = digits[lo]
	}
	return len(src) * 2
}
func EncodeToString(src []byte) string {
	b := make([]byte, EncodedLen(len(src)))
	Encode(b, src)
	return string(b)
}
func val(c byte) (byte, bool) {
	if c >= '0' && c <= '9' {
		return c - '0', true
	}
	if c >= 'a' && c <= 'f' {
		return c - 'a' + 10, true
	}
	if c >= 'A' && c <= 'F' {
		return c - 'A' + 10, true
	}
	return 0, false
}
func Decode(dst, src []byte) (int, error) {
	n := 0
	for len(src) >= 2 {
		a, ok := val(src[0])
		if !ok {
			return n, InvalidByteError(src[0])
		}
		b, ok := val(src[1])
		if !ok {
			return n, InvalidByteError(src[1])
		}
		dst[n] = a<<4 | b
		n++
		src = src[2:]
	}
	if len(src) > 0 {
		return n, InvalidByteError(src[0])
	}
	return n, nil
}
func DecodeString(s string) ([]byte, error) {
	b := make([]byte, DecodedLen(len(s)))
	n, e := Decode(b, []byte(s))
	return b[:n], e
}
