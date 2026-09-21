package syntax

// RuneLiteralValue decodes exactly one rune, including numeric byte escapes.
// Unlike a string, a rune permits an escaped single quote but not an escaped
// double quote. Invalid UTF-8, Unicode scalars, and multi-rune bodies fail.
func RuneLiteralValue(src []byte, tok Token) (int, bool) {
	start, end := int(tok.Start), int(tok.End)
	if tok.KindLine&255 != TokenChar || start < 0 || end > len(src) || end-start < 3 || src[start] != '\'' || src[end-1] != '\'' {
		return 0, false
	}
	start++
	end--
	if src[start] == '\\' {
		if start+1 >= end || src[start+1] == '"' {
			return 0, false
		}
		if src[start+1] == '\'' {
			return int('\''), start+2 == end
		}
		next, value, _, ok := stringEscapeValue(src, start, end)
		return value, ok && next == end
	}
	if src[start] == '\n' || src[start] == '\'' || !validSourceEncoding(src[start:end]) {
		return 0, false
	}
	value, width := identifierRune(src, start)
	return value, width > 0 && start+width == end
}

// Source encoding has already been validated before scanner calls this helper.
func identifierRune(src []byte, start int) (int, int) {
	if start < 0 || start >= len(src) {
		return 0, 0
	}
	c := src[start]
	if c < 128 {
		return int(c), 1
	}
	width := 2
	value := int(c & 31)
	if c >= 240 {
		width = 4
		value = int(c & 7)
	} else if c >= 224 {
		width = 3
		value = int(c & 15)
	}
	if start+width > len(src) {
		return 0, 0
	}
	for i := 1; i < width; i++ {
		value = value*64 + int(src[start+i]&63)
	}
	return value, width
}
