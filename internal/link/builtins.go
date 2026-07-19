package link

import (
	"renvo.dev/internal/syntax"
	"renvo.dev/internal/unit"
)

func lowerOrdinaryBuiltins(program *unit.Program) bool {
	stringLess := ""
	generated := ""
	generatedCount := 0
	for {
		changed := false
		for i := len(program.Tokens) - 2; i >= 0; i-- {
			name := functionValueTokenText(program, i)
			if (name != "min" && name != "max" && name != "clear") || !functionValueTokenEquals(program, i+1, "(") || ordinaryBuiltinShadowed(program, i, name) {
				continue
			}
			close := functionValueFindMatchingParen(program, i+1)
			if close < 0 {
				return false
			}
			starts, ends := ordinaryBuiltinArguments(program, i+2, close)
			replacement := ""
			if name == "clear" {
				if len(starts) != 1 {
					return false
				}
				replacement = ordinaryClearText(program, i, starts[0], ends[0])
			} else {
				if len(starts) == 0 {
					return false
				}
				if constant := ordinaryConstantMinMax(program, name, starts, ends); constant != "" {
					replacement = constant
				} else {
					typ := ""
					for arg := 0; arg < len(starts); arg++ {
						candidate := ordinaryBuiltinExprType(program, i, starts[arg], ends[arg])
						if candidate != "" && !ordinaryUntypedExpression(program, starts[arg], ends[arg]) {
							typ = candidate
							break
						}
					}
					if typ == "" {
						typ = ordinaryBuiltinExprType(program, i, starts[0], ends[0])
					}
					if typ == "" {
						return false
					}
					underlying := ordinaryUnderlyingType(program, typ, 0)
					if underlying == "string" {
						if stringLess == "" {
							stringLess = ordinaryBuiltinGeneratedName(program, "__renvo_builtin_string_less")
						}
					}
					var declaration string
					replacement, declaration = ordinaryMinMaxText(program, name, typ, underlying, stringLess, generatedCount, starts, ends)
					generated += declaration
					generatedCount++
				}
			}
			if replacement == "" {
				return false
			}
			text, ok := applyFunctionValueEdits(program.Text, []functionValueEdit{functionValueTokenRangeEdit(program, i, close+1, replacement)})
			if !ok || !reparseFunctionValueProgram(program, text) {
				return false
			}
			changed = true
			break
		}
		if !changed {
			break
		}
	}
	if stringLess == "" && generated == "" {
		return true
	}
	text := program.Text
	if len(text) > 0 && text[len(text)-1] != '\n' {
		text = append(text, '\n')
	}
	text = appendFunctionValueString(text, generated)
	if stringLess != "" {
		help := "func " + stringLess + "(left string, right string) bool { limit := len(left); if len(right) < limit { limit = len(right) }; for index := 0; index < limit; index++ { if left[index] < right[index] { return true }; if left[index] > right[index] { return false } }; return len(left) < len(right) }\n"
		text = appendFunctionValueString(text, help)
	}
	return reparseFunctionValueProgram(program, text)
}

func ordinaryBuiltinArguments(program *unit.Program, start int, close int) ([]int, []int) {
	var starts []int
	var ends []int
	part := start
	paren := 0
	bracket := 0
	brace := 0
	for i := start; i <= close; i++ {
		text := functionValueTokenText(program, i)
		if i < close {
			if text == "(" {
				paren++
			} else if text == ")" {
				paren--
			} else if text == "[" {
				bracket++
			} else if text == "]" {
				bracket--
			} else if text == "{" {
				brace++
			} else if text == "}" {
				brace--
			}
		}
		if i == close || text == "," && paren == 0 && bracket == 0 && brace == 0 {
			if part < i {
				starts = append(starts, part)
				ends = append(ends, i)
			}
			part = i + 1
		}
	}
	return starts, ends
}

func ordinaryMinMaxText(program *unit.Program, name string, typ string, underlying string, stringLess string, call int, starts []int, ends []int) (string, string) {
	prefix := "__renvo_builtin_" + name + "_" + functionValueDecimal(call)
	typeName := prefix + "_arguments"
	functionName := prefix + "_call"
	declaration := "type " + typeName + " struct { "
	callText := functionName + "(" + typeName + "{"
	for i := 0; i < len(starts); i++ {
		field := "value" + functionValueDecimal(i)
		declaration += field + " " + typ + "; "
		if i > 0 {
			callText += ", "
		}
		callText += field + ": " + functionValueTokensText(program, starts[i], ends[i])
	}
	declaration += "}\nfunc " + functionName + "(values " + typeName + ") " + typ + " { result := values.value0; "
	for i := 1; i < len(starts); i++ {
		value := "values.value" + functionValueDecimal(i)
		less := value + " < result"
		if stringLess != "" {
			less = stringLess + "(" + value + ", result)"
		}
		if name == "max" {
			if stringLess != "" {
				less = stringLess + "(result, " + value + ")"
			} else {
				less = "result < " + value
			}
		}
		if underlying == "float32" || underlying == "float64" {
			declaration += "if " + value + " != " + value + " { return " + value + " }; "
		}
		declaration += "if " + less + " { result = " + value + " }; "
	}
	declaration += "return result }\n"
	return callText + "})", declaration
}

func ordinaryClearText(program *unit.Program, call int, start int, end int) string {
	typ := ordinaryBuiltinExprType(program, call, start, end)
	underlying := ordinaryUnderlyingType(program, typ, 0)
	prefix := "__renvo_builtin_" + functionValueDecimal(call)
	expr := functionValueTokensText(program, start, end)
	if functionValueHasPrefix(underlying, "map[") {
		return "{ " + prefix + " := " + expr + "; for " + prefix + "_key := range " + prefix + " { delete(" + prefix + ", " + prefix + "_key) } }"
	}
	if functionValueHasPrefix(underlying, "[]") {
		elem := underlying[2:]
		return "{ " + prefix + " := " + expr + "; var " + prefix + "_zero " + elem + "; for " + prefix + "_index := range " + prefix + " { " + prefix + "[" + prefix + "_index] = " + prefix + "_zero } }"
	}
	return ""
}

func ordinaryBuiltinExprType(program *unit.Program, before int, start int, end int) string {
	for end-start >= 2 && functionValueTokenEquals(program, start, "(") && functionValueFindMatchingParen(program, start) == end-1 {
		start++
		end--
	}
	if end <= start {
		return ""
	}
	if end-start == 1 {
		kind := program.Tokens[start].Kind
		if kind == unit.TokenString {
			return "string"
		}
		if kind == unit.TokenFloat {
			return "float64"
		}
		if kind == unit.TokenNumber || kind == unit.TokenChar {
			return "int"
		}
		name := functionValueTokenText(program, start)
		if name == "true" || name == "false" {
			return "bool"
		}
		if typ := functionValueEnclosingLocalType(program, before, name); typ != "" {
			return typ
		}
		return ordinaryGlobalType(program, name)
	}
	if (functionValueTokenEquals(program, start, "+") || functionValueTokenEquals(program, start, "-") || functionValueTokenEquals(program, start, "^")) && start+1 < end {
		return ordinaryBuiltinExprType(program, before, start+1, end)
	}
	if program.Tokens[start].Kind == unit.TokenIdent && functionValueTokenEquals(program, start+1, "(") && functionValueFindMatchingParen(program, start+1) == end-1 {
		name := functionValueTokenText(program, start)
		if name == "make" {
			starts, ends := ordinaryBuiltinArguments(program, start+2, end-1)
			if len(starts) > 0 {
				return functionValueTokensText(program, starts[0], ends[0])
			}
		}
		if name == "min" || name == "max" {
			starts, ends := ordinaryBuiltinArguments(program, start+2, end-1)
			for i := 0; i < len(starts); i++ {
				if typ := ordinaryBuiltinExprType(program, before, starts[i], ends[i]); typ != "" {
					return typ
				}
			}
		}
		if ordinaryBuiltinTypeName(name) {
			return name
		}
		return functionValueDeclaredFunctionResultType(program, name)
	}
	paren := 0
	bracket := 0
	brace := 0
	for i := start; i < end; i++ {
		text := functionValueTokenText(program, i)
		if text == "(" {
			paren++
		} else if text == ")" {
			paren--
		} else if text == "[" {
			bracket++
		} else if text == "]" {
			bracket--
		} else if text == "{" {
			brace++
		} else if text == "}" {
			brace--
		} else if paren == 0 && bracket == 0 && brace == 0 && (text == "+" || text == "-" || text == "*" || text == "/" || text == "%" || text == "&" || text == "|" || text == "^" || text == "<<" || text == ">>") {
			return ordinaryBuiltinExprType(program, before, start, i)
		}
	}
	return ""
}

func ordinaryBuiltinTypeName(name string) bool {
	return name == "string" || name == "float32" || name == "float64" || name == "int" || name == "int8" || name == "int16" || name == "int32" || name == "int64" || name == "uint" || name == "uint8" || name == "uint16" || name == "uint32" || name == "uint64" || name == "uintptr" || name == "byte" || name == "rune"
}

func ordinaryGlobalType(program *unit.Program, name string) string {
	for i := 0; i < len(program.Decls); i++ {
		decl := program.Decls[i]
		if functionValueTokenText(program, functionValueTokenAtSpan(program, decl.NameStart, decl.NameEnd)) != name {
			continue
		}
		nameTok := functionValueTokenAtSpan(program, decl.NameStart, decl.NameEnd)
		start := nameTok + 1
		end := functionValueTypeEnd(program, start)
		if end > start && !functionValueTokenEquals(program, start, "=") {
			return functionValueTokensText(program, start, end)
		}
		if functionValueTokenEquals(program, start, "=") {
			return ordinaryBuiltinExprType(program, nameTok, start+1, decl.EndTok)
		}
	}
	return ""
}

func ordinaryUnderlyingType(program *unit.Program, typ string, depth int) string {
	if depth > len(program.Decls)+1 || ordinaryBuiltinTypeName(typ) || functionValueHasPrefix(typ, "[]") || functionValueHasPrefix(typ, "map[") {
		return typ
	}
	for i := 0; i < len(program.Decls); i++ {
		decl := program.Decls[i]
		nameTok := functionValueTokenAtSpan(program, decl.NameStart, decl.NameEnd)
		if functionValueTokenText(program, nameTok) != typ {
			continue
		}
		start := nameTok + 1
		if functionValueTokenEquals(program, start, "=") {
			start++
		}
		return ordinaryUnderlyingType(program, functionValueTokensText(program, start, decl.EndTok), depth+1)
	}
	return typ
}

func ordinaryUntypedExpression(program *unit.Program, start int, end int) bool {
	for end-start >= 2 && functionValueTokenEquals(program, start, "(") && functionValueFindMatchingParen(program, start) == end-1 {
		start++
		end--
	}
	if end-start == 1 {
		kind := program.Tokens[start].Kind
		return kind == unit.TokenNumber || kind == unit.TokenFloat || kind == unit.TokenString || kind == unit.TokenChar
	}
	return false
}

func ordinaryConstantMinMax(program *unit.Program, name string, starts []int, ends []int) string {
	best := 0
	values := make([]ordinaryBuiltinConstant, len(starts))
	for i := 0; i < len(starts); i++ {
		values[i] = ordinaryConstantValue(program, starts[i], ends[i], 0)
		if !values[i].ok || i > 0 && values[i].kind != values[0].kind {
			return ""
		}
	}
	commonType := ""
	floating := false
	for i := 0; i < len(values); i++ {
		if values[i].typ != "" {
			if commonType != "" && commonType != values[i].typ {
				return ""
			}
			commonType = values[i].typ
		}
		floating = floating || values[i].floating
	}
	if len(values) == 0 {
		return ""
	}
	for i := 1; i < len(starts); i++ {
		comparison := ordinaryConstantNumberCompare(values[i].number, values[best].number)
		if values[0].kind == 2 {
			comparison = 0
			if ordinaryStringLess(values[i].text, values[best].text) {
				comparison = -1
			} else if values[i].text != values[best].text {
				comparison = 1
			}
		}
		selectNext := comparison < 0
		if name == "max" {
			selectNext = comparison > 0
		}
		if selectNext {
			best = i
		}
	}
	result := functionValueTokensText(program, starts[best], ends[best])
	if commonType != "" && values[best].typ != commonType {
		return commonType + "(" + result + ")"
	}
	if commonType == "" && floating && !values[best].floating {
		return "(" + result + " + 0.0)"
	}
	return result
}

func ordinaryStringLess(left string, right string) bool {
	limit := len(left)
	if len(right) < limit {
		limit = len(right)
	}
	for i := 0; i < limit; i++ {
		if left[i] < right[i] {
			return true
		}
		if left[i] > right[i] {
			return false
		}
	}
	return len(left) < len(right)
}

type ordinaryBuiltinConstant struct {
	kind     int
	number   ordinaryConstantNumber
	text     string
	typ      string
	floating bool
	ok       bool
}

type ordinaryConstantNumber struct {
	negative bool
	digits   string
	scale    int
}

func ordinaryConstantValue(program *unit.Program, start int, end int, depth int) ordinaryBuiltinConstant {
	if depth > len(program.Decls)+8 {
		return ordinaryBuiltinConstant{}
	}
	for end-start >= 2 && functionValueTokenEquals(program, start, "(") && functionValueFindMatchingParen(program, start) == end-1 {
		start++
		end--
	}
	if start < 0 || end <= start || end > len(program.Tokens) {
		return ordinaryBuiltinConstant{}
	}
	if program.Tokens[start].Kind == unit.TokenIdent && start+2 < end && functionValueTokenEquals(program, start+1, "(") && functionValueFindMatchingParen(program, start+1) == end-1 {
		typ := functionValueTokenText(program, start)
		if ordinaryBuiltinTypeName(typ) || functionValueDeclaredType(program, typ) {
			value := ordinaryConstantValue(program, start+2, end-1, depth+1)
			value.typ = typ
			return value
		}
	}
	if end-start == 1 && program.Tokens[start].Kind == unit.TokenString {
		tok := program.Tokens[start]
		value, ok := syntax.StringLiteralValue(program.Text, syntax.Token{Kind: syntax.TokenString, Start: tok.Start, End: tok.Start + tok.Size, Line: tok.Line})
		return ordinaryBuiltinConstant{kind: 2, text: value, ok: ok}
	}
	if end-start == 1 && (program.Tokens[start].Kind == unit.TokenNumber || program.Tokens[start].Kind == unit.TokenFloat) {
		number, floating, ok := ordinaryConstantParseNumber(functionValueTokenText(program, start))
		return ordinaryBuiltinConstant{kind: 1, number: number, floating: floating, ok: ok}
	}
	if end-start == 1 && program.Tokens[start].Kind == unit.TokenIdent {
		return ordinaryConstantNamedValue(program, functionValueTokenText(program, start), depth+1)
	}
	if (functionValueTokenEquals(program, start, "+") || functionValueTokenEquals(program, start, "-")) && start+1 < end {
		value := ordinaryConstantValue(program, start+1, end, depth+1)
		if value.ok && value.kind == 1 && functionValueTokenEquals(program, start, "-") && value.number.digits != "0" {
			value.number.negative = !value.number.negative
		}
		return value
	}
	paren, bracket, brace := 0, 0, 0
	for i := end - 1; i >= start; i-- {
		text := functionValueTokenText(program, i)
		if text == ")" {
			paren++
		} else if text == "(" {
			paren--
		} else if text == "]" {
			bracket++
		} else if text == "[" {
			bracket--
		} else if text == "}" {
			brace++
		} else if text == "{" {
			brace--
		} else if paren == 0 && bracket == 0 && brace == 0 && (text == "<<" || text == ">>") {
			left := ordinaryConstantValue(program, start, i, depth+1)
			right := ordinaryConstantValue(program, i+1, end, depth+1)
			shift, ok := ordinaryConstantSmallNonnegative(right)
			if !left.ok || left.kind != 1 || !ok {
				return ordinaryBuiltinConstant{}
			}
			left.number = ordinaryConstantShift(left.number, shift, text == ">>")
			left.floating = false
			return left
		}
	}
	return ordinaryBuiltinConstant{}
}

func ordinaryConstantNamedValue(program *unit.Program, name string, depth int) ordinaryBuiltinConstant {
	for i := 0; i < len(program.Decls); i++ {
		decl := program.Decls[i]
		if decl.Kind != unit.TokenConst {
			continue
		}
		nameTok := functionValueTokenAtSpan(program, decl.NameStart, decl.NameEnd)
		if nameTok < 0 || functionValueTokenText(program, nameTok) != name {
			continue
		}
		assign := -1
		for tok := nameTok + 1; tok < decl.EndTok; tok++ {
			if functionValueTokenEquals(program, tok, "=") {
				assign = tok
				break
			}
		}
		if assign < 0 || assign+1 >= decl.EndTok {
			return ordinaryBuiltinConstant{}
		}
		value := ordinaryConstantValue(program, assign+1, decl.EndTok, depth+1)
		if assign > nameTok+1 {
			value.typ = functionValueTokensText(program, nameTok+1, assign)
		}
		return value
	}
	return ordinaryBuiltinConstant{}
}

func ordinaryConstantParseNumber(text string) (ordinaryConstantNumber, bool, bool) {
	clean := ordinaryConstantRemoveUnderscores(text)
	floating := false
	for i := 0; i < len(clean); i++ {
		if clean[i] == '.' || clean[i] == 'e' || clean[i] == 'E' || clean[i] == 'p' || clean[i] == 'P' {
			floating = true
			break
		}
	}
	if len(clean) > 2 && clean[0] == '0' && (clean[1] == 'x' || clean[1] == 'X') {
		return ordinaryConstantParseHex(clean, floating)
	}
	if floating {
		return ordinaryConstantParseDecimalFloat(clean)
	}
	base := 10
	start := 0
	if len(clean) > 2 && clean[0] == '0' {
		if clean[1] == 'b' || clean[1] == 'B' {
			base, start = 2, 2
		} else if clean[1] == 'o' || clean[1] == 'O' {
			base, start = 8, 2
		}
	}
	if base == 10 && len(clean)-start > 1 && clean[start] == '0' {
		base, start = 8, start+1
	}
	digits, ok := ordinaryConstantDigitsInBase(clean, start, len(clean), base)
	return ordinaryConstantNormalize(ordinaryConstantNumber{digits: digits}), false, ok
}

func ordinaryConstantParseDecimalFloat(text string) (ordinaryConstantNumber, bool, bool) {
	exponent := 0
	exponentAt := len(text)
	for i := 0; i < len(text); i++ {
		if text[i] == 'e' || text[i] == 'E' {
			exponentAt = i
			var ok bool
			exponent, ok = ordinaryConstantSignedSmall(text, i+1, len(text))
			if !ok {
				return ordinaryConstantNumber{}, true, false
			}
			break
		}
	}
	digits := ""
	fraction := 0
	dot := false
	for i := 0; i < exponentAt; i++ {
		if text[i] == '.' {
			if dot {
				return ordinaryConstantNumber{}, true, false
			}
			dot = true
			continue
		}
		if text[i] < '0' || text[i] > '9' {
			return ordinaryConstantNumber{}, true, false
		}
		digits += text[i : i+1]
		if dot {
			fraction++
		}
	}
	if digits == "" {
		return ordinaryConstantNumber{}, true, false
	}
	return ordinaryConstantNormalize(ordinaryConstantNumber{digits: digits, scale: exponent - fraction}), true, true
}

func ordinaryConstantParseHex(text string, floating bool) (ordinaryConstantNumber, bool, bool) {
	end := len(text)
	binaryExponent := 0
	for i := 2; i < len(text); i++ {
		if text[i] == 'p' || text[i] == 'P' {
			end = i
			var ok bool
			binaryExponent, ok = ordinaryConstantSignedSmall(text, i+1, len(text))
			if !ok {
				return ordinaryConstantNumber{}, floating, false
			}
			break
		}
	}
	hex := ""
	fraction := 0
	dot := false
	for i := 2; i < end; i++ {
		if text[i] == '.' {
			if dot {
				return ordinaryConstantNumber{}, floating, false
			}
			dot = true
			continue
		}
		hex += text[i : i+1]
		if dot {
			fraction++
		}
	}
	digits, ok := ordinaryConstantDigitsInBase(hex, 0, len(hex), 16)
	if !ok {
		return ordinaryConstantNumber{}, floating, false
	}
	number := ordinaryConstantNumber{digits: digits}
	binaryExponent -= fraction * 4
	if binaryExponent >= 0 {
		for i := 0; i < binaryExponent; i++ {
			number.digits = ordinaryConstantMultiplySmall(number.digits, 2)
		}
	} else {
		for i := 0; i < -binaryExponent; i++ {
			number.digits = ordinaryConstantMultiplySmall(number.digits, 5)
			number.scale--
		}
	}
	return ordinaryConstantNormalize(number), floating, true
}

func ordinaryConstantDigitsInBase(text string, start int, end int, base int) (string, bool) {
	if start >= end {
		return "", false
	}
	digits := "0"
	for i := start; i < end; i++ {
		digit := ordinaryConstantDigit(text[i])
		if digit < 0 || digit >= base {
			return "", false
		}
		digits = ordinaryConstantMultiplySmall(digits, base)
		digits = ordinaryConstantAddSmall(digits, digit)
	}
	return digits, true
}

func ordinaryConstantDigit(c byte) int {
	if c >= '0' && c <= '9' {
		return int(c - '0')
	}
	if c >= 'a' && c <= 'f' {
		return int(c-'a') + 10
	}
	if c >= 'A' && c <= 'F' {
		return int(c-'A') + 10
	}
	return -1
}

func ordinaryConstantMultiplySmall(digits string, multiplier int) string {
	if digits == "0" || multiplier == 0 {
		return "0"
	}
	out := make([]byte, len(digits)+2)
	write := len(out)
	carry := 0
	for i := len(digits) - 1; i >= 0; i-- {
		value := int(digits[i]-'0')*multiplier + carry
		write--
		out[write] = byte(value%10) + '0'
		carry = value / 10
	}
	for carry > 0 {
		write--
		out[write] = byte(carry%10) + '0'
		carry /= 10
	}
	return string(out[write:])
}

func ordinaryConstantAddSmall(digits string, add int) string {
	out := []byte(digits)
	for i := len(out) - 1; i >= 0 && add > 0; i-- {
		value := int(out[i]-'0') + add
		out[i] = byte(value%10) + '0'
		add = value / 10
	}
	for add > 0 {
		out = append([]byte{byte(add%10) + '0'}, out...)
		add /= 10
	}
	return string(out)
}

func ordinaryConstantNormalize(number ordinaryConstantNumber) ordinaryConstantNumber {
	start := 0
	for start+1 < len(number.digits) && number.digits[start] == '0' {
		start++
	}
	number.digits = number.digits[start:]
	for len(number.digits) > 1 && number.digits[len(number.digits)-1] == '0' {
		number.digits = number.digits[:len(number.digits)-1]
		number.scale++
	}
	if number.digits == "" || number.digits == "0" {
		number.digits = "0"
		number.scale = 0
		number.negative = false
	}
	return number
}

func ordinaryConstantNumberCompare(left ordinaryConstantNumber, right ordinaryConstantNumber) int {
	if left.negative != right.negative {
		if left.negative {
			return -1
		}
		return 1
	}
	comparison := ordinaryConstantMagnitudeCompare(left, right)
	if left.negative {
		return -comparison
	}
	return comparison
}

func ordinaryConstantMagnitudeCompare(left ordinaryConstantNumber, right ordinaryConstantNumber) int {
	leftExponent := len(left.digits) + left.scale
	rightExponent := len(right.digits) + right.scale
	if leftExponent < rightExponent {
		return -1
	}
	if leftExponent > rightExponent {
		return 1
	}
	limit := len(left.digits)
	if len(right.digits) > limit {
		limit = len(right.digits)
	}
	for i := 0; i < limit; i++ {
		leftDigit, rightDigit := byte('0'), byte('0')
		if i < len(left.digits) {
			leftDigit = left.digits[i]
		}
		if i < len(right.digits) {
			rightDigit = right.digits[i]
		}
		if leftDigit < rightDigit {
			return -1
		}
		if leftDigit > rightDigit {
			return 1
		}
	}
	return 0
}

func ordinaryConstantSmallNonnegative(value ordinaryBuiltinConstant) (int, bool) {
	if !value.ok || value.kind != 1 || value.number.negative || value.number.scale < 0 {
		return 0, false
	}
	result := 0
	for i := 0; i < len(value.number.digits); i++ {
		if result > 65536 {
			return 0, false
		}
		result = result*10 + int(value.number.digits[i]-'0')
	}
	for i := 0; i < value.number.scale; i++ {
		if result > 65536 {
			return 0, false
		}
		result *= 10
	}
	return result, true
}

func ordinaryConstantShift(number ordinaryConstantNumber, shift int, right bool) ordinaryConstantNumber {
	digits := number.digits
	for number.scale > 0 {
		digits += "0"
		number.scale--
	}
	if right {
		for i := 0; i < shift; i++ {
			digits = ordinaryConstantDivideSmall(digits, 2)
		}
	} else {
		for i := 0; i < shift; i++ {
			digits = ordinaryConstantMultiplySmall(digits, 2)
		}
	}
	number.digits = digits
	return ordinaryConstantNormalize(number)
}

func ordinaryConstantDivideSmall(digits string, divisor int) string {
	out := make([]byte, len(digits))
	carry := 0
	write := 0
	for i := 0; i < len(digits); i++ {
		value := carry*10 + int(digits[i]-'0')
		quotient := value / divisor
		carry = value % divisor
		if quotient != 0 || write > 0 {
			out[write] = byte(quotient) + '0'
			write++
		}
	}
	if write == 0 {
		return "0"
	}
	return string(out[:write])
}

func ordinaryConstantSignedSmall(text string, start int, end int) (int, bool) {
	negative := false
	if start < end && (text[start] == '+' || text[start] == '-') {
		negative = text[start] == '-'
		start++
	}
	if start >= end {
		return 0, false
	}
	value := 0
	for i := start; i < end; i++ {
		if text[i] < '0' || text[i] > '9' || value > 65536 {
			return 0, false
		}
		value = value*10 + int(text[i]-'0')
	}
	if negative {
		value = -value
	}
	return value, true
}

func ordinaryConstantRemoveUnderscores(text string) string {
	out := make([]byte, 0, len(text))
	for i := 0; i < len(text); i++ {
		if text[i] != '_' {
			out = append(out, text[i])
		}
	}
	return string(out)
}

func ordinaryBuiltinGeneratedName(program *unit.Program, base string) string {
	name := base
	for ordinaryBuiltinTopLevelObject(program, name) {
		name += "_"
	}
	return name
}

func ordinaryBuiltinShadowed(program *unit.Program, at int, name string) bool {
	for i := 0; i < len(program.Funcs); i++ {
		if program.Funcs[i].NameTok == at {
			return true
		}
	}
	if at > 0 && functionValueTokenEquals(program, at-1, ".") {
		return true
	}
	if ordinaryBuiltinTopLevelObject(program, name) || functionValueEnclosingLocalType(program, at, name) != "" {
		return true
	}
	fnIndex := -1
	for i := 0; i < len(program.Funcs); i++ {
		if program.Funcs[i].BodyStart < at && at < program.Funcs[i].BodyEnd {
			fnIndex = i
			break
		}
	}
	if fnIndex < 0 {
		return false
	}
	// Track the active lexical blocks so declarations in an earlier sibling
	// block do not hide a predeclared builtin at the call site.
	active := []bool{false}
	for i := program.Funcs[fnIndex].BodyStart + 1; i < at; i++ {
		if functionValueTokenEquals(program, i, "{") {
			active = append(active, false)
			continue
		}
		if functionValueTokenEquals(program, i, "}") {
			if len(active) > 1 {
				active = active[:len(active)-1]
			}
			continue
		}
		if functionValueTokenEquals(program, i, name) && ordinaryBuiltinLocalDeclaration(program, i, at) {
			active[len(active)-1] = true
		}
	}
	for i := 0; i < len(active); i++ {
		if active[i] {
			return true
		}
	}
	return false
}

func ordinaryBuiltinTopLevelObject(program *unit.Program, name string) bool {
	for i := 0; i < len(program.Funcs); i++ {
		fn := program.Funcs[i]
		if fn.ReceiverStart >= fn.ReceiverEnd && functionValueTokenText(program, fn.NameTok) == name {
			return true
		}
	}
	for i := 0; i < len(program.Decls); i++ {
		decl := program.Decls[i]
		nameTok := functionValueTokenAtSpan(program, decl.NameStart, decl.NameEnd)
		if nameTok >= 0 && functionValueTokenText(program, nameTok) == name {
			return true
		}
	}
	return false
}

func ordinaryBuiltinLocalDeclaration(program *unit.Program, nameTok int, limit int) bool {
	if nameTok > 0 && (functionValueTokenEquals(program, nameTok-1, "var") || functionValueTokenEquals(program, nameTok-1, "const") || functionValueTokenEquals(program, nameTok-1, "type")) {
		return true
	}
	paren, bracket := 0, 0
	for i := nameTok + 1; i < limit; i++ {
		text := functionValueTokenText(program, i)
		if text == "(" {
			paren++
		} else if text == ")" {
			if paren == 0 {
				return false
			}
			paren--
		} else if text == "[" {
			bracket++
		} else if text == "]" {
			bracket--
		} else if paren == 0 && bracket == 0 {
			if text == ":=" {
				return true
			}
			if text == ";" || text == "{" || text == "}" || text == "=" {
				return false
			}
		}
	}
	return false
}
