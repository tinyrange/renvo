package ide

type syntaxKind int

const (
	syntaxText syntaxKind = iota
	syntaxKeyword
	syntaxBuiltin
	syntaxString
	syntaxComment
	syntaxNumber
)

type goSyntaxState int

const (
	goSyntaxNormal goSyntaxState = iota
	goSyntaxBlockComment
	goSyntaxRawString
)

type syntaxSpan struct {
	start int
	end   int
	kind  syntaxKind
}

func highlightGoLine(line string, state goSyntaxState) ([]syntaxSpan, goSyntaxState) {
	return highlightGoLineInto(nil, line, state)
}

func highlightGoLineInto(spans []syntaxSpan, line string, state goSyntaxState) ([]syntaxSpan, goSyntaxState) {
	spans = spans[:0]
	at := 0
	for at < len(line) {
		start := at
		if state == goSyntaxBlockComment {
			for at+1 < len(line) && !(line[at] == '*' && line[at+1] == '/') {
				at++
			}
			if at+1 < len(line) {
				at += 2
				state = goSyntaxNormal
			} else {
				at = len(line)
			}
			spans = appendSyntaxSpan(spans, start, at, syntaxComment)
			continue
		}
		if state == goSyntaxRawString {
			for at < len(line) && line[at] != '`' {
				at++
			}
			if at < len(line) {
				at++
				state = goSyntaxNormal
			}
			spans = appendSyntaxSpan(spans, start, at, syntaxString)
			continue
		}

		c := line[at]
		if c == '/' && at+1 < len(line) && line[at+1] == '/' {
			spans = appendSyntaxSpan(spans, at, len(line), syntaxComment)
			at = len(line)
			continue
		}
		if c == '/' && at+1 < len(line) && line[at+1] == '*' {
			at += 2
			state = goSyntaxBlockComment
			for at+1 < len(line) && !(line[at] == '*' && line[at+1] == '/') {
				at++
			}
			if at+1 < len(line) {
				at += 2
				state = goSyntaxNormal
			} else {
				at = len(line)
			}
			spans = appendSyntaxSpan(spans, start, at, syntaxComment)
			continue
		}
		if c == '`' {
			at++
			state = goSyntaxRawString
			for at < len(line) && line[at] != '`' {
				at++
			}
			if at < len(line) {
				at++
				state = goSyntaxNormal
			}
			spans = appendSyntaxSpan(spans, start, at, syntaxString)
			continue
		}
		if c == '"' || c == '\'' {
			quote := c
			at++
			for at < len(line) {
				if line[at] == '\\' && at+1 < len(line) {
					at += 2
					continue
				}
				closed := line[at] == quote
				at++
				if closed {
					break
				}
			}
			spans = appendSyntaxSpan(spans, start, at, syntaxString)
			continue
		}
		if c >= '0' && c <= '9' {
			at++
			for at < len(line) {
				numberByte := line[at]
				if isGoNumberByte(numberByte) {
					at++
					continue
				}
				if (numberByte == '+' || numberByte == '-') && at > start {
					previous := line[at-1]
					if previous == 'e' || previous == 'E' || previous == 'p' || previous == 'P' {
						at++
						continue
					}
				}
				break
			}
			spans = appendSyntaxSpan(spans, start, at, syntaxNumber)
			continue
		}
		if isGoIdentStart(c) {
			at++
			for at < len(line) && isGoIdentContinue(line[at]) {
				at++
			}
			word := line[start:at]
			kind := syntaxText
			if isGoKeyword(word) {
				kind = syntaxKeyword
			} else if isGoBuiltin(word) {
				kind = syntaxBuiltin
			}
			spans = appendSyntaxSpan(spans, start, at, kind)
			continue
		}
		at++
		for at < len(line) && !isGoSyntaxStart(line, at) {
			at++
		}
		spans = appendSyntaxSpan(spans, start, at, syntaxText)
	}
	return spans, state
}

func appendSyntaxSpan(spans []syntaxSpan, start, end int, kind syntaxKind) []syntaxSpan {
	if end <= start {
		return spans
	}
	if len(spans) > 0 {
		last := &spans[len(spans)-1]
		if last.end == start && last.kind == kind {
			last.end = end
			return spans
		}
	}
	return append(spans, syntaxSpan{start: start, end: end, kind: kind})
}

func isGoSyntaxStart(line string, at int) bool {
	c := line[at]
	return c == '/' || c == '`' || c == '"' || c == '\'' || c >= '0' && c <= '9' || isGoIdentStart(c)
}

func isGoIdentStart(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= 0x80
}

func isGoIdentContinue(c byte) bool {
	return isGoIdentStart(c) || c >= '0' && c <= '9'
}

func isGoNumberByte(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_' || c == '.'
}

func isGoKeyword(word string) bool {
	switch word {
	case "break", "case", "chan", "const", "continue", "default", "defer", "else", "fallthrough", "for", "func", "go", "goto", "if", "import", "interface", "map", "package", "range", "return", "select", "struct", "switch", "type", "var":
		return true
	}
	return false
}

func isGoBuiltin(word string) bool {
	switch word {
	case "any", "bool", "byte", "comparable", "complex64", "complex128", "error", "float32", "float64", "int", "int8", "int16", "int32", "int64", "rune", "string", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr":
		return true
	case "append", "cap", "clear", "close", "complex", "copy", "delete", "imag", "len", "make", "max", "min", "new", "panic", "print", "println", "real", "recover":
		return true
	case "false", "iota", "nil", "true":
		return true
	}
	return false
}
