package c11

type ppExpressionParser struct {
	pp     *preprocessor
	tokens []ppToken
	at     int
	ok     bool
}

func (p *preprocessor) evaluate(tokens []ppToken) (bool, bool) {
	defined := make([]ppToken, 0, len(tokens))
	for i := 0; i < len(tokens); i++ {
		if !p.tokenIs(tokens[i], "defined") {
			defined = append(defined, tokens[i])
			continue
		}
		parenthesized := i+1 < len(tokens) && p.tokenIs(tokens[i+1], "(")
		if parenthesized {
			i++
		}
		if i+1 >= len(tokens) || ppTokenKind(tokens[i+1]) != ppIdent {
			return false, false
		}
		i++
		value := byte('0')
		if p.lookupToken(tokens[i]) >= 0 {
			value = '1'
		}
		defined = append(defined, p.generatedToken(ppNumber, []byte{value}, ppTokenSpace(tokens[i])))
		if parenthesized {
			if i+1 >= len(tokens) || !p.tokenIs(tokens[i+1], ")") {
				return false, false
			}
			i++
		}
	}
	expanded := p.expand(defined, 0)
	if !p.ok {
		return false, false
	}
	for i := 0; i < len(expanded); i++ {
		if ppTokenKind(expanded[i]) == ppIdent {
			expanded[i] = p.generatedToken(ppNumber, []byte("0"), ppTokenSpace(expanded[i]))
		}
	}
	parser := ppExpressionParser{pp: p, tokens: expanded, ok: true}
	value := parser.conditional(true)
	return value != 0, parser.ok && parser.at == len(expanded)
}

func (p *ppExpressionParser) conditional(evaluate bool) int64 {
	value := p.binary(1, evaluate)
	if p.take("?") {
		yes := p.conditional(evaluate && value != 0)
		if !p.take(":") {
			p.ok = false
			return 0
		}
		no := p.conditional(evaluate && value == 0)
		if !evaluate {
			return 0
		}
		if value != 0 {
			return yes
		}
		return no
	}
	return value
}

func (p *ppExpressionParser) binary(minimum int, evaluate bool) int64 {
	value := p.unary(evaluate)
	for p.at < len(p.tokens) {
		precedence := p.binaryPrecedence(p.tokens[p.at])
		if precedence < minimum {
			break
		}
		operator := p.tokens[p.at]
		p.at++
		rightEvaluate := evaluate
		if evaluate && (p.pp.tokenIs(operator, "&&") && value == 0 || p.pp.tokenIs(operator, "||") && value != 0) {
			rightEvaluate = false
		}
		right := p.binary(precedence+1, rightEvaluate)
		if evaluate {
			value = p.applyBinary(operator, value, right)
		}
	}
	return value
}

func (p *ppExpressionParser) binaryPrecedence(tok ppToken) int {
	switch string(p.pp.tokenText(tok)) {
	case "||":
		return 1
	case "&&":
		return 2
	case "|":
		return 3
	case "^":
		return 4
	case "&":
		return 5
	case "==", "!=":
		return 6
	case "<", ">", "<=", ">=":
		return 7
	case "<<", ">>":
		return 8
	case "+", "-":
		return 9
	case "*", "/", "%":
		return 10
	}
	return 0
}

func (p *ppExpressionParser) applyBinary(operator ppToken, left int64, right int64) int64 {
	op := string(p.pp.tokenText(operator))
	switch op {
	case "||":
		if left != 0 || right != 0 {
			return 1
		}
		return 0
	case "&&":
		if left != 0 && right != 0 {
			return 1
		}
		return 0
	case "|":
		return left | right
	case "^":
		return left ^ right
	case "&":
		return left & right
	case "==":
		if left == right {
			return 1
		}
		return 0
	case "!=":
		if left != right {
			return 1
		}
		return 0
	case "<":
		if left < right {
			return 1
		}
		return 0
	case ">":
		if left > right {
			return 1
		}
		return 0
	case "<=":
		if left <= right {
			return 1
		}
		return 0
	case ">=":
		if left >= right {
			return 1
		}
		return 0
	case "<<":
		return left << (uint(right) & 63)
	case ">>":
		return left >> (uint(right) & 63)
	case "+":
		return left + right
	case "-":
		return left - right
	case "*":
		return left * right
	case "/", "%":
		if right == 0 {
			p.ok = false
			return 0
		}
		if op == "/" {
			return left / right
		}
		return left % right
	}
	p.ok = false
	return 0
}

func (p *ppExpressionParser) unary(evaluate bool) int64 {
	if p.take("+") {
		return p.unary(evaluate)
	}
	if p.take("-") {
		return -p.unary(evaluate)
	}
	if p.take("!") {
		if p.unary(evaluate) == 0 && evaluate {
			return 1
		}
		return 0
	}
	if p.take("~") {
		value := p.unary(evaluate)
		if !evaluate {
			return 0
		}
		return -value - 1
	}
	if p.take("(") {
		value := p.conditional(evaluate)
		if !p.take(")") {
			p.ok = false
		}
		return value
	}
	if p.at >= len(p.tokens) {
		p.ok = false
		return 0
	}
	tok := p.tokens[p.at]
	p.at++
	if ppTokenKind(tok) == ppChar {
		value, ok := ppParseChar(p.pp.tokenText(tok))
		if !ok {
			p.ok = false
		}
		return value
	}
	if ppTokenKind(tok) != ppNumber {
		p.ok = false
		return 0
	}
	value, ok := ppParseInteger(p.pp.tokenText(tok))
	if !ok {
		p.ok = false
	}
	return value
}

func (p *ppExpressionParser) take(text string) bool {
	if p.at >= len(p.tokens) || !p.pp.tokenIs(p.tokens[p.at], text) {
		return false
	}
	p.at++
	return true
}

func ppParseInteger(text []byte) (int64, bool) {
	if len(text) == 0 {
		return 0, false
	}
	base := int64(10)
	at := 0
	if len(text) > 2 && text[0] == '0' && (text[1] == 'x' || text[1] == 'X') {
		base = 16
		at = 2
	} else if len(text) > 1 && text[0] == '0' {
		base = 8
		at = 1
	}
	value := int64(0)
	digits := 0
	for at < len(text) {
		ch := text[at]
		digit := int64(-1)
		if ch >= '0' && ch <= '9' {
			digit = int64(ch - '0')
		} else if ch >= 'a' && ch <= 'f' {
			digit = int64(ch-'a') + 10
		} else if ch >= 'A' && ch <= 'F' {
			digit = int64(ch-'A') + 10
		}
		if digit < 0 || digit >= base {
			break
		}
		value = value*base + digit
		digits++
		at++
	}
	if digits == 0 && !(base == 8 && len(text) > 0 && text[0] == '0') {
		return 0, false
	}
	for at < len(text) && (text[at] == 'u' || text[at] == 'U' || text[at] == 'l' || text[at] == 'L') {
		at++
	}
	return value, at == len(text)
}

func ppParseChar(text []byte) (int64, bool) {
	start := 1
	if len(text) > 2 && (text[0] == 'L' || text[0] == 'u' || text[0] == 'U') {
		start++
	}
	if start >= len(text)-1 {
		return 0, false
	}
	if text[start] != '\\' {
		return int64(text[start]), start+2 == len(text)
	}
	start++
	if start >= len(text)-1 {
		return 0, false
	}
	switch text[start] {
	case 'n':
		return 10, true
	case 'r':
		return 13, true
	case 't':
		return 9, true
	case 'v':
		return 11, true
	case 'f':
		return 12, true
	case 'a':
		return 7, true
	case 'b':
		return 8, true
	case 'e':
		return 27, true
	case '\\', '\'', '"', '?':
		return int64(text[start]), true
	}
	return 0, false
}
