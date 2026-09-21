package check

import "renvo.dev/internal/syntax"

func pointerOrderingNestedFunctionEnd(file syntax.File, start, end int) int {
	for tok := start + 1; tok < end; tok++ {
		if tokCharIs(&file, tok, ';') {
			return start
		}
		if tokCharIs(&file, tok, '(') || tokCharIs(&file, tok, '[') {
			open, close := byte('('), byte(')')
			if tokCharIs(&file, tok, '[') {
				open, close = '[', ']'
			}
			finish := findTypeMatching(file, tok, open, close)
			if finish <= tok {
				return start
			}
			tok = finish - 1
			continue
		}
		if tokCharIs(&file, tok, '{') {
			finish := findTypeMatching(file, tok, '{', '}')
			if finish <= tok {
				return start
			}
			if isCompositeTypeBodyOpen(file, tok) {
				tok = finish - 1
				continue
			}
			return finish - 1
		}
	}
	return start
}
