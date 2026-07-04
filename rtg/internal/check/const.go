package check

import "j5.nz/rtg/rtg/internal/syntax"

func evalConstValue(file syntax.File, values []ExprSpan, valueIndex int) ConstValue {
	if valueIndex < 0 || valueIndex >= len(values) {
		return ConstValue{}
	}
	return evalConstSpan(file, values[valueIndex])
}

func evalConstSpan(file syntax.File, span ExprSpan) ConstValue {
	start := span.StartTok
	end := span.EndTok
	if start < 0 || end <= start || end > len(file.Tokens) {
		return ConstValue{}
	}
	sign := 1
	if end-start == 2 && tokenTextIs(file, start, "-") {
		sign = -1
		start++
	}
	if end-start != 1 {
		return ConstValue{}
	}
	tok := file.Tokens[start]
	if tok.Kind == syntax.TokenNumber {
		value, ok := parseConstInt(file, start)
		if !ok {
			return ConstValue{}
		}
		return ConstValue{Kind: ConstInt, Int: sign * value, Ok: true}
	}
	if sign != 1 {
		return ConstValue{}
	}
	if tok.Kind == syntax.TokenString {
		value, ok := syntax.StringLiteralValue(file.Src, tok)
		if !ok {
			return ConstValue{}
		}
		return ConstValue{Kind: ConstString, String: value, Ok: true}
	}
	if tok.Kind == syntax.TokenIdent && tokenString(file, start) == "true" {
		return ConstValue{Kind: ConstBool, Bool: true, Ok: true}
	}
	if tok.Kind == syntax.TokenIdent && tokenString(file, start) == "false" {
		return ConstValue{Kind: ConstBool, Bool: false, Ok: true}
	}
	return ConstValue{}
}

func parseConstInt(file syntax.File, tok int) (int, bool) {
	if tok < 0 || tok >= len(file.Tokens) {
		return 0, false
	}
	token := file.Tokens[tok]
	if token.Kind != syntax.TokenNumber || token.Start < 0 || token.End > len(file.Src) || token.Start >= token.End {
		return 0, false
	}
	value := 0
	for i := token.Start; i < token.End; i++ {
		c := file.Src[i]
		if c == '_' {
			continue
		}
		if c < '0' || c > '9' {
			return 0, false
		}
		value = value*10 + int(c-'0')
	}
	return value, true
}
