package syntax

type Scanner struct {
	Ok     bool
	Tokens []Token
}

func Scan(src []byte) []Token {
	var tokens []Token
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
				break
			}
			i += 2
			continue
		}
		if isIdentStart(c) {
			start := i
			i++
			for i < len(src) && isIdentPart(src[i]) {
				i++
			}
			tokens = appendScanToken(tokens, keywordKind(src, start, i), start, i, line)
			continue
		}
		if c >= '0' && c <= '9' {
			start := i
			kind := TokenNumber
			i++
			for i < len(src) && isIdentPart(src[i]) {
				i++
			}
			if i < len(src) && src[i] == '.' {
				i++
				for i < len(src) && isIdentPart(src[i]) {
					i++
				}
			}
			tokens = appendScanToken(tokens, kind, start, i, line)
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
			if i < len(src) && src[i] == '"' {
				i++
			}
			tokens = appendScanToken(tokens, TokenString, start, i, line)
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
			if i < len(src) && src[i] == '`' {
				i++
			}
			tokens = appendScanToken(tokens, TokenString, start, i, tokenLine)
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
			if i < len(src) && src[i] == '\'' {
				i++
			}
			tokens = appendScanToken(tokens, TokenChar, start, i, line)
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
		tokens = appendScanToken(tokens, TokenOperator, start, i, line)
	}
	tokens = appendScanToken(tokens, TokenEOF, len(src), len(src), line)
	return tokens
}

func (s *Scanner) Scan(src []byte) {
	s.Tokens = nil
	s.Ok = true
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
				s.Ok = false
				break
			}
			i += 2
			continue
		}
		if isIdentStart(c) {
			start := i
			i++
			for i < len(src) && isIdentPart(src[i]) {
				i++
			}
			s.add(keywordKind(src, start, i), start, i, line)
			continue
		}
		if c >= '0' && c <= '9' {
			start := i
			kind := TokenNumber
			if c == '0' && i+1 < len(src) && (src[i+1] == 'x' || src[i+1] == 'X' || src[i+1] == 'b' || src[i+1] == 'B' || src[i+1] == 'o' || src[i+1] == 'O') {
				hex := src[i+1] == 'x' || src[i+1] == 'X'
				i += 2
				for i < len(src) {
					cc := src[i]
					if cc == '.' && hex {
						i++
						continue
					}
					if hex && (cc == 'p' || cc == 'P') {
						i++
						if i < len(src) && (src[i] == '+' || src[i] == '-') {
							i++
						}
						for i < len(src) && isDigitOrUnderscore(src[i]) {
							i++
						}
						break
					}
					if !isIdentPart(cc) {
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
			s.add(kind, start, i, line)
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
					s.Ok = false
					break
				}
				i++
			}
			if i >= len(src) || src[i] != '"' {
				s.Ok = false
				break
			}
			i++
			s.add(TokenString, start, i, line)
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
				s.Ok = false
				break
			}
			i++
			s.add(TokenString, start, i, tokenLine)
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
					s.Ok = false
					break
				}
				i++
			}
			if i >= len(src) || src[i] != '\'' {
				s.Ok = false
				break
			}
			i++
			s.add(TokenChar, start, i, line)
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
		s.add(TokenOperator, start, i, line)
	}
	s.add(TokenEOF, len(src), len(src), line)
}

func appendScanToken(tokens []Token, kind int, start int, end int, line int) []Token {
	var tok Token
	tok.Kind = kind
	tok.Start = start
	tok.End = end
	tok.Line = line
	return append(tokens, tok)
}

func (s *Scanner) add(kind int, start int, end int, line int) {
	s.Tokens = append(s.Tokens, Token{Kind: kind, Start: start, End: end, Line: line})
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

func keywordKind(src []byte, start int, end int) int {
	n := end - start
	if n > 11 {
		return TokenIdent
	}
	h := 0
	for i := start; i < end; i++ {
		h = h*5 + int(src[i])
	}
	if n == 2 {
		if h == 627 {
			return TokenIf
		}
		if h == 626 {
			return TokenGo
		}
	}
	if n == 3 {
		if h == 3549 {
			return TokenVar
		}
		if h == 3219 {
			return TokenFor
		}
		if h == 3322 {
			return TokenMap
		}
	}
	if n == 4 {
		if h == 18186 {
			return TokenType
		}
		if h == 16324 {
			return TokenFunc
		}
		if h == 16001 {
			return TokenElse
		}
		if h == 16341 {
			return TokenGoto
		}
		if h == 15476 {
			return TokenCase
		}
		if h == 15570 {
			return TokenChan
		}
	}
	if n == 5 {
		if h == 79191 {
			return TokenConst
		}
		if h == 78617 {
			return TokenBreak
		}
		if h == 86741 {
			return TokenRange
		}
		if h == 78294 {
			return TokenDefer
		}
	}
	if n == 6 {
		if h == 449661 {
			return TokenStruct
		}
		if h == 437480 {
			return TokenReturn
		}
		if h == 450374 {
			return TokenSwitch
		}
		if h == 413711 {
			return TokenImport
		}
		if h == 439136 {
			return TokenSelect
		}
	}
	if n == 7 {
		if h == 2131416 {
			return TokenPackage
		}
		if h == 1957581 {
			return TokenDefault
		}
	}
	if n == 8 {
		if h == 9901561 {
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
