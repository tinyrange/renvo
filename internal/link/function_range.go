package link

import "renvo.dev/internal/arena"
import "renvo.dev/internal/syntax"
import "renvo.dev/internal/unit"

// Function ranges are expanded before closure conversion. The callback's
// parameters are the iteration bindings, so := creates a fresh binding on each
// invocation, while = evaluates the authored assignment inside the callback.
func lowerFunctionRangesCore(program *unit.Program, transient bool) bool {
	return lowerRangesCore(program, transient, true, false)
}

// Classify each range expression once. In particular, ordinary slice ranges
// must not repeat lexical type resolution in separate iterator/integer passes.
func lowerRangesCore(program *unit.Program, transient bool, functions bool, integers bool) bool {
	count := 0
	for {
		changed := false
		for at := len(program.Tokens) - 1; at >= 0; at-- {
			if program.Tokens[at].KindLine&255 != unit.TokenFor {
				continue
			}
			rangeTok, open := functionRangeHeader(program, at)
			if rangeTok < 0 || open < 0 {
				continue
			}
			typ := ordinaryBuiltinExprType(program, rangeTok, rangeTok+1, open)
			if typ == "" && functions {
				typ = functionRangeExpressionFallbackType(program, rangeTok, rangeTok+1, open)
			}
			underlying := ordinaryUnderlyingType(program, typ, 0)
			if integers && mapLowerIntegerKey(underlying) {
				if !lowerIntegerRangeAt(program, transient, at, rangeTok, open, typ, count) {
					return false
				}
				count++
				changed = true
				break
			}
			if !functions || !functionValueHasPrefix(functionValueCompactTypeText(underlying), "func(") {
				continue
			}
			iterator, ok := functionValueSignatureFromTypeText(underlying)
			if !ok || len(iterator.paramTypes) != 1 || len(iterator.resultTypes) != 0 {
				return false
			}
			yield, ok := functionValueSignatureFromTypeText(ordinaryUnderlyingType(program, iterator.paramTypes[0], 0))
			if !ok || len(yield.paramTypes) > 2 || len(yield.resultTypes) != 1 || ordinaryUnderlyingType(program, yield.resultTypes[0], 0) != "bool" {
				return false
			}
			close := functionValueFindMatchingBrace(program, open)
			if close < 0 {
				return false
			}
			stem := "__renvo_function_range_" + functionValueDecimal(count)
			state := integerRangeName(program, stem+"_state")
			sequence := integerRangeName(program, stem+"_sequence")
			previous := integerRangeName(program, stem+"_previous")
			action := integerRangeName(program, stem+"_action")
			params := ""
			values := ""
			for n := 0; n < len(yield.paramTypes); n++ {
				name := integerRangeName(program, stem+"_value_"+functionValueDecimal(n))
				if n > 0 {
					params += ", "
					values += ", "
				}
				params += name + " " + yield.paramTypes[n]
				values += name
			}
			binding := ""
			if rangeTok != at+1 {
				assign := concurrencyTopLevelAssignment(program, at+1, rangeTok)
				if assign < 0 {
					return false
				}
				starts, ends := functionValueCommaParts(program, at+1, assign)
				if len(starts) > len(yield.paramTypes) {
					return false
				}
				values = ""
				allBlank := true
				for n := 0; n < len(starts); n++ {
					if ends[n] != starts[n]+1 || !functionValueTokenEquals(program, starts[n], "_") {
						allBlank = false
					}
					if n > 0 {
						values += ", "
					}
					values += integerRangeName(program, stem+"_value_"+functionValueDecimal(n))
				}
				op := functionValueTokenText(program, assign)
				if allBlank {
					op = "="
				}
				binding = functionValueTokensText(program, at+1, assign) + " " + op + " " + values + "; "
			}
			label := ""
			if at > 1 && functionValueTokenEquals(program, at-1, ":") {
				label = functionValueTokenText(program, at-2)
			}
			body, actions, returnPrefix, ok := functionRangeBody(program, at, open, close, label, state, action, stem)
			if !ok {
				return false
			}
			prefix := "if true { "
			if label != "" {
				once := integerRangeName(program, stem+"_once")
				prefix = "for " + once + " := false; !" + once + "; " + once + " = true { "
			}
			replacement := prefix + returnPrefix + "var " + state + " int = 1; "
			if len(actions) > 0 {
				replacement += "var " + action + " int; "
			}
			replacement += "var " + sequence + " " + typ + " = " + functionValueTokensText(program, rangeTok+1, open) + "; " + sequence + "(func(" + params + ") bool { "
			// Mark an invalid invocation as panicking too, so an iterator that
			// recovers the checking panic cannot silently complete the loop.
			replacement += previous + " := " + state + "; " + state + " = 2; if " + previous + " != 1 { "
			replacement += "if " + previous + " == 0 { panic(\"range function continued iteration after function for loop body returned false\") }; "
			replacement += "if " + previous + " == 3 { panic(\"range function continued iteration after whole loop exit\") }; "
			replacement += "panic(\"range function continued iteration after loop body panic\") }; "
			replacement += binding + body + "; " + state + " = 1; return true }); "
			replacement += "if " + state + " == 2 { panic(\"range function recovered a loop body panic and did not resume panicking\") }; " + state + " = 3; "
			for n := 0; n < len(actions); n++ {
				replacement += "if " + action + " == " + functionValueDecimal(n+1) + " { " + actions[n] + " }; "
			}
			replacement += "}"
			edits := []functionValueEdit{functionValueTokenRangeEdit(program, at, close+1, replacement)}
			edits = appendFunctionValuePackageEdits(program, edits)
			originalLength := len(program.Text)
			if transient {
				renvo_runtime_ArenaDiscardLinkTokens(program.Tokens)
			}
			text, ok := applyFunctionValueEdits(program.Text, edits)
			if transient {
				arena.DiscardBytes(program.Text)
			}
			if !ok || !reparseFunctionValueProgram(program, text, edits, originalLength, -1) {
				return false
			}
			count++
			changed = true
			break
		}
		if !changed {
			return true
		}
	}
}

// Unlike the first top-level brace, the loop body follows any function or
// composite literal in the range expression. Parentheses also shield literals
// in calls, conversions, and indexed expressions.
func functionRangeHeader(program *unit.Program, at int) (int, int) {
	rangeTok := -1
	for i := at + 1; i < len(program.Tokens); i++ {
		if functionValueTokenEquals(program, i, "func") || functionValueTokenEquals(program, i, "struct") || functionValueTokenEquals(program, i, "interface") || functionValueTokenEquals(program, i, "map") || functionValueTokenEquals(program, i, "[") {
			end := functionValueTypeEnd(program, i)
			if end > i {
				i = end - 1
				if functionValueTokenEquals(program, end, "{") {
					i = functionValueFindMatchingBrace(program, end)
					if i < 0 {
						return -1, -1
					}
				}
				continue
			}
		}
		if functionValueTokenEquals(program, i, "(") || functionValueTokenEquals(program, i, "[") {
			left := functionValueTokenText(program, i)
			right := ")"
			if left == "[" {
				right = "]"
			}
			i = functionValueFindMatching(program, i, left, right)
			if i < 0 {
				return -1, -1
			}
			continue
		}
		if functionValueTokenEquals(program, i, "range") {
			rangeTok = i
		}
		if functionValueTokenEquals(program, i, "{") {
			return rangeTok, i
		}
	}
	return -1, -1
}

func functionRangeExpressionType(program *unit.Program, before int, start int, end int) string {
	if typ := ordinaryBuiltinExprType(program, before, start, end); typ != "" {
		return typ
	}
	return functionRangeExpressionFallbackType(program, before, start, end)
}

func functionRangeExpressionFallbackType(program *unit.Program, before int, start int, end int) string {
	for end-start >= 2 && functionValueTokenEquals(program, start, "(") && functionValueFindMatchingParen(program, start) == end-1 {
		start++
		end--
	}
	if functionValueTokenEquals(program, start, "func") {
		_, body, ok := parseFunctionValueSignature(program, start, "")
		if ok && functionValueTokenEquals(program, body, "{") && functionValueFindMatchingBrace(program, body) == end-1 {
			return functionValueTokensText(program, start, body)
		}
	}
	if end-start >= 3 && functionValueTokenEquals(program, end-2, ".") {
		owner := ordinaryBuiltinExprType(program, before, start, end-2)
		if typ := functionRangeInterfaceMethodType(program, owner, functionValueTokenText(program, end-1)); typ != "" {
			return typ
		}
		for n := 0; owner != "" && n < len(program.Funcs); n++ {
			fn := &program.Funcs[n]
			if fn.ReceiverStart < fn.ReceiverEnd && functionValueTokenEquals(program, end-1, functionValueTokenText(program, fn.NameTok)) && functionValueTypeEmbeds(program, owner, functionValueReceiverType(program, fn), 0) {
				_, sigEnd, ok := parseFunctionValueCallableSignature(program, fn.NameTok, "")
				if ok {
					return "func" + functionValueTokensText(program, fn.NameTok+1, sigEnd)
				}
			}
		}
	}
	if functionValueTokenEquals(program, end-1, ")") {
		open := functionValueFindMatchingBackward(program, end-1, "(", ")")
		if open > start {
			return functionValueCallableResultType(program, functionRangeExpressionType(program, before, start, open))
		}
	}
	if end-start == 1 && functionValueEnclosingLocalTypeDepthMode(program, before, functionValueTokenText(program, start), 0, false) == "" {
		for n := 0; n < len(program.Funcs); n++ {
			fn := program.Funcs[n]
			if fn.ReceiverStart == fn.ReceiverEnd && functionValueTokenEquals(program, start, functionValueTokenText(program, fn.NameTok)) {
				_, sigEnd, ok := parseFunctionValueCallableSignature(program, fn.NameTok, "")
				if ok {
					return "func" + functionValueTokensText(program, fn.NameTok+1, sigEnd)
				}
			}
		}
	}
	return ""
}

// Keep the interface's authored signature before function-value conversion.
// The existing semantic member resolver includes aliases and embedded method
// sets, with private method ownership retained by the linked declarations.
func functionRangeInterfaceMethodType(program *unit.Program, owner string, method string) string {
	underlying := ordinaryUnderlyingType(program, owner, 0)
	if !functionValueHasPrefix(underlying, "interface") {
		return ""
	}
	for tok := 0; tok < len(program.Tokens); tok++ {
		if !functionValueTokenEquals(program, tok, "interface") {
			continue
		}
		end := functionValueTypeEnd(program, tok)
		if functionValueTokensText(program, tok, end) != underlying {
			continue
		}
		members, ok := functionValueInterfaceMembers(program, tok, end, 0)
		if !ok {
			return ""
		}
		for _, member := range members {
			if member.name == method && !member.universeError {
				return "func" + functionValueTokensText(program, member.start+1, member.end)
			}
		}
		return ""
	}
	return ""
}

type functionRangeControl struct {
	start int
	end   int
	loop  bool
}

// Native nested loops and switches retain their own branches. A branch that
// leaves the callback first stops the iterator, then executes in its original
// enclosing scope. Nested literals have independent control/defer ownership.
func functionRangeBody(program *unit.Program, loop int, open int, close int, label string, state string, action string, stem string) (string, []string, string, bool) {
	controls := []functionRangeControl{}
	edits := []functionValueEdit{}
	actions := []string{}
	returnPrefix := ""
	returnTargets := ""
	returns := false
	labels := []string{}
	labelsParsed := false
	for at := open + 1; at < close; at++ {
		word := functionValueTokenText(program, at)
		if word == "func" {
			_, body, ok := parseFunctionValueSignature(program, at, "")
			if ok && functionValueTokenEquals(program, body, "{") {
				at = functionValueFindMatchingBrace(program, body)
				if at < 0 {
					return "", nil, "", false
				}
				continue
			}
		}
		if word == "for" || word == "switch" || word == "select" {
			_, body := functionRangeHeader(program, at)
			end := functionValueFindMatchingBrace(program, body)
			if body < 0 || end < 0 {
				return "", nil, "", false
			}
			controls = append(controls, functionRangeControl{start: at, end: end, loop: word == "for"})
		}
		if word == "defer" {
			// These require enclosing-function ownership. Never silently make
			// the callback their owner while that lowering is incomplete.
			return "", nil, "", false
		}
		if word == "return" {
			if !returns {
				var ok bool
				returnPrefix, returnTargets, ok = functionRangeReturnStorage(program, loop, stem)
				if !ok {
					return "", nil, "", false
				}
				returns = true
			}
			end := functionRangeStatementEnd(program, at, close)
			rhs := functionValueTokensText(program, at+1, end)
			replacement := "{ "
			if rhs != "" {
				replacement += returnTargets + " = " + rhs + "; "
			}
			actions = append(actions, "return "+returnTargets)
			replacement += state + " = 0; " + action + " = " + functionValueDecimal(len(actions)) + "; return false }"
			edits = append(edits, functionValueTokenRangeEdit(program, at, end, replacement))
			at = end - 1
			continue
		}
		if word != "break" && word != "continue" && word != "goto" {
			continue
		}
		end := at + 1
		target := ""
		if end < close && program.Tokens[end].KindLine&255 == unit.TokenIdent && program.Tokens[end].KindLine>>8 == program.Tokens[at].KindLine>>8 {
			target = functionValueTokenText(program, end)
			end++
		}
		local := false
		if target != "" && target != label {
			if !labelsParsed {
				var ok bool
				labels, ok = functionRangeLabels(program, open, close)
				if !ok {
					return "", nil, "", false
				}
				labelsParsed = true
			}
			for n := 0; n < len(labels); n++ {
				if labels[n] == target {
					local = true
					break
				}
			}
		} else if target == "" {
			for n := 0; n < len(controls); n++ {
				if controls[n].start < at && controls[n].end > at && (word != "continue" || controls[n].loop) {
					local = true
					break
				}
			}
		}
		if local {
			continue
		}
		replacement := "{ " + state + " = 0; "
		if (target != "" && target != label) || word == "goto" {
			actions = append(actions, functionValueTokensText(program, at, end))
			replacement += action + " = " + functionValueDecimal(len(actions)) + "; "
		} else if word == "continue" {
			replacement = "{ " + state + " = 1; return true }"
		}
		if word != "continue" || (target != "" && target != label) {
			replacement += "return false }"
		}
		edits = append(edits, functionValueTokenRangeEdit(program, at, end, replacement))
		at = end - 1
	}
	startByte := program.Tokens[open].Start + program.Tokens[open].Size
	endByte := program.Tokens[close].Start
	for n := 0; n < len(edits); n++ {
		edits[n].start -= startByte
		edits[n].end -= startByte
	}
	body, ok := applyFunctionValueEdits(program.Text[startByte:endByte], edits)
	return string(body), actions, returnPrefix, ok
}

// Labels live in statement blocks. A keyed literal's field and a label inside
// a nested function can share the spelling of an outward branch's target.
func functionRangeLabels(program *unit.Program, open int, close int) ([]string, bool) {
	source := []byte("package main\nfunc labels() {\n" + functionValueTokensText(program, open+1, close) + "\n}\n")
	file := syntax.ParseFile(source)
	if !file.Ok || len(file.Funcs) != 1 {
		return nil, false
	}
	body := syntax.ParseFuncBodyStatements(file, file.Funcs[0])
	if !body.Ok {
		return nil, false
	}
	labels := []string{}
	for _, stmt := range body.Stmts {
		if stmt.Kind == syntax.StmtLabel {
			token := file.Tokens[stmt.StartTok]
			labels = append(labels, string(file.Src[token.Start:token.End]))
		}
	}
	return labels, true
}

// Named results must be assigned at the return inside the body, before the
// iterator's cleanup runs. Capturing their addresses outside the loop also
// preserves this assignment when the body shadows a result name.
func functionRangeReturnStorage(program *unit.Program, loop int, stem string) (string, string, bool) {
	fn, ok := functionValueLexicalFunction(program, loop)
	if !ok {
		return "", "", false
	}
	sig, _, ok := parseFunctionValueCallableSignature(program, fn.NameTok, "")
	if !ok {
		return "", "", false
	}
	names := []string{}
	paramEnd := functionValueFindMatchingParen(program, fn.NameTok+1)
	resultStart := paramEnd + 1
	if functionValueTokenEquals(program, resultStart, "(") {
		resultEnd := functionValueFindMatchingParen(program, resultStart)
		starts, ends := functionValueCommaParts(program, resultStart+1, resultEnd)
		named := false
		for n := 0; n < len(starts); n++ {
			if starts[n]+1 < ends[n] && program.Tokens[starts[n]].KindLine&255 == unit.TokenIdent && functionValueTypeEnd(program, starts[n]+1) == ends[n] {
				named = true
			}
		}
		if named {
			_, names, _, ok = normalizedFunctionValueParams(program, resultStart+1, resultEnd)
			if !ok {
				return "", "", false
			}
		}
	}
	prefix := ""
	targets := ""
	for n := 0; n < len(sig.resultTypes); n++ {
		storage := integerRangeName(program, stem+"_result_"+functionValueDecimal(n))
		if n < len(names) && names[n] != "_" {
			prefix += storage + " := &" + names[n] + "; "
			storage = "*" + storage
		} else {
			prefix += "var " + storage + " " + sig.resultTypes[n] + "; "
		}
		if n > 0 {
			targets += ", "
		}
		targets += storage
	}
	return prefix, targets, true
}

func functionRangeStatementEnd(program *unit.Program, start int, limit int) int {
	previous := start
	for at := start + 1; at < limit; at++ {
		if functionValueTokenEquals(program, at, ";") || functionValueTokenEquals(program, at, "}") {
			return at
		}
		if program.Tokens[at].KindLine>>8 > program.Tokens[previous].KindLine>>8 {
			kind := program.Tokens[previous].KindLine & 255
			if previous == start || kind == unit.TokenIdent || kind == unit.TokenNumber || kind == unit.TokenFloat || kind == unit.TokenString || kind == unit.TokenChar || functionValueTokenEquals(program, previous, ")") || functionValueTokenEquals(program, previous, "]") || functionValueTokenEquals(program, previous, "}") {
				return at
			}
		}
		left := functionValueTokenText(program, at)
		right := ""
		if left == "(" {
			right = ")"
		} else if left == "[" {
			right = "]"
		} else if left == "{" {
			right = "}"
		}
		if right != "" {
			at = functionValueFindMatching(program, at, left, right)
			if at < 0 || at >= limit {
				return limit
			}
		}
		previous = at
	}
	return limit
}
