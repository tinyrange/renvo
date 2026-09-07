package check

import "renvo.dev/internal/syntax"

func invalidArrayLengthTypeSpan(context constantIndexContext, start int, end int) int {
	file := context.pkg.Files[context.fileIndex].File
	for tok := start; tok >= 0 && tok+1 < end; tok++ {
		if !tokCharIs(&file, tok, '[') || tokCharIs(&file, tok+1, ']') || tokenTextIs(&file, tok+1, "...") {
			continue
		}
		if tok > start && file.Tokens[tok-1].KindLine&255 == syntax.TokenMap {
			continue
		}
		close := findTypeMatching(file, tok, '[', ']')
		if close <= tok || close > end {
			continue
		}
		if unsafeAddFractionalDecimal(file, tok+1, close-1) {
			return tok + 1
		}
		value := wideConstantExpr(context, tok+1, close-1, 0)
		if value.ok {
			// Every supported target has at most 64-bit ints. Target-specific
			// smaller bounds remain the target layout checker's responsibility.
			if value.negative || len(value.words) > 5 || len(value.words) == 5 && value.words[4] >= 8 {
				return tok + 1
			}
		}
		tok = close - 1
	}
	return -1
}
