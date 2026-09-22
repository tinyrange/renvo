package check

import "renvo.dev/internal/syntax"

func unsafeAddFractionalDecimal(file syntax.File, start int, end int) bool {
	if start < end && (tokenTextIs(&file, start, "+") || tokenTextIs(&file, start, "-")) {
		start++
	}
	if end-start != 1 || file.Tokens[start].KindLine&255 != syntax.TokenNumber {
		return false
	}
	text := tokenString(&file, start)
	if len(text) > 1 && text[0] == '0' && (text[1] == 'x' || text[1] == 'X') {
		return false
	}
	digits, point, lastNonzero := 0, -1, 0
	i := 0
	for ; i < len(text) && text[i] != 'e' && text[i] != 'E'; i++ {
		c := text[i]
		if c == '_' {
			continue
		}
		if c == '.' {
			point = digits
			continue
		}
		if c < '0' || c > '9' {
			return false
		}
		digits++
		if c != '0' {
			lastNonzero = digits
		}
	}
	if point < 0 {
		point = digits
	}
	exponent, sign := 0, 1
	if i < len(text) {
		i++
		if i < len(text) && (text[i] == '+' || text[i] == '-') {
			if text[i] == '-' {
				sign = -1
			}
			i++
		}
		for ; i < len(text); i++ {
			if text[i] == '_' {
				continue
			}
			if text[i] < '0' || text[i] > '9' {
				return false
			}
			if exponent < len(text) {
				exponent = exponent*10 + int(text[i]-'0')
			}
		}
	}
	return lastNonzero > 0 && lastNonzero > point+sign*exponent
}
