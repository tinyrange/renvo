package link

import "renvo.dev/internal/arena"
import "renvo.dev/internal/unit"

func errorsAsNamesCore(programs []unit.Program, aliases []string, offsets []int) []string {
	var names []string
	for i := 0; i < len(programs); i++ {
		if programs[i].ImportPath != "errors" && programs[i].ImportPath != "renvo.dev/std/errors" {
			continue
		}
		targetName := ""
		asName := ""
		for j := 0; j < len(programs[i].Symbols); j++ {
			name := programs[i].Symbols[j].Name
			if name != "asTarget" && name != "As" {
				continue
			}
			alias := corePackageSymbolAlias(aliases, offsets, i, j)
			if alias == "" {
				alias = name
			}
			if name == "asTarget" {
				targetName = alias
			} else {
				asName = alias
			}
		}
		if targetName != "" && asName != "" {
			names = append(names, cloneCoreLinkString(targetName), cloneCoreLinkString(asName))
		}
	}
	return names
}

func lowerErrorsAsCore(program *unit.Program, names []string, transient bool) bool {
	if !errorsAsReferenced(program, names) {
		return true
	}
	receivers := []string{"error"}
	for _, fn := range program.Funcs {
		if fn.ReceiverStart < fn.ReceiverEnd && functionValueTokenText(program, fn.NameTok) == "Error" {
			receivers = append(receivers, functionValueBareType(functionValueReceiverType(program, fn)))
		}
	}
	for _, decl := range program.Decls {
		if decl.Kind == unit.TokenType {
			typ := string(program.Text[decl.NameStart:decl.NameEnd])
			if interfaceExpressionMethod(program, typ, "Error", 0) >= 0 {
				receivers = append(receivers, typ)
			}
		}
	}
	body := "{"
	body += errorsAsTargetCase("error", true)
	body += errorsAsTargetCase("any", true)
	for _, decl := range program.Decls {
		if decl.Kind != unit.TokenType {
			continue
		}
		typ := string(program.Text[decl.NameStart:decl.NameEnd])
		underlying := functionValueCompactTypeText(ordinaryUnderlyingType(program, typ, 0))
		isInterface := underlying == "any" || underlying == "error" || functionValueHasPrefix(underlying, "interface{")
		if isInterface {
			body += errorsAsTargetCase(typ, true)
			continue
		}
		candidate := false
		for _, receiver := range receivers {
			if functionValueTypeEmbeds(program, typ, receiver, 0) {
				candidate = true
				break
			}
		}
		if candidate {
			body += errorsAsTargetCase(typ, false)
			body += errorsAsTargetCase("*"+typ, false)
		}
	}
	body += "return false,false }"
	var edits []functionValueEdit
	for i := 0; i < len(names); i += 2 {
		index := findCoreFuncByName(program, names[i])
		if index < 0 {
			return false
		}
		fn := program.Funcs[index]
		end := functionValueFindMatchingBrace(program, fn.BodyStart)
		if end < 0 {
			return false
		}
		edits = append(edits, functionValueTokenRangeEdit(program, fn.BodyStart, end+1, body))
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
	return reparseFunctionValueProgram(program, text, edits, originalLength, len(text))
}

// Most users of errors need only New or Is. Avoid collecting every linked type
// when As is never referenced outside its own recursive implementation.
func errorsAsReferenced(program *unit.Program, names []string) bool {
	// An errors package root can export As from a library object.
	if len(names) != 0 && (program.ImportPath == "errors" || program.ImportPath == "renvo.dev/std/errors") {
		return true
	}
	for i := 1; i < len(names); i += 2 {
		index := findCoreFuncByName(program, names[i])
		if index < 0 {
			continue
		}
		fn := program.Funcs[index]
		for tok := 0; tok < len(program.Tokens); tok++ {
			if tok == fn.StartTok {
				tok = fn.EndTok - 1
				continue
			}
			if program.Tokens[tok].KindLine&255 == unit.TokenIdent && functionValueTokenEquals(program, tok, names[i]) {
				return true
			}
		}
	}
	return false
}

func errorsAsTargetCase(typ string, isInterface bool) string {
	text := "if pointer,ok := target.(*" + typ + "); ok { if pointer==nil { return false,false }; "
	if !isInterface {
		text += "var zero " + typ + "; if _,valid:=any(zero).(error); !valid { return false,false }; "
	}
	if typ == "any" {
		text += "*pointer=err; return true,true };\n"
	} else {
		text += "if value,matched:=any(err).(" + typ + "); matched { *pointer=value; return true,true }; return true,false };\n"
	}
	return text
}
