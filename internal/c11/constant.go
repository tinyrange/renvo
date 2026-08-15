package c11

type constantParser struct {
	t      *translator
	tokens []token
	pos    int
}

func (t *translator) constantExpression(tokens []token) (int, bool) {
	p := constantParser{t: t, tokens: tokens}
	value, ok := p.binary(1)
	return value, ok && p.pos == len(tokens)
}

func (p *constantParser) current(text string) bool {
	return p.pos < len(p.tokens) && tokenIs(p.t.src, p.tokens[p.pos], text)
}

func (p *constantParser) binary(minimum int) (int, bool) {
	left, ok := p.unary()
	if !ok {
		return 0, false
	}
	for p.pos < len(p.tokens) {
		op := string(tokenText(p.t.src, p.tokens[p.pos]))
		precedence := constantPrecedence(op)
		if precedence < minimum {
			break
		}
		p.pos++
		right, valid := p.binary(precedence + 1)
		if !valid {
			return 0, false
		}
		left, ok = applyConstantOperator(op, left, right)
		if !ok {
			return 0, false
		}
	}
	return left, true
}

func (p *constantParser) unary() (int, bool) {
	if p.pos >= len(p.tokens) {
		return 0, false
	}
	if p.current("+") || p.current("-") || p.current("~") || p.current("!") {
		op := string(tokenText(p.t.src, p.tokens[p.pos]))
		p.pos++
		value, ok := p.unary()
		if !ok {
			return 0, false
		}
		switch op {
		case "+":
			return value, true
		case "-":
			return -value, true
		case "~":
			return -value - 1, true
		default:
			return boolConstant(value == 0), true
		}
	}
	if p.current("sizeof") || p.current("_Alignof") {
		align := p.current("_Alignof")
		p.pos++
		if !p.current("(") {
			return 0, false
		}
		end := matchingToken(p.t.src, p.tokens, p.pos, "(", ")")
		if end < 0 {
			return 0, false
		}
		typeID, ok := p.t.typeFromTokens(p.tokens[p.pos+1 : end])
		if !ok {
			return 0, false
		}
		p.pos = end + 1
		if align {
			return p.t.typeAlign(typeID), true
		}
		return p.t.typeSize(typeID), true
	}
	if p.current("(") {
		p.pos++
		value, ok := p.binary(1)
		if !ok || !p.current(")") {
			return 0, false
		}
		p.pos++
		return value, true
	}
	tok := p.tokens[p.pos]
	p.pos++
	if tokenKind(tok) == tokenNumber {
		return integerTokenValue(p.t.src, []token{tok})
	}
	if tokenKind(tok) == tokenIdent {
		return p.t.lookupValue(tokenText(p.t.src, tok))
	}
	return 0, false
}

func constantPrecedence(op string) int {
	switch op {
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
	case "<", "<=", ">", ">=":
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

func applyConstantOperator(op string, left int, right int) (int, bool) {
	switch op {
	case "+":
		return left + right, true
	case "-":
		return left - right, true
	case "*":
		return left * right, true
	case "/":
		if right == 0 {
			return 0, false
		}
		return left / right, true
	case "%":
		if right == 0 {
			return 0, false
		}
		return left % right, true
	case "<<":
		if right < 0 {
			return 0, false
		}
		return left << uint(right), true
	case ">>":
		if right < 0 {
			return 0, false
		}
		return left >> uint(right), true
	case "&":
		return left & right, true
	case "^":
		return left ^ right, true
	case "|":
		return left | right, true
	case "==":
		return boolConstant(left == right), true
	case "!=":
		return boolConstant(left != right), true
	case "<":
		return boolConstant(left < right), true
	case "<=":
		return boolConstant(left <= right), true
	case ">":
		return boolConstant(left > right), true
	case ">=":
		return boolConstant(left >= right), true
	case "&&":
		return boolConstant(left != 0 && right != 0), true
	case "||":
		return boolConstant(left != 0 || right != 0), true
	}
	return 0, false
}

func boolConstant(value bool) int {
	if value {
		return 1
	}
	return 0
}
