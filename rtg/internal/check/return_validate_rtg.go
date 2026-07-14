//go:build rtg

package check

import "j5.nz/rtg/rtg/internal/syntax"

func invalidReturnCount(file syntax.File, fn syntax.FuncDecl, signature FuncSignature) int {
	limit := fn.BodyEnd - 1
	for i := fn.BodyStart + 1; i < limit; i++ {
		if file.Tokens[i].Kind == syntax.TokenFunc {
			i = rtgSkipNestedFunction(file, i, limit)
			continue
		}
		if file.Tokens[i].Kind != syntax.TokenReturn {
			continue
		}
		count := 0
		call := false
		paren, bracket, brace := 0, 0, 0
		for j := i + 1; j < limit; j++ {
			if j == i+1 && file.Tokens[j].Line > file.Tokens[i].Line {
				break
			}
			if j > i+1 && paren == 0 && bracket == 0 && brace == 0 && file.Tokens[j].Line > file.Tokens[j-1].Line {
				break
			}
			if paren == 0 && bracket == 0 && brace == 0 {
				if tokCharIs(file, j, ';') || tokCharIs(file, j, '}') {
					break
				}
				if count == 0 {
					count = 1
				}
				if tokCharIs(file, j, ',') {
					count++
				}
			}
			if tokCharIs(file, j, '(') {
				paren++
				call = true
			} else if tokCharIs(file, j, ')') && paren > 0 {
				paren--
			} else if tokCharIs(file, j, '[') {
				bracket++
			} else if tokCharIs(file, j, ']') && bracket > 0 {
				bracket--
			} else if tokCharIs(file, j, '{') {
				brace++
			} else if tokCharIs(file, j, '}') && brace > 0 {
				brace--
			}
		}
		want := len(signature.Results)
		if count == want || (count == 0 && rtgResultsNamed(signature.Results)) || (count == 1 && want > 1 && call) {
			continue
		}
		return i
	}
	return -1
}

func rtgSkipNestedFunction(file syntax.File, start int, limit int) int {
	open := start
	for open < limit && !tokCharIs(file, open, '{') {
		open++
	}
	depth := 0
	for i := open; i < limit; i++ {
		if tokCharIs(file, i, '{') {
			depth++
		} else if tokCharIs(file, i, '}') {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return start
}

func rtgResultsNamed(results []Field) bool {
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

func invalidDefiniteAssignmentType(file syntax.File, fn syntax.FuncDecl) int {
	for i := fn.BodyStart + 2; i+1 < fn.BodyEnd; i++ {
		if !tokenTextIs(file, i, "=") || file.Tokens[i-1].Kind != syntax.TokenIdent || file.Tokens[i+1].Kind != syntax.TokenString {
			continue
		}
		for j := i - 2; j >= fn.BodyStart+1; j-- {
			if file.Tokens[j].Kind == syntax.TokenVar && j+2 < i && tokenTextIs(file, j+1, tokenString(file, i-1)) && tokenTextIs(file, j+2, "int") {
				return i + 1
			}
		}
	}
	return -1
}

func excludedFeatureToken(file syntax.File, fn syntax.FuncDecl) int {
	for i := fn.StartTok; i < fn.EndTok && i < len(file.Tokens); i++ {
		kind := file.Tokens[i].Kind
		if kind == syntax.TokenGo || kind == syntax.TokenChan || kind == syntax.TokenSelect {
			return i
		}
	}
	return -1
}
