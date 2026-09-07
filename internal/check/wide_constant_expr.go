package check

import "renvo.dev/internal/syntax"

func wideIntegerLiteral(text string) wideConstant {
	if len(text) == 0 {
		return wideConstant{}
	}
	base, start := 10, 0
	if len(text) > 1 && text[0] == '0' {
		base = 8
		if text[1] == 'x' || text[1] == 'X' {
			base, start = 16, 2
		}
		if text[1] == 'b' || text[1] == 'B' {
			base, start = 2, 2
		}
		if text[1] == 'o' || text[1] == 'O' {
			base, start = 8, 2
		}
	}
	value := wideSmall(0)
	digits := 0
	for i := start; i < len(text); i++ {
		c := text[i]
		if c == '_' {
			continue
		}
		digit := -1
		if c >= '0' && c <= '9' {
			digit = int(c - '0')
		}
		if c >= 'a' && c <= 'f' {
			digit = int(c-'a') + 10
		}
		if c >= 'A' && c <= 'F' {
			digit = int(c-'A') + 10
		}
		if digit < 0 || digit >= base {
			return wideConstant{}
		}
		// This magnitude is private to the parser. Accumulate in place so a
		// long literal does not retain one scratch allocation per digit.
		carry := digit
		for word := 0; word < len(value.words); word++ {
			next := value.words[word]*base + carry
			value.words[word] = next & 32767
			carry = next >> 15
		}
		if carry != 0 {
			value.words = append(value.words, carry)
		}
		digits++
	}
	if digits == 0 {
		return wideConstant{}
	}
	return value
}

// This checker path never uses a truncated intermediate as proof that a type
// is invalid. Unsupported expressions or precision exhaustion stay unknown.
// The precision budget limits scratch allocation, not the target integer size.
func wideConstantExpr(context constantIndexContext, start int, end int, depth int) wideConstant {
	if depth > 64 {
		return wideConstant{}
	}
	file := context.pkg.Files[context.fileIndex].File
	start, end = trimExprSpan(file, start, end)
	start, end = stripOuterParens(file, start, end)
	if start < 0 || start >= end {
		return wideConstant{}
	}
	for precedence := 1; precedence <= 2; precedence++ {
		op := constantIndexOperator(file, start, end, precedence)
		if op < 0 {
			continue
		}
		left := wideConstantExpr(context, start, op, depth+1)
		right := wideConstantExpr(context, op+1, end, depth+1)
		if !left.ok || !right.ok {
			return wideConstant{}
		}
		operator := tokenString(&file, op)
		if operator == "&" || operator == "|" || operator == "^" || operator == "&^" {
			return wideBitwise(left, right, operator)
		}
		if operator == "+" {
			return wideAdd(left, right)
		}
		if operator == "-" {
			return wideAdd(left, wideNegate(right))
		}
		if operator == "*" {
			if len(left.words)+len(right.words) > 4096 {
				return wideConstant{}
			}
			return wideMultiply(left, right)
		}
		if operator == "/" || operator == "%" {
			quotient, remainder := wideDivide(left, right)
			if operator == "/" {
				return quotient
			}
			return remainder
		}
		if operator == "<<" || operator == ">>" {
			count, known := wideInt(right)
			if !known || count < 0 {
				return wideConstant{}
			}
			if operator == "<<" && count/15+len(left.words)+1 > 4096 {
				return wideConstant{}
			}
			return wideShift(left, count, operator == "<<")
		}
		return wideConstant{}
	}
	if tokenTextIs(&file, start, "+") || tokenTextIs(&file, start, "-") || tokenTextIs(&file, start, "^") {
		value := wideConstantExpr(context, start+1, end, depth+1)
		if tokenTextIs(&file, start, "-") {
			return wideNegate(value)
		}
		if tokenTextIs(&file, start, "^") {
			return wideAdd(wideNegate(value), wideSmall(-1))
		}
		return value
	}
	if end-start != 1 {
		return wideConstant{}
	}
	if file.Tokens[start].KindLine&255 == syntax.TokenChar {
		value, ok := syntax.RuneLiteralValue(file.Src, file.Tokens[start])
		if ok {
			return wideSmall(value)
		}
		return wideConstant{}
	}
	if file.Tokens[start].KindLine&255 == syntax.TokenNumber {
		if file.Tokens[start].End-file.Tokens[start].Start > 16384 {
			return wideConstant{}
		}
		return wideIntegerLiteral(tokenString(&file, start))
	}
	if file.Tokens[start].KindLine&255 == syntax.TokenIdent {
		chosen := -1
		for i, binding := range context.bindings {
			if binding.visible <= context.before && context.before < binding.end && coreTokensEqual(&file, binding.name, start) && (chosen < 0 || binding.visible > context.bindings[chosen].visible) {
				chosen = i
			}
		}
		if chosen >= 0 {
			binding := context.bindings[chosen]
			if !binding.constant || binding.typeEnd > binding.typeStart {
				return wideConstant{}
			}
			// Resolve the initializer at its declaration, not at the use site.
			// An omitted expression keeps its template but receives a new iota.
			context.before = binding.name
			context.scope = CoreScope{}
			context.iotaKnown, context.iotaValue = true, binding.iotaValue
			return wideConstantExpr(context, binding.valueStart, binding.valueEnd, depth+1)
		}
		if lookupScopeTokenNameCore(context.scope, &file, start) >= 0 {
			return wideConstant{}
		}
		if context.iotaKnown && tokenTextIs(&file, start, "iota") && LookupPackageSymbol(*context.info, "iota") < 0 {
			return wideSmall(context.iotaValue)
		}
		index := LookupDecl(*context.info, tokenString(&file, start))
		if index < 0 {
			return wideConstant{}
		}
		decl := context.info.Decls[index]
		if decl.Kind != SymbolConst {
			return wideConstant{}
		}
		return wideDeclaredConstant(context, decl, depth+1)
	}
	return wideConstant{}
}
