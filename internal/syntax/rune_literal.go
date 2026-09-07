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
