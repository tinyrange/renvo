// Package base64 implements the standard and URL-safe Base64 alphabets.
package base64

import "strconv"

const StdPadding rune = '='
const NoPadding rune = -1

type Encoding struct {
	alphabet string
	decode   [256]int
	padding  rune
	strict   bool
}

func NewEncoding(alphabet string) *Encoding {
	if len(alphabet) != 64 {
		panic("base64: alphabet must contain 64 bytes")
	}
	e := &Encoding{alphabet: alphabet, padding: StdPadding}
	for i := 0; i < 256; i++ {
		e.decode[i] = -1
	}
	for i := 0; i < 64; i++ {
		b := alphabet[i]
		if b == '\n' || b == '\r' || e.decode[b] >= 0 {
			panic("base64: invalid alphabet")
		}
		e.decode[b] = i
	}
	return e
}

func (e Encoding) WithPadding(padding rune) *Encoding {
	if padding != NoPadding && (padding < 0 || padding > 255 || padding == '\n' || padding == '\r' || e.decode[byte(padding)] >= 0) {
		panic("base64: invalid padding")
	}
	e.padding = padding
	return &e
}
func (e Encoding) Strict() *Encoding { e.strict = true; return &e }

var StdEncoding = NewEncoding("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/")
var URLEncoding = NewEncoding("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_")
var RawStdEncoding = StdEncoding.WithPadding(NoPadding)
var RawURLEncoding = URLEncoding.WithPadding(NoPadding)

func (e *Encoding) EncodedLen(n int) int {
	if e.padding == NoPadding {
		return n/3*4 + (n%3*8+5)/6
	}
	return (n + 2) / 3 * 4
}
func (e *Encoding) DecodedLen(n int) int {
	if e.padding == NoPadding {
		return n/4*3 + n%4*6/8
	}
	return n / 4 * 3
}

func (e *Encoding) Encode(dst, src []byte) {
	at := 0
	for len(src) >= 3 {
		v := uint32(src[0])<<16 | uint32(src[1])<<8 | uint32(src[2])
		dst[at] = e.alphabet[v>>18]
		dst[at+1] = e.alphabet[(v>>12)&63]
		dst[at+2] = e.alphabet[(v>>6)&63]
		dst[at+3] = e.alphabet[v&63]
		at += 4
		src = src[3:]
	}
	if len(src) == 0 {
		return
	}
	v := uint32(src[0]) << 16
	if len(src) == 2 {
		v |= uint32(src[1]) << 8
	}
	dst[at] = e.alphabet[v>>18]
	dst[at+1] = e.alphabet[(v>>12)&63]
	if len(src) == 2 {
		dst[at+2] = e.alphabet[(v>>6)&63]
	} else if e.padding != NoPadding {
		dst[at+2] = byte(e.padding)
	}
	if e.padding != NoPadding {
		dst[at+3] = byte(e.padding)
	}
}
func (e *Encoding) AppendEncode(dst, src []byte) []byte {
	at := len(dst)
	dst = append(dst, make([]byte, e.EncodedLen(len(src)))...)
	e.Encode(dst[at:], src)
	return dst
}
func (e *Encoding) EncodeToString(src []byte) string { return string(e.AppendEncode(nil, src)) }

type CorruptInputError int64

func (err CorruptInputError) Error() string {
	return "illegal base64 data at input byte " + strconv.FormatInt(int64(err), 10)
}

func (e *Encoding) Decode(dst, src []byte) (int, error) {
	out := 0
	at := 0
	for {
		var digits [4]byte
		count := 0
		padded := false
		for count < 4 {
			if at == len(src) {
				if count == 0 {
					return out, nil
				}
				if e.padding != NoPadding || count == 1 {
					return out, CorruptInputError(at - count)
				}
				break
			}
			b := src[at]
			at++
			if b == '\r' || b == '\n' {
				continue
			}
			if e.padding != NoPadding && rune(b) == e.padding {
				if count < 2 {
					return out, CorruptInputError(at - 1)
				}
				padded = true
				if count == 2 {
					for at < len(src) && (src[at] == '\r' || src[at] == '\n') {
						at++
					}
					if at == len(src) {
						return out, CorruptInputError(at)
					}
					if rune(src[at]) != e.padding {
						return out, CorruptInputError(at - 1)
					}
					at++
				}
				break
			}
			value := e.decode[b]
			if value < 0 {
				return out, CorruptInputError(at - 1)
			}
			digits[count] = byte(value)
			count++
		}
		if e.strict && (count == 2 && digits[1]&15 != 0 || count == 3 && digits[2]&3 != 0) {
			return out, CorruptInputError(at - 1)
		}
		dst[out] = digits[0]<<2 | digits[1]>>4
		out++
		if count >= 3 {
			dst[out] = digits[1]<<4 | digits[2]>>2
			out++
		}
		if count == 4 {
			dst[out] = digits[2]<<6 | digits[3]
			out++
		}
		if padded {
			for at < len(src) && (src[at] == '\r' || src[at] == '\n') {
				at++
			}
			if at != len(src) {
				return out, CorruptInputError(at)
			}
			return out, nil
		}
		if count < 4 {
			return out, nil
		}
	}
}

func (e *Encoding) AppendDecode(dst, src []byte) ([]byte, error) {
	at := len(dst)
	// Ignored newlines can only reduce the decoded output length.
	dst = append(dst, make([]byte, e.DecodedLen(len(src)))...)
	n, err := e.Decode(dst[at:], src)
	return dst[:at+n], err
}
func (e *Encoding) DecodeString(s string) ([]byte, error) { return e.AppendDecode(nil, []byte(s)) }
