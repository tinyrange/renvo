package check

import "renvo.dev/internal/syntax"

// invalidReturnCount rejects return lists whose arity is statically certain
// to disagree with the function signature. A single call expression is left
// to later tuple-aware checking because it may return multiple values.
func invalidReturnCount(file *syntax.File, fn *syntax.FuncDecl, signature *FuncSignature) (int, int) {
	start := fn.BodyStart + 1
	end := fn.BodyEnd - 1
	if start < 0 {
		start = 0
	}
	if end > len(file.Tokens) {
		end = len(file.Tokens)
	}
	for i := start; i < end; i++ {
		if file.Tokens[i].KindLine&255 == syntax.TokenFunc {
			i = skipNestedFunction(file, i, end)
			continue
		}
		if file.Tokens[i].KindLine&255 != syntax.TokenReturn {
			continue
		}
		valueStart, valueEnd, count := returnValueList(file, i, end)
		expected := len(signature.Results)
		if count == expected {
			if count == 1 && valueEnd-valueStart == 1 && signature.Results[0].TypeEnd-signature.Results[0].TypeStart == 1 {
				want := tokenString(file, signature.Results[0].TypeStart)
				if primitiveTypeMismatch(want, definiteLiteralKind(file, valueStart)) {
					return CheckErrReturnType, valueStart
				}
			}
			continue
		}
		if count == 0 && resultsAreNamed(signature.Results) {
			continue
		}
		if count == 1 && expected > 1 && returnMayBeMultiValueCall(file, valueStart, valueEnd) {
			continue
		}
		return CheckErrReturnCount, i
	}
	return CheckOK, -1
}

func skipNestedFunction(file *syntax.File, start int, limit int) int {
	// Struct/interface types in parameters or results own braces too. Parse the
	// complete signature before skipping the literal's body; a function type
	// without a body must leave later returns in the enclosing function visible.
	fn := genericFunctionLiteral(file, start, limit)
	if fn.BodyStart >= 0 && fn.BodyEnd > fn.BodyStart && fn.BodyEnd <= limit {
		return fn.BodyEnd - 1
	}
	return start
}

func returnValueList(file *syntax.File, returnTok int, limit int) (int, int, int) {
	start := returnTok + 1
	if start >= limit || tokCharIs(file, start, ';') || tokCharIs(file, start, '}') || syntax.TokenLine(file.Tokens[start]) > syntax.TokenLine(file.Tokens[returnTok]) {
		return start, start, 0
	}
	parenDepth := 0
	bracketDepth := 0
	braceDepth := 0
	count := 1
	end := start
	for i := start; i < limit; i++ {
		if i > start && parenDepth == 0 && bracketDepth == 0 && braceDepth == 0 && syntax.TokenLine(file.Tokens[i]) > syntax.TokenLine(file.Tokens[i-1]) && !returnLineContinues(file, i-1) {
			break
		}
		ch := file.Tokens[i].KindLine >> syntax.TokenOperatorCharShift & syntax.TokenOperatorCharMask
		if parenDepth == 0 && bracketDepth == 0 && braceDepth == 0 {
			if ch == int(';') || ch == int('}') {
				break
			}
			if ch == int(',') {
				count++
			}
		}
		if ch == int('(') {
			parenDepth++
		} else if ch == int(')') && parenDepth > 0 {
			parenDepth--
		} else if ch == int('[') {
			bracketDepth++
		} else if ch == int(']') && bracketDepth > 0 {
			bracketDepth--
		} else if ch == int('{') {
			braceDepth++
		} else if ch == int('}') && braceDepth > 0 {
			braceDepth--
		}
		end = i + 1
	}
	return start, end, count
}

func returnLineContinues(file *syntax.File, tok int) bool {
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

func returnMayBeMultiValueCall(file *syntax.File, start int, end int) bool {
	for i := start; i < end; i++ {
		if tokCharIs(file, i, '(') {
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
