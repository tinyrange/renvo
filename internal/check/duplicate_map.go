package check

import "renvo.dev/internal/syntax"

// Inspect explicit map literals, including package initializers and nested
// literals. Nonconstant keys are deliberately left alone: their values may
// coincide at runtime without making the literal invalid.
func invalidDuplicateMapKey(file syntax.File) int {
	hasMap := false
	for _, tok := range file.Tokens {
		if tok.KindLine&255 == syntax.TokenMap {
			hasMap = true
			break
		}
	}
	if !hasMap {
		return -1
	}
	for _, literal := range appendExprComposites(nil, file, 0, len(file.Tokens)) {
		if file.Tokens[literal.TypeStart].KindLine&255 != syntax.TokenMap {
			continue
		}
		keys := make([]ConstValue, len(literal.Elems)*2+1)
		for _, elem := range literal.Elems {
			colon := findTypeTopLevelChar(file, elem.StartTok, elem.EndTok, ':')
			if colon < 0 {
				continue
			}
			start, end := stripOuterParens(file, elem.StartTok, colon)
			value := duplicateCaseLiteral(file, ExprSpan{StartTok: start, EndTok: end})
			if !value.Ok {
				continue
			}
			hash := value.Int
			if value.Kind == ConstString {
				hash = hashCheckString(value.String)
			}
			bucket := int(uint(hash) % uint(len(keys)))
			for keys[bucket].Ok {
				previous := keys[bucket]
				if value.Kind == previous.Kind && ((value.Kind == ConstInt && value.Int == previous.Int) || (value.Kind == ConstString && value.String == previous.String)) {
					return elem.StartTok
				}
				bucket = (bucket + 1) % len(keys)
			}
			keys[bucket] = value
		}
	}
	return -1
}
