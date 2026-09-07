package check

import "renvo.dev/internal/syntax"

// These expression forms have a definite type without resolving any names.
// Leave conversions, calls, selectors, and indexes to declaration-aware type
// checking: a suffix can change a literal's type (e.g. a map lookup of a slice).
func invalidCapacityLiteral(file syntax.File, span ExprSpan) bool {
	start, end := stripOuterParens(file, span.StartTok, span.EndTok)
	if start < 0 || start >= end {
		return false
	}
	kind := file.Tokens[start].KindLine & 255
	if end-start == 1 {
		return kind == syntax.TokenString || kind == syntax.TokenNumber || kind == syntax.TokenChar
	}
	if kind != syntax.TokenMap && kind != syntax.TokenStruct {
		return false
	}
	open := findTypeTopLevelChar(file, start, end, '{')
	if open < 0 {
		return false
	}
	close := findTypeMatching(file, open, '{', '}')
	if kind == syntax.TokenStruct && close < end && tokCharIs(&file, close, '{') {
		close = findTypeMatching(file, close, '{', '}')
	}
	return close == end
}
