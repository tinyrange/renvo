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
		if line > TokenLineLimit {
			ok = false
			break
		}
		c := src[i]
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			for i < len(src) {
				c = src[i]
				if c == '\n' {
					line++
				} else if c != ' ' && c != '\t' && c != '\r' {
					break
				}
				i++
			}
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
			i++
			for i < len(src) {
				part := src[i]
				if !((part >= 'a' && part <= 'z') || (part >= 'A' && part <= 'Z') || (part >= '0' && part <= '9') || part == '_') {
					break
				}
				i++
			}
			kind := TokenIdent
			size := i - start
			if size >= 2 && size <= 9 || size == 11 {
				kind = keywordKind(src, start, i, c)
			}
			tokens = append(tokens, MakeToken(kind, start, i, line))
			continue
		}
		if c >= '0' && c <= '9' {
			start := i
			i = scanNumberEnd(src, i)
			tokens = append(tokens, MakeToken(TokenNumber, start, i, line))
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
			for escape := start + 1; escape < i; escape++ {
				if src[escape] != '\\' {
					continue
				}
				next, _, _, valid := stringEscapeValue(src, escape, i)
				if !valid {
					ok = false
					break
				}
				escape = next - 1
			}
			if !ok {
				break
			}
			i++
			tokens = append(tokens, MakeToken(TokenString, start, i, line))
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
			tokens = append(tokens, MakeToken(TokenString, start, i, tokenLine))
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
			tokens = append(tokens, MakeToken(TokenChar, start, i, line))
			continue
		}
		start := i
		i++
		if i < len(src) {
			b := src[i]
			two := false
			if b == '=' {
				two = c == ':' || c == '=' || c == '!' || c == '<' || c == '>' || c == '+' || c == '-' || c == '*' || c == '/' || c == '%' || c == '&' || c == '|' || c == '^'
			}
			if c == '&' && (b == '&' || b == '^') {
				two = true
			}
			if c == '|' && b == '|' || c == '<' && (b == '<' || b == '-') || c == '>' && b == '>' {
				two = true
			}
			if c == '+' && b == '+' || c == '-' && b == '-' || c == '.' && b == '.' {
				two = true
			}
			if two {
				i++
				if i < len(src) && isThreeByteOperator(c, b, src[i]) {
					i++
				}
			}
		}
		tok := MakeToken(TokenOperator, start, i, line)
		if i == start+1 && c <= TokenOperatorCharMask {
			tok.KindLine = tok.KindLine | int(c)<<TokenOperatorCharShift
		}
		tokens = append(tokens, tok)
	}
	if line > TokenLineLimit {
		ok = false
	}
	tokens = append(tokens, MakeToken(TokenEOF, len(src), len(src), line))
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
	capacity := 0
	// Large linked programs are less token-dense than individual source files.
	// Keep the wider estimate for small and hand-minified inputs, but avoid
	// permanently touching an oversized token arena in no-GC self-hosted runs.
	if len(src) >= 262144 {
		capacity = len(src) / 5
		capacity += capacity / 10
	} else {
		capacity = len(src) / 4
	}
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

func keywordKind(src []byte, start int, end int, first byte) int {
	n := end - start
	if n > 11 {
		return TokenIdent
	}
	last := src[end-1]
	if n == 2 {
		if first == 'i' && last == 'f' && bytesEqualText(src, start, end, "if") {
			return TokenIf
		}
		if first == 'g' && last == 'o' && bytesEqualText(src, start, end, "go") {
			return TokenGo
		}
		return TokenIdent
	}
	if n == 3 {
		if first == 'v' && last == 'r' && bytesEqualText(src, start, end, "var") {
			return TokenVar
		}
		if first == 'f' && last == 'r' && bytesEqualText(src, start, end, "for") {
			return TokenFor
		}
		if first == 'm' && last == 'p' && bytesEqualText(src, start, end, "map") {
			return TokenMap
		}
		return TokenIdent
	}
	if n == 4 {
		if first == 't' && last == 'e' && bytesEqualText(src, start, end, "type") {
			return TokenType
		}
		if first == 'f' && last == 'c' && bytesEqualText(src, start, end, "func") {
			return TokenFunc
		}
		if first == 'e' && last == 'e' && bytesEqualText(src, start, end, "else") {
			return TokenElse
		}
		if first == 'g' && last == 'o' && bytesEqualText(src, start, end, "goto") {
			return TokenGoto
		}
		if first == 'c' && last == 'e' && bytesEqualText(src, start, end, "case") {
			return TokenCase
		}
		if first == 'c' && last == 'n' && bytesEqualText(src, start, end, "chan") {
			return TokenChan
		}
		return TokenIdent
	}
	if n == 5 {
		if first == 'c' && last == 't' && bytesEqualText(src, start, end, "const") {
			return TokenConst
		}
		if first == 'b' && last == 'k' && bytesEqualText(src, start, end, "break") {
			return TokenBreak
		}
		if first == 'r' && last == 'e' && bytesEqualText(src, start, end, "range") {
			return TokenRange
		}
		if first == 'd' && last == 'r' && bytesEqualText(src, start, end, "defer") {
			return TokenDefer
		}
		return TokenIdent
	}
	if n == 6 {
		if first == 's' && last == 't' && bytesEqualText(src, start, end, "struct") {
			return TokenStruct
		}
		if first == 'r' && last == 'n' && bytesEqualText(src, start, end, "return") {
			return TokenReturn
		}
		if first == 's' && last == 'h' && bytesEqualText(src, start, end, "switch") {
			return TokenSwitch
		}
		if first == 'i' && last == 't' && bytesEqualText(src, start, end, "import") {
			return TokenImport
		}
		if first == 's' && last == 't' && bytesEqualText(src, start, end, "select") {
			return TokenSelect
		}
		return TokenIdent
	}
	if n == 7 {
		if first == 'p' && last == 'e' && bytesEqualText(src, start, end, "package") {
			return TokenPackage
		}
		if first == 'd' && last == 't' && bytesEqualText(src, start, end, "default") {
			return TokenDefault
		}
		return TokenIdent
	}
	if n == 8 {
		if first == 'c' && last == 'e' && bytesEqualText(src, start, end, "continue") {
			return TokenContinue
		}
		return TokenIdent
	}
	if n == 9 {
		if first == 'i' && last == 'e' && bytesEqualText(src, start, end, "interface") {
			return TokenInterface
		}
		return TokenIdent
	}
	if n == 11 {
		if first == 'f' && last == 'h' && bytesEqualText(src, start, end, "fallthrough") {
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
