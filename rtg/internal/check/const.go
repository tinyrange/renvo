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
	if end-start == 2 && tokenTextIs(&file, start, "-") {
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
	if tok.Kind == syntax.TokenIdent && tokenString(&file, start) == "true" {
		return ConstValue{Kind: ConstBool, Bool: true, Ok: true}
	}
	if tok.Kind == syntax.TokenIdent && tokenString(&file, start) == "false" {
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
	start := token.Start
	end := token.End
	base := 10
	if end-start > 2 && file.Src[start] == '0' {
		prefix := file.Src[start+1]
		if prefix == 'x' || prefix == 'X' {
			base = 16
			start += 2
		} else if prefix == 'b' || prefix == 'B' {
			base = 2
			start += 2
		} else if prefix == 'o' || prefix == 'O' {
			base = 8
			start += 2
		}
	}
	if base == 10 && end-start > 1 && file.Src[start] == '0' {
		base = 8
		start++
	}
	value := 0
	for i := start; i < end; i++ {
		c := file.Src[i]
		if c == '_' {
			continue
		}
		digit := -1
		if c >= '0' && c <= '9' {
			digit = int(c - '0')
		} else if c >= 'a' && c <= 'f' {
			digit = int(c-'a') + 10
		} else if c >= 'A' && c <= 'F' {
			digit = int(c-'A') + 10
		}
		if digit < 0 || digit >= base {
			return 0, false
		}
		value = value*base + digit
	}
	return value, true
}
