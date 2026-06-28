package scan

import "strconv"

const (
	Ident  = "ident"
	Number = "number"
	String = "string"
	Char   = "char"
	Op     = "op"
	EOF    = "eof"
)

type Token struct {
	Kind   string
	Text   string
	Start  int
	End    int
	Line   int
	Column int
}

type scanError string

func (e scanError) Error() string {
	return string(e)
}

func posError(line int, column int, message string) error {
	lineText := strconv.Itoa(line)
	columnText := strconv.Itoa(column)
	text := lineText + ":" + columnText + ": " + message
	return scanError(text)
}

func Tokens(src []byte) ([]Token, error) {
	var toks []Token
	line := 1
	col := 1
	i := 0
	for i < len(src) {
		c := src[i]
		if c == ' ' || c == '\t' || c == '\r' {
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
		if c == '/' && i+1 < len(src) && src[i+1] == '/' {
			i += 2
			col += 2
			for i < len(src) && src[i] != '\n' {
				i++
				col++
			}
			continue
		}
		if c == '/' && i+1 < len(src) && src[i+1] == '*' {
			startLine := line
			startCol := col
			i += 2
			col += 2
			closed := false
			for i+1 < len(src) {
				if src[i] == '*' && src[i+1] == '/' {
					i += 2
					col += 2
					closed = true
					break
				}
				if src[i] == '\n' {
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
		if isIdentStart(c) {
			start := i
			startLine := line
			startCol := col
			i++
			col++
			for i < len(src) && isIdent(src[i]) {
				i++
				col++
			}
			toks = append(toks, Token{Kind: Ident, Text: string(src[start:i]), Start: start, End: i, Line: startLine, Column: startCol})
			continue
		}
		if c >= '0' && c <= '9' {
			start := i
			startLine := line
			startCol := col
			i++
			col++
			for i < len(src) && (isIdent(src[i]) || src[i] == '.') {
				i++
				col++
			}
			toks = append(toks, Token{Kind: Number, Text: string(src[start:i]), Start: start, End: i, Line: startLine, Column: startCol})
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
				if quote != '`' && src[i] == '\\' {
					i += 2
					col += 2
					continue
				}
				if src[i] == quote {
					i++
					col++
					kind := String
					if quote == '\'' {
						kind = Char
					}
					toks = append(toks, Token{Kind: kind, Text: string(src[start:i]), Start: start, End: i, Line: startLine, Column: startCol})
					break
				}
				if src[i] == '\n' {
					i++
					line++
					col = 1
				} else {
					i++
					col++
				}
			}
			if len(toks) == 0 || toks[len(toks)-1].Start != start {
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
		toks = append(toks, Token{Kind: Op, Text: string(src[start:i]), Start: start, End: i, Line: startLine, Column: startCol})
	}
	toks = append(toks, Token{Kind: EOF, Text: "", Start: len(src), End: len(src), Line: line, Column: col})
	return toks, nil
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
