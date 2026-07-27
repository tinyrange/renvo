package rtg

func scan(source []byte, filename string) ([]Token, []Diagnostic) {
	tokens := make([]Token, 0, len(source)/5+1)
	var diagnostics []Diagnostic
	offset := 0
	line := 1
	column := 1
	for offset < len(source) {
		ch := source[offset]
		if ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n' {
			if ch == '\n' {
				line++
				column = 1
			} else {
				column++
			}
			offset++
			continue
		}
		if ch == '#' {
			offset, line, column = skipLineComment(source, offset, line, column)
			continue
		}
		if ch == '/' && offset+1 < len(source) && source[offset+1] == '/' {
			offset, line, column = skipLineComment(source, offset, line, column)
			continue
		}
		if ch == '/' && offset+1 < len(source) && source[offset+1] == '*' {
			start := offset
			startLine := line
			startColumn := column
			offset += 2
			column += 2
			closed := false
			for offset < len(source) {
				if source[offset] == '*' && offset+1 < len(source) && source[offset+1] == '/' {
					offset += 2
					column += 2
					closed = true
					break
				}
				if source[offset] == '\n' {
					line++
					column = 1
				} else {
					column++
				}
				offset++
			}
			if !closed {
				diagnostics = append(diagnostics, scanDiagnostic(filename, source, start, len(source), "RTG-SCAN-001", "unterminated block comment"))
				tokens = append(tokens, Token{Kind: TokenInvalid, Start: start, End: len(source), Line: startLine, Column: startColumn})
				break
			}
			continue
		}
		start := offset
		startLine := line
		startColumn := column
		if isIdentStart(ch) {
			offset++
			column++
			for offset < len(source) && isIdentPart(source[offset]) {
				offset++
				column++
			}
			tokens = append(tokens, Token{Kind: TokenIdent, Start: start, End: offset, Line: startLine, Column: startColumn})
			continue
		}
		if ch >= '0' && ch <= '9' {
			offset++
			column++
			for offset < len(source) {
				next := source[offset]
				if !isIdentPart(next) && next != '.' {
					break
				}
				offset++
				column++
			}
			tokens = append(tokens, Token{Kind: TokenNumber, Start: start, End: offset, Line: startLine, Column: startColumn})
			continue
		}
		if ch == '"' || ch == '\'' || ch == '`' {
			quote := ch
			offset++
			column++
			closed := false
			for offset < len(source) {
				next := source[offset]
				if quote != '`' && next == '\\' {
					if offset+1 >= len(source) || source[offset+1] == '\n' {
						break
					}
					offset += 2
					column += 2
					continue
				}
				if next == quote {
					offset++
					column++
					closed = true
					break
				}
				if next == '\n' {
					if quote != '`' {
						break
					}
					line++
					column = 1
				} else {
					column++
				}
				offset++
			}
			if !closed {
				diagnostics = append(diagnostics, scanDiagnostic(filename, source, start, offset, "RTG-SCAN-002", "unterminated quoted literal"))
				tokens = append(tokens, Token{Kind: TokenInvalid, Start: start, End: offset, Line: startLine, Column: startColumn})
				break
			}
			tokens = append(tokens, Token{Kind: TokenString, Start: start, End: offset, Line: startLine, Column: startColumn})
			continue
		}
		offset++
		column++
		if offset < len(source) && ((ch == '=' && source[offset] == '>') ||
			(ch == '<' && source[offset] == '<') ||
			(ch == '>' && source[offset] == '>') ||
			(ch == '&' && source[offset] == '&') ||
			(ch == '|' && source[offset] == '|') ||
			(ch == '!' && source[offset] == '=') ||
			(ch == '=' && source[offset] == '=') ||
			(ch == '<' && source[offset] == '=') ||
			(ch == '>' && source[offset] == '=')) {
			offset++
			column++
		}
		tokens = append(tokens, Token{Kind: TokenOperator, Start: start, End: offset, Line: startLine, Column: startColumn})
	}
	tokens = append(tokens, Token{Kind: TokenEOF, Start: len(source), End: len(source), Line: line, Column: column})
	return tokens, diagnostics
}

func skipLineComment(source []byte, offset int, line int, column int) (int, int, int) {
	for offset < len(source) && source[offset] != '\n' {
		offset++
		column++
	}
	return offset, line, column
}

func isIdentStart(ch byte) bool {
	return ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch == '_'
}

func isIdentPart(ch byte) bool {
	return isIdentStart(ch) || ch >= '0' && ch <= '9'
}

func scanDiagnostic(filename string, source []byte, start int, end int, code string, message string) Diagnostic {
	return Diagnostic{Filename: filename, Span: sourceSpan(source, start, end), Code: code, Message: message}
}
