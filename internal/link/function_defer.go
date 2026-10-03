package link

import "renvo.dev/internal/arena"
import "renvo.dev/internal/unit"

// Register the selected callable in the original function's native defer stack.
// Deferring a tagged dispatcher instead would insert a frame between the defer
// and its callable and change the meaning of a direct recover in that callable.
// Run after closure extraction, when all implementation tags are known.
func lowerFunctionValueDefers(program *unit.Program, signatures []functionValueSignature, transient bool) bool {
	var edits []functionValueEdit
	for tok := 0; tok+3 < len(program.Tokens); tok++ {
		if !functionValueTokenEquals(program, tok, "defer") || !functionValueTokenCharIs(program, tok+2, '(') {
			continue
		}
		sigIndex := -1
		for i := range signatures {
			if functionValueTokenEquals(program, tok+1, "__renvo_call_"+functionValueDecimal(i)) {
				sigIndex = i
				break
			}
		}
		if sigIndex < 0 {
			continue
		}
		close := functionValueFindMatchingParen(program, tok+2)
		if close < 0 {
			return false
		}
		starts, ends := functionValueCommaParts(program, tok+3, close)
		if len(starts) == 0 {
			return false
		}
		sig := signatures[sigIndex]
		stem := "__renvo_defer_" + functionValueDecimal(tok)
		callee := integerRangeName(program, stem+"_callee")
		replacement := "if true { var " + callee + " " + sig.name + " = " + functionValueTokensText(program, starts[0], ends[0]) + "; "
		// The generated dispatcher's signature already contains concrete tagged
		// types for callback arguments. Use those types for argument snapshots.
		var params []string
		for i := len(program.Funcs) - 1; i >= 0; i-- {
			fn := &program.Funcs[i]
			if functionValueTokenEquals(program, fn.NameTok, "__renvo_call_"+functionValueDecimal(sigIndex)) {
				parsed, _, ok := parseFunctionValueCallableSignature(program, fn.NameTok, "")
				if !ok || len(parsed.paramTypes) == 0 {
					return false
				}
				params = parsed.paramTypes[1:]
				break
			}
		}
		if len(params) != len(sig.paramTypes) {
			return false
		}
		prefix, args, ok := functionDeferArguments(program, starts[1:], ends[1:], params, stem)
		if !ok {
			return false
		}
		replacement += prefix
		for i, impl := range sig.storage.impls {
			if i > 0 {
				replacement += " else "
			}
			replacement += "if " + callee + ".kind == " + functionValueDecimal(i+1) + " { defer "
			if impl.method != "" {
				replacement += callee + ".data.(*" + impl.environmentType + ").value." + impl.method + "(" + args + ")"
			} else {
				callArgs := args
				if impl.environmentType != "" {
					callArgs = callee + ".data.(" + impl.receiverType + ")"
					if args != "" {
						callArgs += ", " + args
					}
				}
				replacement += impl.function + "(" + callArgs + ")"
			}
			replacement += " }"
		}
		if len(sig.storage.impls) > 0 {
			replacement += " else { "
		}
		// A nil deferred function faults when invoked, after registration and
		// the rest of the original function have completed.
		replacement += "defer func(){panic(\"call of nil function\")}()"
		if len(sig.storage.impls) > 0 {
			replacement += " }"
		}
		replacement += " }"
		edits = append(edits, functionValueTokenRangeEdit(program, tok, close+1, replacement))
		tok = close
	}
	if len(edits) == 0 {
		return true
	}
	return functionDeferApplyEdits(program, edits, transient)
}

// Evaluate arguments once, left to right, including a sole tuple expression or
// the slice descriptor/element values supplied to a variadic call.
func functionDeferArguments(program *unit.Program, starts []int, ends []int, params []string, stem string) (string, string, bool) {
	prefix := ""
	names := []string{}
	variadic := len(params) > 0 && functionValueHasPrefix(functionValueCompactTypeText(params[len(params)-1]), "...")
	fixed := len(params)
	if variadic {
		fixed--
	}
	if variadic && len(starts) == 1 {
		count := functionDeferTupleArity(program, starts[0], ends[0])
		if count > 1 {
			if count < fixed {
				return "", "", false
			}
			element := functionDeferVariadicElement(params[fixed])
			for i := 0; i < count; i++ {
				typ := element
				if i < fixed {
					typ = params[i]
				}
				name := integerRangeName(program, stem+"_arg_"+functionValueDecimal(i))
				prefix += "var " + name + " " + typ + "; "
				names = append(names, name)
			}
			prefix += functionValueJoin(names, ", ") + " = " + functionValueTokensText(program, starts[0], ends[0]) + "; "
			value := "nil"
			if count > fixed {
				value = "[]" + element + "{" + functionValueJoin(names[fixed:], ", ") + "}"
			}
			name := integerRangeName(program, stem+"_variadic")
			prefix += "var " + name + " []" + element + " = " + value + "; "
			return prefix, functionValueJoin(append(names[:fixed], name+"..."), ", "), true
		}
	}
	if !variadic && len(starts) == 1 && len(params) > 1 {
		for i, typ := range params {
			name := integerRangeName(program, stem+"_arg_"+functionValueDecimal(i))
			prefix += "var " + name + " " + typ + "; "
			names = append(names, name)
		}
		args := functionValueJoin(names, ", ")
		prefix += args + " = " + functionValueTokensText(program, starts[0], ends[0]) + "; "
		return prefix, args, true
	}
	if len(starts) < fixed || !variadic && len(starts) != fixed {
		return "", "", false
	}
	for i := 0; i < fixed; i++ {
		name := integerRangeName(program, stem+"_arg_"+functionValueDecimal(i))
		prefix += "var " + name + " " + params[i] + " = " + functionValueTokensText(program, starts[i], ends[i]) + "; "
		names = append(names, name)
	}
	if variadic {
		element := functionDeferVariadicElement(params[fixed])
		name := integerRangeName(program, stem+"_variadic")
		value := "[]" + element + "{"
		expanded := len(starts) == fixed+1 && ends[fixed]-starts[fixed] >= 3 && functionValueTokenCharIs(program, ends[fixed]-1, '.') && functionValueTokenCharIs(program, ends[fixed]-2, '.') && functionValueTokenCharIs(program, ends[fixed]-3, '.')
		if len(starts) == fixed {
			value = "nil"
		} else if expanded {
			value = functionValueTokensText(program, starts[fixed], ends[fixed]-3)
		} else {
			for i := fixed; i < len(starts); i++ {
				arg := integerRangeName(program, stem+"_arg_"+functionValueDecimal(i))
				prefix += "var " + arg + " " + element + " = " + functionValueTokensText(program, starts[i], ends[i]) + "; "
				if i > fixed {
					value += ", "
				}
				value += arg
			}
			value += "}"
		}
		prefix += "var " + name + " []" + element + " = " + value + "; "
		names = append(names, name+"...")
	}
	return prefix, functionValueJoin(names, ", "), true
}

func functionDeferVariadicElement(typ string) string {
	first := 0
	for first < len(typ) && functionValueIsSpace(typ[first]) {
		first++
	}
	return typ[first+3:]
}

func functionDeferTupleArity(program *unit.Program, start int, end int) int {
	start, end = functionValueUnparen(program, start, end)
	if !functionValueTokenCharIs(program, end-1, ')') {
		return 0
	}
	open := functionValueFindMatchingBackward(program, end-1, "(", ")")
	if open <= start {
		return 0
	}
	if fn, ok := functionValueCalledFunction(program, open); ok {
		sig, _, valid := parseFunctionValueCallableSignature(program, fn.NameTok, "")
		if valid {
			return len(sig.resultTypes)
		}
	}
	typ := functionRangeExpressionType(program, start, start, open)
	sig, ok := functionValueSignatureFromTypeText(ordinaryUnderlyingType(program, typ, 0))
	if ok {
		return len(sig.resultTypes)
	}
	return 0
}

func functionDeferApplyEdits(program *unit.Program, edits []functionValueEdit, transient bool) bool {
	edits = appendFunctionValuePackageEdits(program, edits)
	originalLength := len(program.Text)
	if transient {
		renvo_runtime_ArenaDiscardLinkTokens(program.Tokens)
	}
	text, ok := applyFunctionValueEdits(program.Text, edits)
	if transient {
		arena.DiscardBytes(program.Text)
	}
	return ok && reparseFunctionValueProgramMode(program, text, edits, originalLength, -1, transient)
}
