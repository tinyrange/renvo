package syntax

type Scanner struct {
	Ok     bool
	Tokens []Token
}

func Scan(src []byte) []Token {
	tokens, _ := scanTokens(src)
	return tokens
}

func scanTokens(src []byte) ([]Token, bool) {
	tokens := make([]Token, 0, scanTokenCapacity(src))
	ok := true
	i := 0
	line := 1
	for i < len(src) {
		c := src[i]
		if c == ' ' || c == '\t' || c == '\r' {
			i++
			continue
		}
		if c == '\n' {
			line++
			i++
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
			i += 2
			for i+1 < len(src) && !(src[i] == '*' && src[i+1] == '/') {
				if src[i] == '\n' {
					line++
				}
				i++
			}
			if i+1 >= len(src) {
				ok = false
				break
			}
			i += 2
			continue
		}
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' {
			start := i
			hash := int(c)
			i++
			for i < len(src) {
				part := src[i]
				if !((part >= 'a' && part <= 'z') || (part >= 'A' && part <= 'Z') || (part >= '0' && part <= '9') || part == '_') {
					break
				}
				hash = hash*5 + int(part)
				i++
			}
			tokens = append(tokens, Token{Kind: keywordKindHash(src, start, i, hash), Start: start, End: i, Line: line})
			continue
		}
		if c >= '0' && c <= '9' {
			start := i
			i = scanNumberEnd(src, i)
			tokens = append(tokens, Token{Kind: TokenNumber, Start: start, End: i, Line: line})
			continue
		}
		if c == '"' {
			start := i
			i++
			for i < len(src) && src[i] != '"' {
				if src[i] == '\\' && i+1 < len(src) {
					i += 2
					continue
				}
				if src[i] == '\n' {
					break
				}
				i++
			}
			if i >= len(src) || src[i] != '"' {
				ok = false
				break
			}
			i++
			tokens = append(tokens, Token{Kind: TokenString, Start: start, End: i, Line: line})
			continue
		}
		if c == '`' {
			start := i
			tokenLine := line
			i++
			for i < len(src) && src[i] != '`' {
				if src[i] == '\n' {
					line++
				}
				i++
			}
			if i >= len(src) || src[i] != '`' {
				ok = false
				break
			}
			i++
			tokens = append(tokens, Token{Kind: TokenString, Start: start, End: i, Line: tokenLine})
			continue
		}
		if c == '\'' {
			start := i
			i++
			for i < len(src) && src[i] != '\'' {
				if src[i] == '\\' && i+1 < len(src) {
					i += 2
					continue
				}
				if src[i] == '\n' {
					break
				}
				i++
			}
			if i >= len(src) || src[i] != '\'' {
				ok = false
				break
			}
			i++
			tokens = append(tokens, Token{Kind: TokenChar, Start: start, End: i, Line: line})
			continue
		}
		start := i
		i++
		if i < len(src) && isTwoByteOperator(c, src[i]) {
			i++
			if i < len(src) && isThreeByteOperator(c, src[start+1], src[i]) {
				i++
			}
		}
		tokens = append(tokens, Token{Kind: TokenOperator, Start: start, End: i, Line: line})
	}
	tokens = append(tokens, Token{Kind: TokenEOF, Start: len(src), End: len(src), Line: line})
	return tokens, ok
}

func (s *Scanner) Scan(src []byte) {
	s.Tokens, s.Ok = scanTokens(src)
}

func scanNumberEnd(src []byte, start int) int {
	i := start
	if src[i] == '0' && i+1 < len(src) && (src[i+1] == 'x' || src[i+1] == 'X' || src[i+1] == 'b' || src[i+1] == 'B' || src[i+1] == 'o' || src[i+1] == 'O') {
		hex := src[i+1] == 'x' || src[i+1] == 'X'
		i += 2
		for i < len(src) {
			c := src[i]
			if c == '.' && hex {
				i++
				continue
			}
			if hex && (c == 'p' || c == 'P') {
				i++
				if i < len(src) && (src[i] == '+' || src[i] == '-') {
					i++
				}
				for i < len(src) && isDigitOrUnderscore(src[i]) {
					i++
				}
				break
			}
			if !isIdentPart(c) {
				break
			}
			i++
		}
	} else {
		i++
		for i < len(src) && isDigitOrUnderscore(src[i]) {
			i++
		}
		if i < len(src) && src[i] == '.' {
			i++
			for i < len(src) && isDigitOrUnderscore(src[i]) {
				i++
			}
		}
		if i < len(src) && (src[i] == 'e' || src[i] == 'E') {
			i++
			if i < len(src) && (src[i] == '+' || src[i] == '-') {
				i++
			}
			for i < len(src) && isDigitOrUnderscore(src[i]) {
				i++
			}
		}
	}
	if i < len(src) && src[i] == 'i' {
		i++
	}
	return i
}

func scanTokenCapacity(src []byte) int {
	capacity := len(src) / 4
	if capacity < 16 {
		return 16
	}
	return capacity + 16
}

func isIdentStart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

func isIdentPart(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}

func isDigitOrUnderscore(c byte) bool {
	return (c >= '0' && c <= '9') || c == '_'
}

func isTwoByteOperator(a byte, b byte) bool {
	if b == '=' {
		return a == ':' || a == '=' || a == '!' || a == '<' || a == '>' || a == '+' || a == '-' || a == '*' || a == '/' || a == '%' || a == '&' || a == '|' || a == '^'
	}
	if a == '&' && (b == '&' || b == '^') {
		return true
	}
	if a == '|' && b == '|' {
		return true
	}
	if a == '<' && (b == '<' || b == '-') {
		return true
	}
	if a == '>' && b == '>' {
		return true
	}
	if a == '+' && b == '+' {
		return true
	}
	if a == '-' && b == '-' {
		return true
	}
	if a == '.' && b == '.' {
		return true
	}
	return false
}

func isThreeByteOperator(a byte, b byte, c byte) bool {
	if a == '.' && b == '.' && c == '.' {
		return true
	}
	if b == '=' {
		return false
	}
	if a == '<' && b == '<' && c == '=' {
		return true
	}
	if a == '>' && b == '>' && c == '=' {
		return true
	}
	if a == '&' && b == '^' && c == '=' {
		return true
	}
	return false
}

func keywordKindHash(src []byte, start int, end int, h int) int {
	n := end - start
	if n > 11 {
		return TokenIdent
	}
	if n == 2 {
		if h == 627 && bytesEqualText(src, start, end, "if") {
			return TokenIf
		}
		if h == 626 && bytesEqualText(src, start, end, "go") {
			return TokenGo
		}
	}
	if n == 3 {
		if h == 3549 && bytesEqualText(src, start, end, "var") {
			return TokenVar
		}
		if h == 3219 && bytesEqualText(src, start, end, "for") {
			return TokenFor
		}
		if h == 3322 && bytesEqualText(src, start, end, "map") {
			return TokenMap
		}
	}
	if n == 4 {
		if h == 18186 && bytesEqualText(src, start, end, "type") {
			return TokenType
		}
		if h == 16324 && bytesEqualText(src, start, end, "func") {
			return TokenFunc
		}
		if h == 16001 && bytesEqualText(src, start, end, "else") {
			return TokenElse
		}
		if h == 16341 && bytesEqualText(src, start, end, "goto") {
			return TokenGoto
		}
		if h == 15476 && bytesEqualText(src, start, end, "case") {
			return TokenCase
		}
		if h == 15570 && bytesEqualText(src, start, end, "chan") {
			return TokenChan
		}
	}
	if n == 5 {
		if h == 79191 && bytesEqualText(src, start, end, "const") {
			return TokenConst
		}
		if h == 78617 && bytesEqualText(src, start, end, "break") {
			return TokenBreak
		}
		if h == 86741 && bytesEqualText(src, start, end, "range") {
			return TokenRange
		}
		if h == 78294 && bytesEqualText(src, start, end, "defer") {
			return TokenDefer
		}
	}
	if n == 6 {
		if h == 449661 && bytesEqualText(src, start, end, "struct") {
			return TokenStruct
		}
		if h == 437480 && bytesEqualText(src, start, end, "return") {
			return TokenReturn
		}
		if h == 450374 && bytesEqualText(src, start, end, "switch") {
			return TokenSwitch
		}
		if h == 413711 && bytesEqualText(src, start, end, "import") {
			return TokenImport
		}
		if h == 439136 && bytesEqualText(src, start, end, "select") {
			return TokenSelect
		}
	}
	if n == 7 {
		if h == 2131416 && bytesEqualText(src, start, end, "package") {
			return TokenPackage
		}
		if h == 1957581 && bytesEqualText(src, start, end, "default") {
			return TokenDefault
		}
	}
	if n == 8 {
		if h == 9901561 && bytesEqualText(src, start, end, "continue") {
			return TokenContinue
		}
	}
	if n == 9 {
		if bytesEqualText(src, start, end, "interface") {
			return TokenInterface
		}
	}
	if n == 11 {
		if bytesEqualText(src, start, end, "fallthrough") {
			return TokenFallthrough
		}
	}
	return TokenIdent
}

func bytesEqualText(src []byte, start int, end int, text string) bool {
	if end-start != len(text) {
		return false
	}
	for i := 0; i < len(text); i++ {
		if src[start+i] != text[i] {
			return false
		}
	}
	return true
}
