package c11

type constantParser struct {
	t           *translator
	tokens      []token
	pos         int
	ignoreValue bool
}

func (t *translator) constantExpression(tokens []token) (int, bool) {
	p := constantParser{t: t, tokens: tokens}
	value, ok := p.conditional()
	return value, ok && p.pos == len(tokens)
}

func (p *constantParser) conditional() (int, bool) {
	conditionStart := p.pos
	condition, ok := p.binary(1)
	if !ok || !p.current("?") {
		return condition, ok
	}
	conditionEnd := p.pos
	p.pos++
	leftStart := p.pos
	left, leftOK := condition, true
	if !p.current(":") {
		outerIgnore := p.ignoreValue
		p.ignoreValue = outerIgnore || condition == 0
		left, leftOK = p.conditional()
		p.ignoreValue = outerIgnore
	}
	if !leftOK || !p.current(":") {
		return 0, false
	}
	leftEnd := p.pos
	p.pos++
	rightStart := p.pos
	outerIgnore := p.ignoreValue
	p.ignoreValue = outerIgnore || condition != 0
	right, rightOK := p.conditional()
	p.ignoreValue = outerIgnore
	if !rightOK {
		return 0, false
	}
	if outerIgnore {
		return 0, true
	}
	leftTypeTokens := p.tokens[leftStart:leftEnd]
	if leftStart == leftEnd {
		leftTypeTokens = p.tokens[conditionStart:conditionEnd]
	}
	typeID := p.t.usualArithmeticType(p.t.expressionType(leftTypeTokens), p.t.expressionType(p.tokens[rightStart:p.pos]))
	value := right
	if condition != 0 {
		value = left
	}
	value, ok = p.t.convertIntegerConstant(typeID, value)
	return value, ok
}

func (p *constantParser) current(text string) bool {
	return p.pos < len(p.tokens) && tokenIs(p.t.src, p.tokens[p.pos], text)
}

func (p *constantParser) binary(minimum int) (int, bool) {
	leftStart := p.pos
	left, ok := p.unary()
	if !ok {
		return 0, false
	}
	for p.pos < len(p.tokens) {
		op := tokenText(p.t.src, p.tokens[p.pos])
		precedence := cBinaryPrecedence(op)
		if precedence < minimum {
			break
		}
		opAt := p.pos
		p.pos++
		rightStart := p.pos
		right, valid := p.binary(precedence + 1)
		if !valid {
			return 0, false
		}
		if p.ignoreValue {
			left = 0
			continue
		}
		leftType := p.t.expressionType(p.tokens[leftStart:opAt])
		rightType := p.t.expressionType(p.tokens[rightStart:p.pos])
		typeID := p.t.usualArithmeticType(leftType, rightType)
		shift := len(op) == 2 && (op[0] == '<' && op[1] == '<' || op[0] == '>' && op[1] == '>')
		if shift {
			typeID = p.t.integerPromotion(leftType)
		}
		left, valid = p.t.convertIntegerConstant(typeID, left)
		if !valid {
			return 0, false
		}
		if shift {
			right, valid = p.t.convertIntegerConstant(p.t.integerPromotion(rightType), right)
		} else {
			right, valid = p.t.convertIntegerConstant(typeID, right)
		}
		if !valid {
			return 0, false
		}
		left, ok = applyConstantOperator(op, left, right, p.t.typeInfo(typeID).kind == cTypeUint)
		if !ok {
			return 0, false
		}
		if precedence >= 3 && precedence != 6 && precedence != 7 {
			left, ok = p.t.convertIntegerConstant(typeID, left)
		}
	}
	return left, true
}

func (p *constantParser) unary() (int, bool) {
	if p.pos >= len(p.tokens) {
		return 0, false
	}
	if p.ignoreValue && tokenKind(p.tokens[p.pos]) == tokenIdent {
		p.pos++
		if p.current("(") {
			end := matchingToken(p.t.src, p.tokens, p.pos, "(", ")")
			if end < 0 {
				return 0, false
			}
			p.pos = end + 1
		}
		return 0, true
	}
	if p.current("__builtin_types_compatible_p") {
		return p.typesCompatible()
	}
	if p.current("__builtin_constant_p") {
		return p.constantP()
	}
	if p.current("__builtin_offsetof") || p.current("offsetof") {
		return p.offsetof()
	}
	if value, ok, matched := p.integerBuiltin(); matched {
		return value, ok
	}
	if p.current("+") || p.current("-") || p.current("~") || p.current("!") {
		op := string(tokenText(p.t.src, p.tokens[p.pos]))
		p.pos++
		valueStart := p.pos
		value, ok := p.unary()
		if !ok {
			return 0, false
		}
		typeID := p.t.integerPromotion(p.t.expressionType(p.tokens[valueStart:p.pos]))
		value, ok = p.t.convertIntegerConstant(typeID, value)
		if !ok {
			return 0, false
		}
		switch op {
		case "+":
			return value, true
		case "-":
			return p.t.convertIntegerConstant(typeID, -value)
		case "~":
			return p.t.convertIntegerConstant(typeID, -value-1)
		default:
			return boolConstant(value == 0), true
		}
	}
	if p.current("sizeof") || p.current("_Alignof") || p.current("__alignof") || p.current("__alignof__") {
		align := !p.current("sizeof")
		p.pos++
		if !p.current("(") {
			return 0, false
		}
		end := matchingToken(p.t.src, p.tokens, p.pos, "(", ")")
		if end < 0 {
			return 0, false
		}
		operand := p.tokens[p.pos+1 : end]
		typeID, ok := p.t.typeFromTokens(operand)
		if !ok {
			if align {
				return 0, false
			}
			// sizeof observes the operand's type before array-to-pointer
			// conversion, just like typeof.  This is especially visible for
			// string literals: sizeof("") is one, not the pointer width.
			typeID = p.t.typeofExpressionType(operand)
			if typeID == cTypeVoidID {
				trimmed := p.t.trimExpressionParens(operand)
				if len(trimmed) < 2 || !tokenIs(p.t.src, trimmed[0], "*") {
					return 0, false
				}
				pointer := p.t.typeInfo(p.t.expressionType(trimmed[1:]))
				if pointer.kind != cTypePointer || pointer.base != cTypeVoidID {
					return 0, false
				}
				p.pos = end + 1
				// GNU C defines sizeof(void) and sizeof(function) as one. The
				// kernel uses the dereferenced-void-pointer form for type choice.
				return 1, true
			}
		}
		p.pos = end + 1
		if align {
			return p.t.typeAlign(typeID), true
		}
		return p.t.typeSize(typeID), true
	}
	if p.current("(") {
		end := matchingToken(p.t.src, p.tokens, p.pos, "(", ")")
		if end > p.pos+1 && end+1 < len(p.tokens) {
			if typeID, valid := p.t.typeFromTokens(p.tokens[p.pos+1 : end]); valid {
				p.pos = end + 1
				value, valueOK := p.unary()
				if !valueOK {
					return 0, false
				}
				return p.t.convertIntegerConstant(typeID, value)
			}
		}
		p.pos++
		value, ok := p.conditional()
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
	if tokenKind(tok) == tokenChar {
		value, valid := decodeCCharacter(tokenText(p.t.src, tok))
		return value, valid
	}
	if tokenKind(tok) == tokenIdent {
		return p.t.lookupValue(tokenText(p.t.src, tok))
	}
	return 0, false
}

func (p *constantParser) integerBuiltin() (int, bool, bool) {
	if p.pos >= len(p.tokens) || tokenKind(p.tokens[p.pos]) != tokenIdent {
		return 0, false, false
	}
	name := string(tokenText(p.t.src, p.tokens[p.pos]))
	width, operation := 0, ""
	switch name {
	case "__builtin_clz":
		width, operation = 32, "clz"
	case "__builtin_clzl", "__builtin_clzll":
		width, operation = 64, "clz"
	case "__builtin_ctz":
		width, operation = 32, "ctz"
	case "__builtin_ctzl", "__builtin_ctzll":
		width, operation = 64, "ctz"
	case "__builtin_ffs":
		width, operation = 32, "ffs"
	case "__builtin_ffsl", "__builtin_ffsll":
		width, operation = 64, "ffs"
	case "__builtin_popcount":
		width, operation = 32, "popcount"
	case "__builtin_popcountl", "__builtin_popcountll":
		width, operation = 64, "popcount"
	case "__builtin_parity":
		width, operation = 32, "parity"
	case "__builtin_parityl", "__builtin_parityll":
		width, operation = 64, "parity"
	default:
		return 0, false, false
	}
	p.pos++
	if !p.current("(") {
		return 0, false, true
	}
	end := matchingToken(p.t.src, p.tokens, p.pos, "(", ")")
	if end < 0 {
		return 0, false, true
	}
	value, ok := p.t.constantExpression(p.tokens[p.pos+1 : end])
	if !ok {
		return 0, false, true
	}
	p.pos = end + 1
	bits := uint64(value)
	if width == 32 {
		bits = uint64(uint32(value))
	}
	switch operation {
	case "clz":
		count := 0
		for bit := width - 1; bit >= 0 && bits&(uint64(1)<<bit) == 0; bit-- {
			count++
		}
		return count, true, true
	case "ctz", "ffs":
		count := 0
		for count < width && bits&(uint64(1)<<count) == 0 {
			count++
		}
		if operation == "ffs" {
			if bits == 0 {
				return 0, true, true
			}
			count++
		}
		return count, true, true
	default:
		count := 0
		for bits != 0 {
			count += int(bits & 1)
			bits >>= 1
		}
		if operation == "parity" {
			count &= 1
		}
		return count, true, true
	}
}

func (p *constantParser) typesCompatible() (int, bool) {
	p.pos++
	if !p.current("(") {
		return 0, false
	}
	end := matchingToken(p.t.src, p.tokens, p.pos, "(", ")")
	if end < 0 {
		return 0, false
	}
	items := splitTopLevel(p.t.src, p.tokens[p.pos+1:end], ",")
	if len(items) != 2 {
		return 0, false
	}
	left, leftOK := p.t.typeFromTokens(items[0])
	right, rightOK := p.t.typeFromTokens(items[1])
	if !leftOK || !rightOK {
		return 0, false
	}
	p.pos = end + 1
	return boolConstant(p.t.compatibleParameterType(left, right)), true
}

func (p *constantParser) constantP() (int, bool) {
	p.pos++
	if !p.current("(") {
		return 0, false
	}
	end := matchingToken(p.t.src, p.tokens, p.pos, "(", ")")
	if end < 0 {
		return 0, false
	}
	argument := p.tokens[p.pos+1 : end]
	_, constant := p.t.constantExpression(argument)
	if !constant {
		_, constant = p.t.inlineRuntimeAnnihilatorCondition(argument, true)
	}
	p.pos = end + 1
	return boolConstant(constant), true
}

func (p *constantParser) offsetof() (int, bool) {
	p.pos++
	if !p.current("(") {
		return 0, false
	}
	end := matchingToken(p.t.src, p.tokens, p.pos, "(", ")")
	if end < 0 {
		return 0, false
	}
	items := splitTopLevel(p.t.src, p.tokens[p.pos+1:end], ",")
	if len(items) != 2 {
		return 0, false
	}
	typeID, ok := p.t.typeFromTokens(items[0])
	if !ok {
		return 0, false
	}
	offset, ok := p.t.fieldOffset(typeID, items[1])
	if !ok {
		return 0, false
	}
	p.pos = end + 1
	return offset, true
}

func (t *translator) convertIntegerConstant(typeID int, value int) (int, bool) {
	info := t.typeInfo(typeID)
	if info.kind == cTypeBool {
		return boolConstant(value != 0), true
	}
	if info.kind != cTypeInt && info.kind != cTypeUint {
		return 0, false
	}
	if info.kind == cTypeInt {
		switch info.size {
		case 1:
			return int(int8(value)), true
		case 2:
			return int(int16(value)), true
		case 4:
			return int(int32(value)), true
		}
		return value, true
	}
	switch info.size {
	case 1:
		return int(uint8(value)), true
	case 2:
		return int(uint16(value)), true
	case 4:
		return int(uint32(value)), true
	}
	return value, true
}

func cBinaryPrecedence(op []byte) int {
	if len(op) == 1 {
		switch op[0] {
		case '|':
			return 3
		case '^':
			return 4
		case '&':
			return 5
		case '<', '>':
			return 7
		case '+', '-':
			return 9
		case '*', '/', '%':
			return 10
		}
	} else if len(op) == 2 {
		if op[0] == op[1] {
			switch op[0] {
			case '|':
				return 1
			case '&':
				return 2
			case '=', '!':
				return 6
			case '<', '>':
				return 8
			}
		}
		if op[1] == '=' {
			if op[0] == '=' || op[0] == '!' {
				return 6
			}
			if op[0] == '<' || op[0] == '>' {
				return 7
			}
		}
	}
	return 0
}

func applyConstantOperator(op []byte, left int, right int, unsigned bool) (int, bool) {
	if len(op) == 1 {
		switch op[0] {
		case '+':
			return left + right, true
		case '-':
			return left - right, true
		case '*':
			return left * right, true
		case '/', '%':
			if right == 0 {
				return 0, false
			}
			if unsigned {
				if op[0] == '/' {
					return int(uint64(left) / uint64(right)), true
				}
				return int(uint64(left) % uint64(right)), true
			}
			if op[0] == '/' {
				return left / right, true
			}
			return left % right, true
		case '&':
			return left & right, true
		case '^':
			return left ^ right, true
		case '|':
			return left | right, true
		case '<':
			if unsigned {
				return boolConstant(uint64(left) < uint64(right)), true
			}
			return boolConstant(left < right), true
		case '>':
			if unsigned {
				return boolConstant(uint64(left) > uint64(right)), true
			}
			return boolConstant(left > right), true
		}
	} else if len(op) == 2 {
		switch op[0] {
		case '<', '>':
			if op[1] == op[0] {
				if right < 0 {
					return 0, false
				}
				if op[0] == '<' {
					return left << uint(right), true
				}
				if unsigned {
					return int(uint64(left) >> uint(right)), true
				}
				return left >> uint(right), true
			}
			if op[0] == '<' {
				if unsigned {
					return boolConstant(uint64(left) <= uint64(right)), true
				}
				return boolConstant(left <= right), true
			}
			if unsigned {
				return boolConstant(uint64(left) >= uint64(right)), true
			}
			return boolConstant(left >= right), true
		case '=':
			return boolConstant(left == right), true
		case '!':
			return boolConstant(left != right), true
		case '&':
			return boolConstant(left != 0 && right != 0), true
		case '|':
			return boolConstant(left != 0 || right != 0), true
		}
	}
	return 0, false
}

func boolConstant(value bool) int {
	if value {
		return 1
	}
	return 0
}
