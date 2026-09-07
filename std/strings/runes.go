package strings

import (
	"unicode"
	"unicode/utf8"
)

func Map(mapping func(rune) rune, s string) string {
	var out []byte
	for _, r := range s {
		mapped := mapping(r)
		if mapped < 0 {
			continue
		}
		var encoded [utf8.UTFMax]byte
		n := utf8.EncodeRune(encoded[:], mapped)
		out = append(out, encoded[:n]...)
	}
	return string(out)
}
func ToLower(s string) string { return Map(unicode.ToLower, s) }
func ToUpper(s string) string { return Map(unicode.ToUpper, s) }
func ToTitle(s string) string { return Map(unicode.ToTitle, s) }
func EqualFold(s, t string) bool {
	for len(s) > 0 && len(t) > 0 {
		a, n := utf8.DecodeRuneInString(s)
		b, m := utf8.DecodeRuneInString(t)
		s = s[n:]
		t = t[m:]
		if a == b {
			continue
		}
		found := false
		for next := unicode.SimpleFold(a); next != a; next = unicode.SimpleFold(next) {
			if next == b {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return len(s) == 0 && len(t) == 0
}
func IndexFunc(s string, f func(rune) bool) int {
	for i, r := range s {
		if f(r) {
			return i
		}
	}
	return -1
}
func LastIndexFunc(s string, f func(rune) bool) int {
	end := len(s)
	for end > 0 {
		r, n := utf8.DecodeLastRuneInString(s[:end])
		end -= n
		if f(r) {
			return end
		}
	}
	return -1
}
func TrimLeftFunc(s string, f func(rune) bool) string {
	start := 0
	for start < len(s) {
		r, n := utf8.DecodeRuneInString(s[start:])
		if !f(r) {
			break
		}
		start += n
	}
	return s[start:]
}
func TrimRightFunc(s string, f func(rune) bool) string {
	end := len(s)
	for end > 0 {
		r, n := utf8.DecodeLastRuneInString(s[:end])
		if !f(r) {
			break
		}
		end -= n
	}
	return s[:end]
}
func TrimFunc(s string, f func(rune) bool) string { return TrimRightFunc(TrimLeftFunc(s, f), f) }
func TrimRight(s, cutset string) string {
	end := len(s)
	for end > 0 {
		r, n := utf8.DecodeLastRuneInString(s[:end])
		if !ContainsRune(cutset, r) {
			break
		}
		end -= n
	}
	return s[:end]
}
func Trim(s, cutset string) string { return TrimRight(TrimLeft(s, cutset), cutset) }
func FieldsFunc(s string, f func(rune) bool) []string {
	out := make([]string, 0)
	start := -1
	for i, r := range s {
		if f(r) {
			if start >= 0 {
				out = append(out, s[start:i])
				start = -1
			}
		} else if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		out = append(out, s[start:])
	}
	return out
}

func IndexRune(s string, r rune) int {
	if r < 0 || r > utf8.MaxRune {
		return -1
	}
	for i := 0; i < len(s); {
		value, size := utf8.DecodeRuneInString(s[i:])
		if value == r {
			return i
		}
		i += size
	}
	return -1
}

func ContainsRune(s string, r rune) bool { return IndexRune(s, r) >= 0 }

func TrimLeft(s string, cutset string) string {
	start := 0
	for start < len(s) {
		r, size := utf8.DecodeRuneInString(s[start:])
		if !ContainsRune(cutset, r) {
			break
		}
		start += size
	}
	return s[start:]
}

func ContainsAny(s string, chars string) bool { return IndexAny(s, chars) >= 0 }

func IndexAny(s string, chars string) int {
	for i := 0; i < len(s); {
		value, size := utf8.DecodeRuneInString(s[i:])
		if ContainsRune(chars, value) {
			return i
		}
		i += size
	}
	return -1
}
