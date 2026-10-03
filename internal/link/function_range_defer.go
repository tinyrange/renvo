package link

import "renvo.dev/internal/arena"
import "renvo.dev/internal/unit"

// A yield callback is an implementation detail: its body's defers belong to
// the function containing the authored range. Give that invocation one ordered
// list, including registrations outside the loop, before extracting callbacks.
// Typed invocation records snapshot the callee and arguments at registration.
func lowerFunctionRangeDefers(program *unit.Program, transient bool) bool {
	owners := functionRangeDeferOwners(program)
	if len(owners) == 0 {
		return true
	}
	// Builtins cannot be stored as function values. Their existing lowering
	// produces ordinary callables and preserves argument evaluation timing.
	if !lowerDeferredBuiltins(program, transient) {
		return false
	}
	owners = functionRangeDeferOwners(program)
	flush := integerRangeName(program, "__renvo_range_defer_flush")
	generated := "\n//renvo:defer-forward owner\nfunc " + flush + "(list *[]func()) { for _, call := range *list { defer call() } }\n"
	var edits []functionValueEdit
	for _, fn := range owners {
		list := integerRangeName(program, "__renvo_range_defers_"+functionValueDecimal(fn.BodyStart))
		insertion := program.Tokens[fn.BodyStart].Start + program.Tokens[fn.BodyStart].Size
		edits = append(edits, functionValueEdit{start: insertion, end: insertion, text: " var " + list + " *[]func() = new([]func()); defer " + flush + "(" + list + "); "})
		for tok := fn.BodyStart + 1; tok < fn.BodyEnd-1; tok++ {
			if !functionValueTokenEquals(program, tok, "defer") {
				continue
			}
			owner, ok := functionRangeDeferLexicalOwner(program, tok)
			if !ok || owner.BodyStart != fn.BodyStart {
				continue
			}
			end := functionRangeStatementEnd(program, tok, fn.BodyEnd-1)
			if !functionValueTokenCharIs(program, end-1, ')') {
				return false
			}
			open := functionValueFindMatchingBackward(program, end-1, "(", ")")
			if open <= tok+1 {
				return false
			}
			typ := functionRangeExpressionType(program, tok, tok+1, open)
			sig, ok := functionValueSignatureFromTypeText(ordinaryUnderlyingType(program, typ, 0))
			if !ok {
				return false
			}
			stem := "__renvo_range_defer_" + functionValueDecimal(tok)
			callee := integerRangeName(program, stem+"_callee")
			starts, ends := functionValueCommaParts(program, open+1, end-1)
			prefix, _, ok := functionDeferArguments(program, starts, ends, sig.paramTypes, stem)
			if !ok {
				return false
			}
			recordType := integerRangeName(program, stem+"_record")
			record := integerRangeName(program, stem+"_value")
			invoke := integerRangeName(program, stem+"_invoke")
			entries := integerRangeName(program, stem+"_entries")
			fields := "callee " + typ + "; "
			values := "callee: " + callee + ", "
			callArgs := ""
			for n, param := range sig.paramTypes {
				field := "arg_" + functionValueDecimal(n)
				name := integerRangeName(program, stem+"_arg_"+functionValueDecimal(n))
				expansion := ""
				if functionValueHasPrefix(functionValueCompactTypeText(param), "...") {
					param = "[]" + functionDeferVariadicElement(param)
					name = integerRangeName(program, stem+"_variadic")
					expansion = "..."
				}
				fields += field + " " + param + "; "
				values += field + ": " + name + ", "
				if n > 0 {
					callArgs += ", "
				}
				callArgs += "record." + field + expansion
			}
			generated += "type " + recordType + " struct { " + fields + "}\n"
			generated += "//renvo:defer-forward call\nfunc (record *" + recordType + ") Invoke() { record.callee(" + callArgs + ") }\n"
			replacement := "if true { var " + callee + " " + typ + " = " + functionValueTokensText(program, tok+1, open) + "; " + prefix
			replacement += record + " := &" + recordType + "{" + values + "}; var " + invoke + " func() = " + record + ".Invoke; var " + entries + " []func() = *" + list + "; " + entries + " = append(" + entries + ", " + invoke + "); *" + list + " = " + entries + " }"
			edits = append(edits, functionValueTokenRangeEdit(program, tok, end, replacement))
			tok = end - 1
		}
	}
	edits = appendFunctionValuePackageEdits(program, edits)
	originalLength := len(program.Text)
	if transient {
		renvo_runtime_ArenaDiscardLinkTokens(program.Tokens)
	}
	text, ok := applyFunctionValueEditsCapacity(program.Text, edits, len(generated))
	if !ok {
		return false
	}
	generatedStart := len(text)
	text = appendFunctionValueString(text, generated)
	if transient {
		arena.DiscardBytes(program.Text)
	}
	return reparseFunctionValueProgramMode(program, text, edits, originalLength, generatedStart, transient)
}

func functionRangeDeferOwners(program *unit.Program) []unit.Func {
	var owners []unit.Func
	// Avoid signature discovery entirely for the usual units with no defers.
	hasDefer := false
	for tok := 0; tok < len(program.Tokens); tok++ {
		if functionValueTokenEquals(program, tok, "defer") {
			hasDefer = true
			break
		}
	}
	if !hasDefer {
		return owners
	}
	for tok := 0; tok < len(program.Tokens); tok++ {
		if !functionValueTokenKindIs(program, tok, unit.TokenFor) {
			continue
		}
		mark := arena.Mark()
		owner, needed := functionRangeDeferOwner(program, tok)
		arena.Rewind(mark)
		if !needed {
			continue
		}
		found := false
		for _, existing := range owners {
			if existing.BodyStart == owner.BodyStart {
				found = true
			}
		}
		if !found {
			owners = append(owners, owner)
		}
	}
	return owners
}

func functionRangeDeferOwner(program *unit.Program, loop int) (unit.Func, bool) {
	rangeTok, open := functionRangeHeader(program, loop)
	if rangeTok < 0 || open < 0 {
		return unit.Func{}, false
	}
	owner, ok := functionRangeDeferLexicalOwner(program, loop)
	if !ok {
		return unit.Func{}, false
	}
	close := functionValueFindMatchingBrace(program, open)
	for tok := open + 1; tok < close; tok++ {
		if !functionValueTokenEquals(program, tok, "defer") {
			continue
		}
		deferOwner, ok := functionRangeDeferLexicalOwner(program, tok)
		if ok && deferOwner.BodyStart == owner.BodyStart {
			typ := functionRangeExpressionType(program, rangeTok, rangeTok+1, open)
			return owner, functionValueHasPrefix(functionValueCompactTypeText(ordinaryUnderlyingType(program, typ, 0)), "func(")
		}
	}
	return unit.Func{}, false
}

func functionRangeDeferLexicalOwner(program *unit.Program, tok int) (unit.Func, bool) {
	fn, ok := functionValueLexicalFunction(program, tok)
	if !ok {
		return fn, false
	}
	// Original compact declarations delimit the body contents; reparsed
	// declarations and literals delimit the braces. Normalize from the tokens
	// so the first statement and final call argument are never dropped.
	if !functionValueTokenCharIs(program, fn.BodyStart, '{') {
		fn.BodyStart--
	}
	if !functionValueTokenCharIs(program, fn.BodyStart, '{') {
		return fn, false
	}
	close := functionValueFindMatchingBrace(program, fn.BodyStart)
	if close < 0 {
		return fn, false
	}
	fn.BodyEnd = close + 1
	return fn, true
}

// Before range expansion, builtin argument snapshots and method selection may
// need the types of the authored iteration variables. Their concrete types are
// the yield signature's parameters, rather than the iterator's result types.
func functionRangeBindingType(program *unit.Program, name int) string {
	first, last := name, name
	for functionValueTokenCharIs(program, first-1, ',') && first >= 2 && program.Tokens[first-2].KindLine&255 == unit.TokenIdent {
		first -= 2
	}
	for functionValueTokenCharIs(program, last+1, ',') && last+2 < len(program.Tokens) && program.Tokens[last+2].KindLine&255 == unit.TokenIdent {
		last += 2
	}
	if !functionValueTokenKindIs(program, first-1, unit.TokenFor) || !functionValueTokenEquals(program, last+1, ":=") || !functionValueTokenEquals(program, last+2, "range") {
		return ""
	}
	rangeTok, open := functionRangeHeader(program, first-1)
	if open < 0 {
		return ""
	}
	typ := functionRangeExpressionType(program, rangeTok, rangeTok+1, open)
	underlying := ordinaryUnderlyingType(program, typ, 0)
	if !functionValueHasPrefix(functionValueCompactTypeText(underlying), "func(") {
		return ""
	}
	iterator, ok := functionValueSignatureFromTypeText(underlying)
	if !ok || len(iterator.paramTypes) != 1 {
		return ""
	}
	yield, ok := functionValueSignatureFromTypeText(ordinaryUnderlyingType(program, iterator.paramTypes[0], 0))
	index := (name - first) / 2
	if ok && index < len(yield.paramTypes) {
		return yield.paramTypes[index]
	}
	return ""
}
