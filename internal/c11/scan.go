package c11

const (
	ScanOK = iota
	ScanErrByte
	ScanErrComment
	ScanErrLiteral
)

type scanResult struct {
	tokens  []token
	ok      bool
	err     int
	errorAt int
}

// scan keeps only byte spans into src. In particular, it does not normalize
// or copy the complete input before tokenization.
func scan(src []byte) scanResult {
	result := scanResult{ok: true, errorAt: -1}
	result.tokens = make([]token, 0, scanCapacity(src))
	i := 0
	line := 1
	for i < len(src) {
		c := src[i]
		if c == ' ' || c == '\t' || c == '\r' || c == '\v' || c == '\f' {
			i++
			continue
		}
		if c == '\n' {
			line++
			i++
			continue
		}
		if c == '\\' && i+1 < len(src) && src[i+1] == '\n' {
			line++
			i += 2
			continue
		}
		if c == '/' && i+1 < len(src) && src[i+1] == '/' {
			i += 2
			for i < len(src) && src[i] != '\n' {
				i++
			}
			continue
		}
		if c == '/' && i+1 < len(src) && src[i+1] == '*' {
			commentStart := i
			i += 2
			closed := false
			for i+1 < len(src) {
				if src[i] == '\n' {
					line++
				}
				if src[i] == '*' && src[i+1] == '/' {
					i += 2
					closed = true
					break
				}
				i++
			}
			if !closed {
				return scanFail(result, ScanErrComment, commentStart, line, len(src))
			}
			continue
		}
		start := i
		if isIdentStart(c) {
			i++
			for i < len(src) && isIdentPart(src[i]) {
				i++
			}
			// C literal prefixes are part of the literal token.
			if i < len(src) && (src[i] == '"' || src[i] == '\'') && literalPrefix(src[start:i]) {
				delim := src[i]
				i++
				end, ok := scanQuoted(src, i, delim)
				if !ok {
					return scanFail(result, ScanErrLiteral, start, line, len(src))
				}
				i = end
				kind := tokenString
				if delim == '\'' {
					kind = tokenChar
				}
				result.tokens = append(result.tokens, makeToken(kind, start, i, line))
				continue
			}
			result.tokens = append(result.tokens, makeToken(tokenIdent, start, i, line))
			continue
		}
		if isDigit(c) || c == '.' && i+1 < len(src) && isDigit(src[i+1]) {
			i = scanNumber(src, i)
			result.tokens = append(result.tokens, makeToken(tokenNumber, start, i, line))
			continue
		}
		if c == '"' || c == '\'' {
			i++
			end, ok := scanQuoted(src, i, c)
			if !ok {
				return scanFail(result, ScanErrLiteral, start, line, len(src))
			}
			i = end
			kind := tokenString
			if c == '\'' {
				kind = tokenChar
			}
			result.tokens = append(result.tokens, makeToken(kind, start, i, line))
			continue
		}
		end := punctEnd(src, i)
		if end == i {
			return scanFail(result, ScanErrByte, i, line, len(src))
		}
		i = end
		result.tokens = append(result.tokens, makeToken(tokenPunct, start, i, line))
	}
	result.tokens = append(result.tokens, makeToken(tokenEOF, len(src), len(src), line))
	return result
}

func scanFail(result scanResult, err int, at int, line int, sourceEnd int) scanResult {
	result.ok = false
	result.err = err
	result.errorAt = at
	result.tokens = append(result.tokens, makeToken(tokenEOF, sourceEnd, sourceEnd, line))
	return result
}

func scanCapacity(src []byte) int {
	capacity := len(src)/4 + 8
	if capacity < 16 {
		return 16
	}
	return capacity
}

func isIdentStart(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

func isIdentPart(c byte) bool { return isIdentStart(c) || isDigit(c) }
func isDigit(c byte) bool     { return c >= '0' && c <= '9' }

func literalPrefix(prefix []byte) bool {
	return len(prefix) == 1 && (prefix[0] == 'L' || prefix[0] == 'u' || prefix[0] == 'U') ||
		len(prefix) == 2 && prefix[0] == 'u' && prefix[1] == '8'
}

func scanQuoted(src []byte, i int, delim byte) (int, bool) {
	for i < len(src) {
		if src[i] == '\n' {
			return i, false
		}
		if src[i] == '\\' {
			i += 2
			continue
		}
		if src[i] == delim {
			return i + 1, true
		}
		i++
	}
	return i, false
}

func scanNumber(src []byte, i int) int {
	previous := byte(0)
	for i < len(src) {
		c := src[i]
		if isIdentPart(c) || c == '.' {
			previous = c
			i++
			continue
		}
		if (c == '+' || c == '-') && (previous == 'e' || previous == 'E' || previous == 'p' || previous == 'P') {
			previous = c
			i++
			continue
		}
		break
	}
	return i
}

func punctEnd(src []byte, i int) int {
	if i >= len(src) || !punctByte(src[i]) {
		return i
	}
	if i+3 < len(src) && src[i] == '%' && src[i+1] == ':' && src[i+2] == '%' && src[i+3] == ':' {
		return i + 4
	}
	if i+2 < len(src) {
		a, b, c := src[i], src[i+1], src[i+2]
		if a == '.' && b == '.' && c == '.' || (a == '<' || a == '>') && b == a && c == '=' {
			return i + 3
		}
	}
	if i+1 < len(src) {
		a, b := src[i], src[i+1]
		if a == '-' && (b == '>' || b == '-') || a == '+' && b == '+' ||
			(a == '<' || a == '>' || a == '=' || a == '!' || a == '&' || a == '|' || a == '^' || a == '*' || a == '/' || a == '%' || a == '+' || a == '-') && b == '=' ||
			a == '<' && b == '<' || a == '>' && b == '>' || a == '&' && b == '&' || a == '|' && b == '|' || a == '#' && b == '#' {
			return i + 2
		}
	}
	return i + 1
}

func punctByte(c byte) bool {
	for i := 0; i < len("[](){}.,;:?~!%^&*+-=/<>|#"); i++ {
		if c == "[](){}.,;:?~!%^&*+-=/<>|#"[i] {
			return true
		}
	}
	return false
}
