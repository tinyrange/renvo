package scan

import "strconv"

const (
	Ident  = 1
	Number = 2
	String = 3
	Char   = 4
	Op     = 5
	EOF    = 6
)

type Token struct {
	Kind   int
	Text   string
	Start  int32
	End    int32
	Line   int32
	Column int32
}

type scanError string

func (e scanError) Error() string {
	return string(e)
}

func token(kind int, text string, start int, end int, line int, column int) Token {
	return Token{Kind: kind, Text: text, Start: int32(start), End: int32(end), Line: int32(line), Column: int32(column)}
}

func posError(line int, column int, message string) error {
	lineText := strconv.Itoa(line)
	columnText := strconv.Itoa(column)
	text := lineText + ":" + columnText + ": " + message
	return scanError(text)
}

func Tokens(src []byte) ([]Token, error) {
	toks := make([]Token, 0, tokenCapacity(src))
	text := string(src)
	line := 1
	col := 1
	i := 0
	for i < len(src) {
		c := src[i]
		if c == ' ' {
			i++
			col++
			continue
		}
		if c == '\t' {
			i++
			col++
			continue
		}
		if c == '\r' {
			i++
			col++
			continue
		}
		if c == '\n' {
			i++
			line++
			col = 1
			continue
		}
		if c == '/' {
			if i+1 < len(src) {
				next := src[i+1]
				if next == '/' {
					i += 2
					col += 2
					for i < len(src) {
						ch := src[i]
						if ch == '\n' {
							break
						}
						i++
						col++
					}
					continue
				}
			}
		}
		if c == '/' {
			if i+1 < len(src) {
				next := src[i+1]
				if next == '*' {
					startLine := line
					startCol := col
					i += 2
					col += 2
					closed := false
					for i+1 < len(src) {
						ch := src[i]
						nextCh := src[i+1]
						if ch == '*' {
							if nextCh == '/' {
								i += 2
								col += 2
								closed = true
								break
							}
						}
						if ch == '\n' {
							i++
							line++
							col = 1
						} else {
							i++
							col++
						}
					}
					if !closed {
						return nil, posError(startLine, startCol, "unterminated block comment")
					}
					continue
				}
			}
		}
		if isIdentStart(c) {
			start := i
			startLine := line
			startCol := col
			i++
			col++
			for i < len(src) {
				ch := src[i]
				if !isIdent(ch) {
					break
				}
				i++
				col++
			}
			toks = append(toks, token(Ident, text[start:i], start, i, startLine, startCol))
			continue
		}
		if c >= '0' && c <= '9' {
			start := i
			startLine := line
			startCol := col
			i++
			col++
			for i < len(src) {
				ch := src[i]
				if !isIdent(ch) {
					if ch != '.' {
						break
					}
				}
				i++
				col++
			}
			toks = append(toks, token(Number, text[start:i], start, i, startLine, startCol))
			continue
		}
		if c == '"' || c == '`' || c == '\'' {
			start := i
			startLine := line
			startCol := col
			quote := c
			i++
			col++
			for i < len(src) {
				ch := src[i]
				if quote != '`' {
					if ch == '\\' {
						i += 2
						col += 2
						continue
					}
				}
				if ch == quote {
					i++
					col++
					kind := String
					if quote == '\'' {
						kind = Char
					}
					toks = append(toks, token(kind, text[start:i], start, i, startLine, startCol))
					break
				}
				if ch == '\n' {
					i++
					line++
					col = 1
				} else {
					i++
					col++
				}
			}
			if len(toks) == 0 {
				return nil, posError(startLine, startCol, "unterminated literal")
			}
			if int(toks[len(toks)-1].Start) != start {
				return nil, posError(startLine, startCol, "unterminated literal")
			}
			continue
		}
		start := i
		startLine := line
		startCol := col
		width := opWidth(src, i)
		i += width
		col += width
		toks = append(toks, token(Op, text[start:i], start, i, startLine, startCol))
	}
	toks = append(toks, token(EOF, "", len(src), len(src), line, col))
	return toks, nil
}

func tokenCapacity(src []byte) int {
	capacity := len(src)/3 + 32
	if capacity < 64 {
		return 64
	}
	return capacity
}

func UnquoteString(s string) (string, error) {
	if len(s) < 2 {
		return "", scanError("invalid string literal")
	}
	quote := s[0]
	if quote == '`' {
		if s[len(s)-1] != '`' {
			return "", scanError("invalid raw string literal")
		}
		return s[1 : len(s)-1], nil
	}
	if quote != '"' || s[len(s)-1] != '"' {
		return "", scanError("invalid string literal")
	}
	var out []byte
	for i := 1; i < len(s)-1; i++ {
		if s[i] != '\\' {
			out = append(out, s[i])
			continue
		}
		i++
		if i >= len(s)-1 {
			return "", scanError("invalid string escape")
		}
		if s[i] == '"' || s[i] == '\\' {
			out = append(out, s[i])
			continue
		}
		return "", scanError("unsupported string escape")
	}
	return string(out), nil
}

func opWidth(src []byte, i int) int {
	if i+2 < len(src) && src[i] == '.' && src[i+1] == '.' && src[i+2] == '.' {
		return 3
	}
	if i+1 < len(src) {
		c0 := src[i]
		c1 := src[i+1]
		if (c0 == ':' && c1 == '=') || (c0 == '=' && c1 == '=') || (c0 == '!' && c1 == '=') || (c0 == '<' && c1 == '=') || (c0 == '>' && c1 == '=') || (c0 == '&' && c1 == '&') || (c0 == '|' && c1 == '|') || (c0 == '<' && c1 == '<') || (c0 == '>' && c1 == '>') || (c0 == '&' && c1 == '^') || (c0 == '<' && c1 == '-') {
			return 2
		}
	}
	return 1
}

func isIdentStart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

func isIdent(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}
