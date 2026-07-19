package check

import "renvo.dev/internal/syntax"

// invalidReturnCount rejects return lists whose arity is statically certain
// to disagree with the function signature. A single call expression is left
// to later tuple-aware checking because it may return multiple values.
func invalidReturnCount(file syntax.File, fn syntax.FuncDecl, signature FuncSignature) int {
	start := fn.BodyStart + 1
	end := fn.BodyEnd - 1
	if start < 0 {
		start = 0
	}
	if end > len(file.Tokens) {
		end = len(file.Tokens)
	}
	for i := start; i < end; i++ {
		if file.Tokens[i].Kind == syntax.TokenFunc {
			i = skipNestedFunction(file, i, end)
			continue
		}
		if file.Tokens[i].Kind != syntax.TokenReturn {
			continue
		}
		valueStart, valueEnd, count := returnValueList(file, i, end)
		expected := len(signature.Results)
		if count == expected {
			continue
		}
		if count == 0 && resultsAreNamed(signature.Results) {
			continue
		}
		if count == 1 && expected > 1 && returnMayBeMultiValueCall(file, valueStart, valueEnd) {
			continue
		}
		return i
	}
	return -1
}

func skipNestedFunction(file syntax.File, start int, limit int) int {
	open := -1
	for i := start + 1; i < limit; i++ {
		if tokCharIs(&file, i, '{') {
			open = i
			break
		}
		if tokCharIs(&file, i, ';') {
			return start
		}
	}
	if open < 0 {
		return start
	}
	depth := 1
	for i := open + 1; i < limit; i++ {
		if tokCharIs(&file, i, '{') {
			depth++
		} else if tokCharIs(&file, i, '}') {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return start
}

func returnValueList(file syntax.File, returnTok int, limit int) (int, int, int) {
	start := returnTok + 1
	if start >= limit || tokCharIs(&file, start, ';') || tokCharIs(&file, start, '}') || file.Tokens[start].Line > file.Tokens[returnTok].Line {
		return start, start, 0
	}
	parenDepth := 0
	bracketDepth := 0
	braceDepth := 0
	count := 1
	end := start
	for i := start; i < limit; i++ {
		if i > start && parenDepth == 0 && bracketDepth == 0 && braceDepth == 0 && file.Tokens[i].Line > file.Tokens[i-1].Line && !returnLineContinues(file, i-1) {
			break
		}
		if parenDepth == 0 && bracketDepth == 0 && braceDepth == 0 {
			if tokCharIs(&file, i, ';') || tokCharIs(&file, i, '}') {
				break
			}
			if tokCharIs(&file, i, ',') {
				count++
			}
		}
		if tokCharIs(&file, i, '(') {
			parenDepth++
		} else if tokCharIs(&file, i, ')') && parenDepth > 0 {
			parenDepth--
		} else if tokCharIs(&file, i, '[') {
			bracketDepth++
		} else if tokCharIs(&file, i, ']') && bracketDepth > 0 {
			bracketDepth--
		} else if tokCharIs(&file, i, '{') {
			braceDepth++
		} else if tokCharIs(&file, i, '}') && braceDepth > 0 {
			braceDepth--
		}
		end = i + 1
	}
	return start, end, count
}

func returnLineContinues(file syntax.File, tok int) bool {
	if tok < 0 || tok >= len(file.Tokens) {
		return false
	}
	text := syntax.TokenText(file.Src, file.Tokens[tok])
	if len(text) == 0 {
		return false
	}
	last := text[len(text)-1]
	return last == ',' || last == '.' || last == '+' || last == '-' || last == '*' || last == '/' || last == '%' || last == '&' || last == '|' || last == '^' || last == '=' || last == '<' || last == '>' || last == '!' || last == ':'
}

func returnMayBeMultiValueCall(file syntax.File, start int, end int) bool {
	for i := start; i < end; i++ {
		if tokCharIs(&file, i, '(') {
			return true
		}
	}
	return false
}

func resultsAreNamed(results []Field) bool {
	if len(results) == 0 {
		return false
	}
	for i := 0; i < len(results); i++ {
		if results[i].Name == "" {
			return false
		}
	}
	return true
}
