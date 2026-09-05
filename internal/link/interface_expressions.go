package link

import "renvo.dev/internal/arena"
import "renvo.dev/internal/unit"

func lowerInterfaceMethodExpressions(program *unit.Program, transient bool) bool {
	var edits []functionValueEdit
	generated := ""
	count := 0
	for i := 0; i+2 < len(program.Tokens); i++ {
		if program.Tokens[i].KindLine&255 != unit.TokenIdent || !functionValueTokenEquals(program, i+1, ".") || program.Tokens[i+2].KindLine&255 != unit.TokenIdent {
			continue
		}
		name := functionValueTokenText(program, i)
		method := interfaceExpressionMethod(program, name, functionValueTokenText(program, i+2), 0)
		if method < 0 {
			continue
		}
		if functionValueLexicalLocalType(program, i, name) != "" {
			continue
		}
		sig, _, ok := parseFunctionValueCallableSignature(program, method, "")
		if !ok {
			return false
		}
		wrapper := ordinaryBuiltinGeneratedName(program, "__renvo_interface_expression_"+functionValueDecimal(count))
		params := "__renvo_receiver " + name
		args := ""
		for j := 0; j < len(sig.paramTypes); j++ {
			arg := "__renvo_arg_" + functionValueDecimal(j)
			params += ", " + arg + " " + sig.paramTypes[j]
			if j > 0 {
				args += ", "
			}
			args += arg
			if functionValueHasPrefix(sig.paramTypes[j], "...") {
				args += "..."
			}
		}
		body := "__renvo_receiver." + functionValueTokenText(program, i+2) + "(" + args + ")"
		if sig.result != "" {
			body = "return " + body
		}
		generated += "func " + wrapper + "(" + params + ") " + sig.result + " { " + body + " }\n"
		edits = append(edits, functionValueTokenRangeEdit(program, i, i+3, wrapper))
		count++
		i += 2
	}
	if len(edits) == 0 {
		return true
	}
	edits = appendFunctionValuePackageEdits(program, edits)
	originalLength := len(program.Text)
	if transient {
		renvo_runtime_ArenaDiscardLinkTokens(program.Tokens)
	}
	text, ok := applyFunctionValueEdits(program.Text, edits)
	if transient {
		arena.DiscardBytes(program.Text)
	}
	if !ok {
		return false
	}
	text = append(text, '\n')
	generatedStart := len(text)
	text = appendFunctionValueString(text, generated)
	return reparseFunctionValueProgram(program, text, edits, originalLength, generatedStart)
}

func interfaceExpressionMethod(program *unit.Program, typ string, method string, depth int) int {
	if depth > 16 {
		return -1
	}
	for _, decl := range program.Decls {
		if decl.Kind != unit.TokenType || !interfaceExpressionName(program.Text, decl.NameStart, decl.NameEnd, typ) {
			continue
		}
		start := functionValueTokenAtSpan(program, decl.NameStart, decl.NameEnd) + 1
		if functionValueTokenEquals(program, start, "=") {
			start++
		}
		if !functionValueTokenEquals(program, start, "interface") {
			if program.Tokens[start].KindLine&255 == unit.TokenIdent {
				return interfaceExpressionMethod(program, functionValueTokenText(program, start), method, depth+1)
			}
			return -1
		}
		close := functionValueFindMatchingBrace(program, start+1)
		for tok := start + 2; tok < close; tok++ {
			if functionValueTokenEquals(program, tok, method) && functionValueTokenEquals(program, tok+1, "(") {
				return tok
			}
			if functionValueTokenEquals(program, tok+1, "(") {
				_, end, ok := parseFunctionValueCallableSignature(program, tok, "")
				if ok {
					tok = end - 1
				}
			} else if program.Tokens[tok].KindLine&255 == unit.TokenIdent {
				if found := interfaceExpressionMethod(program, functionValueTokenText(program, tok), method, depth+1); found >= 0 {
					return found
				}
			}
		}
		return -1
	}
	return -1
}

func interfaceExpressionName(src []byte, start int, end int, name string) bool {
	if end-start != len(name) {
		return false
	}
	for i := 0; i < len(name); i++ {
		if src[start+i] != name[i] {
			return false
		}
	}
	return true
}
