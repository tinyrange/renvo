// Package utf16 implements UTF-16 encoding and decoding.
package utf16

const replacement = '\ufffd'

func IsSurrogate(r rune) bool { return r >= 0xd800 && r <= 0xdfff }

func DecodeRune(r1, r2 rune) rune {
	if r1 < 0xd800 || r1 > 0xdbff || r2 < 0xdc00 || r2 > 0xdfff {
		return replacement
	}
	return 0x10000 + (r1-0xd800)*0x400 + r2 - 0xdc00
}

func EncodeRune(r rune) (rune, rune) {
	if r < 0x10000 || r > 0x10ffff {
		return replacement, replacement
	}
	r -= 0x10000
	return 0xd800 + r/0x400, 0xdc00 + r%0x400
}

func RuneLen(r rune) int {
	if r < 0 || r > 0x10ffff || IsSurrogate(r) {
		return -1
	}
	if r >= 0x10000 {
		return 2
	}
	return 1
}

func AppendRune(dst []uint16, r rune) []uint16 {
	n := RuneLen(r)
	if n < 0 {
		return append(dst, uint16(replacement))
	}
	if n == 1 {
		return append(dst, uint16(r))
	}
	a, b := EncodeRune(r)
	return append(dst, uint16(a), uint16(b))
}

func Encode(s []rune) []uint16 {
	out := make([]uint16, 0, len(s))
	for _, r := range s {
		out = AppendRune(out, r)
	}
	return out
}

func Decode(s []uint16) []rune {
	out := make([]rune, 0, len(s))
	for i := 0; i < len(s); i++ {
		r := rune(s[i])
		if r >= 0xd800 && r <= 0xdbff && i+1 < len(s) && s[i+1] >= 0xdc00 && s[i+1] <= 0xdfff {
			r = DecodeRune(r, rune(s[i+1]))
			i++
		} else if IsSurrogate(r) {
			r = replacement
		}
		out = append(out, r)
	}
	return out
}
