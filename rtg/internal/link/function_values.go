package link

import (
	"j5.nz/rtg/rtg/internal/syntax"
	"j5.nz/rtg/rtg/internal/unit"
)

// Function values are lowered after package linking, when every possible
// implementation is visible. The backend subset receives ordinary structs and
// direct calls: a zero tag is nil, and each non-zero tag selects a generated
// direct-call arm. Receiver storage remains statically typed, so this needs no
// unsafe conversion, generic runtime, or target-specific callback VM.

type functionValueSignature struct {
	name        string
	params      string
	paramNames  []string
	result      string
	fields      []string
	fieldTypes  []string
	impls       []functionValueImpl
	declFuncTok int
	declEndTok  int
}

type functionValueImpl struct {
	receiverType  string
	receiverField string
	method        string
	function      string
}

type functionValueClosure struct {
	envName  string
	funcName string
	fields   []string
	types    []string
	params   string
	result   string
	body     string
}

type functionValueField struct {
	owner string
	name  string
	sig   int
}

type functionValueEdit struct {
	start int
	end   int
	text  string
}

func lowerFunctionValuesCore(program *unit.Program) bool {
	if !functionValueProgramNeedsLowering(program) {
		return true
	}
	signatures, fields, edits, ok := discoverFunctionValueTypes(program)
	if !ok {
		return false
	}
	if len(signatures) == 0 {
		return true
	}
	edits = appendFunctionValuePackageEdits(program, edits)
	for i := 0; i < len(program.Tokens); i++ {
		text := functionValueTokenText(program, i)
		if text == "=" {
			edits, ok = lowerFunctionValueAssignment(program, i, signatures, fields, edits)
			if !ok {
				return false
			}
		}
	}
	var closures []functionValueClosure
	signatures, closures, edits, ok = lowerFunctionValueLiterals(program, signatures, fields, closures, edits)
	if !ok {
		return false
	}
	for i := 0; i < len(program.Tokens); i++ {
		text := functionValueTokenText(program, i)
		if text == "==" || text == "!=" {
			edits = lowerFunctionValueNilComparison(program, i, fields, edits)
		}
	}
	for i := 0; i < len(program.Tokens); i++ {
		if functionValueTokenEquals(program, i, "(") {
			edits = lowerFunctionValueCall(program, i, signatures, fields, edits)
		}
	}
	for i := 0; i < len(signatures); i++ {
		sig := &signatures[i]
		if sig.declFuncTok >= 0 {
			replacement := functionValueStructText(*sig)
			edits = append(edits, functionValueTokenRangeEdit(program, sig.declFuncTok, sig.declEndTok, replacement))
		}
	}
	generated := functionValueGeneratedText(signatures, closures)
	text, ok := applyFunctionValueEdits(program.Text, edits)
	if !ok {
		return false
	}
	if len(text) > 0 && text[len(text)-1] != '\n' {
		text = append(text, '\n')
	}
	text = appendFunctionValueString(text, generated)
	return reparseFunctionValueProgram(program, text)
}

func functionValueProgramNeedsLowering(program *unit.Program) bool {
	for i := 0; i+1 < len(program.Tokens); i++ {
		if functionValueTokenEquals(program, i, "func") && functionValueTokenEquals(program, i+1, "(") && !functionValueIsDeclaredFunction(program, i) {
			return true
		}
	}
	return false
}

func discoverFunctionValueTypes(program *unit.Program) ([]functionValueSignature, []functionValueField, []functionValueEdit, bool) {
	var signatures []functionValueSignature
	var fields []functionValueField
	var edits []functionValueEdit
	for i := 0; i < len(program.Decls); i++ {
		decl := program.Decls[i]
		if decl.Kind != unit.TokenType {
			continue
		}
		nameTok := functionValueTokenAtSpan(program, decl.NameStart, decl.NameEnd)
		if nameTok < 0 {
			return signatures, fields, edits, false
		}
		owner := functionValueTokenText(program, nameTok)
		start := nameTok + 1
		if functionValueTokenEquals(program, start, "=") {
			start++
		}
		if functionValueTokenEquals(program, start, "func") {
			sig, end, ok := parseFunctionValueSignature(program, start, functionValueTokenText(program, nameTok))
			if !ok {
				return signatures, fields, edits, false
			}
			sig.declFuncTok = start
			sig.declEndTok = end
			signatures = append(signatures, sig)
			continue
		}
		if !functionValueTokenEquals(program, start, "struct") || !functionValueTokenEquals(program, start+1, "{") {
			continue
		}
		close := functionValueFindMatchingBrace(program, start+1)
		if close < 0 {
			return signatures, fields, edits, false
		}
		j := start + 2
		for j < close {
			if program.Tokens[j].Kind != unit.TokenIdent {
				j++
				continue
			}
			fieldName := functionValueTokenText(program, j)
			typeTok := j + 1
			sigIndex := functionValueSignatureByName(signatures, functionValueTokenText(program, typeTok))
			if sigIndex >= 0 {
				fields = append(fields, functionValueField{owner: owner, name: fieldName, sig: sigIndex})
				j = typeTok + 1
				continue
			}
			if functionValueTokenEquals(program, typeTok, "func") {
				name := "__rtg_function_" + functionValueDecimal(len(signatures))
				sig, end, ok := parseFunctionValueSignature(program, typeTok, name)
				if !ok {
					return signatures, fields, edits, false
				}
				sig.declFuncTok = -1
				sig.declEndTok = -1
				sigIndex = len(signatures)
				signatures = append(signatures, sig)
				fields = append(fields, functionValueField{owner: owner, name: fieldName, sig: sigIndex})
				edits = append(edits, functionValueTokenRangeEdit(program, typeTok, end, name))
				j = end
				continue
			}
			j++
		}
	}
	// A named function type can be declared after the struct that uses it.
	for i := 0; i < len(program.Decls); i++ {
		decl := program.Decls[i]
		if decl.Kind != unit.TokenType {
			continue
		}
		nameTok := functionValueTokenAtSpan(program, decl.NameStart, decl.NameEnd)
		owner := functionValueTokenText(program, nameTok)
		start := nameTok + 1
		if !functionValueTokenEquals(program, start, "struct") || !functionValueTokenEquals(program, start+1, "{") {
			continue
		}
		close := functionValueFindMatchingBrace(program, start+1)
		for j := start + 2; j+1 < close; j++ {
			if program.Tokens[j].Kind != unit.TokenIdent {
				continue
			}
			sigIndex := functionValueSignatureByName(signatures, functionValueTokenText(program, j+1))
			if sigIndex >= 0 && functionValueFieldByOwnerAndName(fields, owner, functionValueTokenText(program, j)) < 0 {
				fields = append(fields, functionValueField{owner: owner, name: functionValueTokenText(program, j), sig: sigIndex})
			}
		}
	}
	return signatures, fields, edits, true
}

func parseFunctionValueSignature(program *unit.Program, funcTok int, name string) (functionValueSignature, int, bool) {
	var sig functionValueSignature
	sig.name = name
	sig.declFuncTok = funcTok
	if !functionValueTokenEquals(program, funcTok, "func") || !functionValueTokenEquals(program, funcTok+1, "(") {
		return sig, funcTok, false
	}
	close := functionValueFindMatchingParen(program, funcTok+1)
	if close < 0 {
		return sig, funcTok, false
	}
	params, names, ok := normalizedFunctionValueParams(program, funcTok+2, close)
	if !ok {
		return sig, funcTok, false
	}
	sig.params = params
	sig.paramNames = names
	end := close + 1
	if functionValueTokenEquals(program, end, "(") {
		resultClose := functionValueFindMatchingParen(program, end)
		if resultClose < 0 {
			return sig, funcTok, false
		}
		sig.result = functionValueTokensText(program, end, resultClose+1)
		end = resultClose + 1
	} else if functionValueTokenCanStartType(program, end) && program.Tokens[end].Line == program.Tokens[close].Line {
		sig.result = functionValueTokenText(program, end)
		if sig.result == "*" && end+1 < len(program.Tokens) {
			sig.result = sig.result + functionValueTokenText(program, end+1)
			end += 2
		} else {
			end++
		}
	}
	return sig, end, true
}

func normalizedFunctionValueParams(program *unit.Program, start int, end int) (string, []string, bool) {
	var partStarts []int
	var partEnds []int
	partStart := start
	depth := 0
	for i := start; i <= end; i++ {
		text := functionValueTokenText(program, i)
		if i < end {
			if text == "(" || text == "[" || text == "{" {
				depth++
			} else if text == ")" || text == "]" || text == "}" {
				depth--
			}
		}
		if i != end && !(depth == 0 && text == ",") {
			continue
		}
		if partStart < i {
			partStarts = append(partStarts, partStart)
			partEnds = append(partEnds, i)
		}
		partStart = i + 1
	}
	var out []byte
	var names []string
	for i := 0; i < len(partStarts); i++ {
		if len(out) > 0 {
			out = appendFunctionValueString(out, ", ")
		}
		partLen := partEnds[i] - partStarts[i]
		name := "arg" + functionValueDecimal(i)
		typ := functionValueTokensText(program, partStarts[i], partEnds[i])
		if partLen >= 2 && program.Tokens[partStarts[i]].Kind == unit.TokenIdent {
			name = functionValueTokenText(program, partStarts[i])
			typ = functionValueTokensText(program, partStarts[i]+1, partEnds[i])
		} else if partLen == 1 && i+1 < len(partStarts) && partEnds[i+1]-partStarts[i+1] >= 2 {
			name = functionValueTokenText(program, partStarts[i])
			typ = functionValueTokensText(program, partStarts[i+1]+1, partEnds[i+1])
		}
		out = appendFunctionValueString(out, name+" "+typ)
		names = append(names, name)
	}
	return string(out), names, depth == 0
}

func lowerFunctionValueAssignment(program *unit.Program, op int, signatures []functionValueSignature, fields []functionValueField, edits []functionValueEdit) ([]functionValueEdit, bool) {
	fieldTok := functionValueSelectorFieldBefore(program, op)
	fieldIndex := functionValueFieldForSelector(program, fieldTok, fields)
	if fieldIndex < 0 || op+1 >= len(program.Tokens) {
		return edits, true
	}
	sigIndex := fields[fieldIndex].sig
	rhs := op + 1
	if functionValueTokenEquals(program, rhs, "nil") {
		edits = append(edits, functionValueTokenEdit(program, rhs, signatures[sigIndex].name+"{}"))
		return edits, true
	}
	if rhs+2 >= len(program.Tokens) || !functionValueTokenEquals(program, rhs+1, ".") || program.Tokens[rhs].Kind != unit.TokenIdent || program.Tokens[rhs+2].Kind != unit.TokenIdent {
		return edits, true
	}
	method := functionValueTokenText(program, rhs+2)
	receiverType := functionValueMethodReceiverTypeForBase(program, op, functionValueTokenText(program, rhs), method)
	if receiverType == "" {
		return edits, true
	}
	implIndex := functionValueImplIndex(signatures[sigIndex], receiverType, method, "")
	if implIndex < 0 {
		implIndex = len(signatures[sigIndex].impls)
		fieldName := "receiver" + functionValueDecimal(implIndex)
		signatures[sigIndex].impls = append(signatures[sigIndex].impls, functionValueImpl{receiverType: receiverType, receiverField: fieldName, method: method})
		signatures[sigIndex].fields = append(signatures[sigIndex].fields, fieldName)
		signatures[sigIndex].fieldTypes = append(signatures[sigIndex].fieldTypes, receiverType)
	}
	impl := signatures[sigIndex].impls[implIndex]
	replacement := signatures[sigIndex].name + "{kind: " + functionValueDecimal(implIndex+1) + ", " + impl.receiverField + ": " + functionValueTokenText(program, rhs) + "}"
	edits = append(edits, functionValueTokenRangeEdit(program, rhs, rhs+3, replacement))
	return edits, true
}

func lowerFunctionValueNilComparison(program *unit.Program, op int, fields []functionValueField, edits []functionValueEdit) []functionValueEdit {
	if op+1 >= len(program.Tokens) || !functionValueTokenEquals(program, op+1, "nil") {
		return edits
	}
	fieldTok := functionValueSelectorFieldBefore(program, op)
	if functionValueFieldForSelector(program, fieldTok, fields) < 0 {
		return edits
	}
	end := program.Tokens[fieldTok].Start + program.Tokens[fieldTok].Size
	edits = append(edits, functionValueEdit{start: end, end: end, text: ".kind"})
	edits = append(edits, functionValueTokenEdit(program, op+1, "0"))
	return edits
}

func lowerFunctionValueCall(program *unit.Program, open int, signatures []functionValueSignature, fields []functionValueField, edits []functionValueEdit) []functionValueEdit {
	fieldTok := open - 1
	if fieldTok < 2 || !functionValueTokenEquals(program, fieldTok-1, ".") {
		return edits
	}
	fieldIndex := functionValueFieldForSelector(program, fieldTok, fields)
	if fieldIndex < 0 {
		return edits
	}
	calleeStart := functionValueSelectorStart(program, fieldTok)
	if calleeStart < 0 {
		return edits
	}
	close := functionValueFindMatchingParen(program, open)
	if close < 0 {
		return edits
	}
	start := program.Tokens[calleeStart].Start
	edits = append(edits, functionValueEdit{start: start, end: start, text: "__rtg_call_" + functionValueDecimal(fields[fieldIndex].sig) + "(&"})
	if open+1 == close {
		edits = append(edits, functionValueTokenEdit(program, open, ""))
	} else {
		edits = append(edits, functionValueTokenEdit(program, open, ", "))
	}
	return edits
}

func functionValueStructText(sig functionValueSignature) string {
	out := "struct { kind int"
	for i := 0; i < len(sig.fields); i++ {
		out = out + "; " + sig.fields[i] + " " + sig.fieldTypes[i]
	}
	out = out + " }"
	return out
}

func functionValueGeneratedText(signatures []functionValueSignature, closures []functionValueClosure) string {
	out := ""
	for i := 0; i < len(closures); i++ {
		closure := closures[i]
		out = out + "type " + closure.envName + " struct {"
		for j := 0; j < len(closure.fields); j++ {
			out = out + " " + closure.fields[j] + " " + closure.types[j] + ";"
		}
		out = out + " }\n"
		out = out + "func " + closure.funcName + "(env *" + closure.envName
		if closure.params != "" {
			out = out + ", " + closure.params
		}
		out = out + ")"
		if closure.result != "" {
			out = out + " " + closure.result
		}
		out = out + " {" + closure.body + "}\n"
	}
	for i := 0; i < len(signatures); i++ {
		sig := signatures[i]
		if sig.declFuncTok < 0 {
			out = out + "type " + sig.name + " " + functionValueStructText(sig) + "\n"
		}
		out = out + "func __rtg_call_" + functionValueDecimal(i) + "(fn *" + sig.name
		if sig.params != "" {
			out = out + ", " + sig.params
		}
		out = out + ")"
		if sig.result != "" {
			out = out + " " + sig.result
		}
		out = out + " {\n"
		args := functionValueJoin(sig.paramNames, ", ")
		for j := 0; j < len(sig.impls); j++ {
			impl := sig.impls[j]
			out = out + "if fn.kind == " + functionValueDecimal(j+1) + " { "
			if sig.result != "" {
				out = out + "return "
			}
			if impl.method != "" {
				out = out + "fn." + impl.receiverField + "." + impl.method + "(" + args + ")"
			} else if impl.receiverField != "" {
				callArgs := "fn." + impl.receiverField
				if args != "" {
					callArgs = callArgs + ", " + args
				}
				out = out + impl.function + "(" + callArgs + ")"
			} else {
				out = out + impl.function + "(" + args + ")"
			}
			if sig.result == "" {
				out = out + "; return"
			}
			out = out + " }\n"
		}
		if sig.result != "" {
			out = out + "return " + functionValueZero(sig.result) + "\n"
		} else {
			out = out + "return\n"
		}
		out = out + "}\n"
	}
	return out
}

func lowerFunctionValueLiterals(program *unit.Program, signatures []functionValueSignature, fields []functionValueField, closures []functionValueClosure, edits []functionValueEdit) ([]functionValueSignature, []functionValueClosure, []functionValueEdit, bool) {
	for funcTok := 0; funcTok+1 < len(program.Tokens); funcTok++ {
		if !functionValueTokenEquals(program, funcTok, "func") || !functionValueTokenEquals(program, funcTok+1, "(") || functionValueIsDeclaredFunction(program, funcTok) {
			continue
		}
		fieldTok := funcTok - 2
		if fieldTok < 0 || !functionValueTokenEquals(program, funcTok-1, ":") {
			continue
		}
		fieldIndex := functionValueFieldByName(fields, functionValueTokenText(program, fieldTok))
		if fieldIndex < 0 {
			continue
		}
		sigIndex := fields[fieldIndex].sig
		literalSig, signatureEnd, ok := parseFunctionValueSignature(program, funcTok, signatures[sigIndex].name)
		if !ok || !functionValueTokenEquals(program, signatureEnd, "{") {
			return signatures, closures, edits, false
		}
		bodyClose := functionValueFindMatchingBrace(program, signatureEnd)
		if bodyClose < 0 {
			return signatures, closures, edits, false
		}
		captures, captureTypes := functionValueCaptures(program, funcTok, signatureEnd, bodyClose, literalSig.paramNames)
		if len(captures) == 0 {
			return signatures, closures, edits, false
		}
		closureIndex := len(closures)
		envName := "__rtg_closure_env_" + functionValueDecimal(closureIndex)
		funcName := "__rtg_closure_" + functionValueDecimal(closureIndex)
		body := functionValueClosureBody(program, signatureEnd, bodyClose, captures)
		closures = append(closures, functionValueClosure{envName: envName, funcName: funcName, fields: captures, types: captureTypes, params: literalSig.params, result: literalSig.result, body: body})
		implIndex := len(signatures[sigIndex].impls)
		closureField := "closure" + functionValueDecimal(implIndex)
		signatures[sigIndex].impls = append(signatures[sigIndex].impls, functionValueImpl{receiverType: "*" + envName, receiverField: closureField, function: funcName})
		signatures[sigIndex].fields = append(signatures[sigIndex].fields, closureField)
		signatures[sigIndex].fieldTypes = append(signatures[sigIndex].fieldTypes, "*"+envName)
		init := "&" + envName + "{"
		for i := 0; i < len(captures); i++ {
			if i > 0 {
				init = init + ", "
			}
			init = init + captures[i] + ": " + captures[i]
		}
		init = init + "}"
		replacement := signatures[sigIndex].name + "{kind: " + functionValueDecimal(implIndex+1) + ", " + closureField + ": " + init + "}"
		edits = append(edits, functionValueTokenRangeEdit(program, funcTok, bodyClose+1, replacement))
		funcTok = bodyClose
	}
	return signatures, closures, edits, true
}

func functionValueIsDeclaredFunction(program *unit.Program, funcTok int) bool {
	for i := 0; i < len(program.Funcs); i++ {
		if program.Funcs[i].StartTok == funcTok {
			return true
		}
	}
	return false
}

func functionValueCaptures(program *unit.Program, literalStart int, bodyOpen int, bodyClose int, params []string) ([]string, []string) {
	var names []string
	var types []string
	for i := bodyOpen + 1; i < bodyClose; i++ {
		if program.Tokens[i].Kind != unit.TokenIdent {
			continue
		}
		name := functionValueTokenText(program, i)
		if name == "return" || name == "true" || name == "false" || name == "nil" || functionValueNameInList(params, name) || functionValueNameInList(names, name) {
			continue
		}
		typ := functionValueEnclosingLocalType(program, literalStart, name)
		if typ == "" {
			continue
		}
		names = append(names, name)
		types = append(types, typ)
	}
	return names, types
}

func functionValueEnclosingLocalType(program *unit.Program, before int, name string) string {
	fnIndex := -1
	for i := 0; i < len(program.Funcs); i++ {
		fn := program.Funcs[i]
		if fn.BodyStart < before && before < fn.BodyEnd {
			fnIndex = i
			break
		}
	}
	if fnIndex < 0 {
		return ""
	}
	fn := program.Funcs[fnIndex]
	if fn.ReceiverStart < fn.ReceiverEnd {
		start := fn.ReceiverStart
		end := fn.ReceiverEnd
		if functionValueTokenEquals(program, start, "(") {
			start++
		}
		if functionValueTokenEquals(program, end-1, ")") {
			end--
		}
		if end-start >= 2 && functionValueTokenEquals(program, start, name) {
			return functionValueTokensText(program, start+1, end)
		}
	}
	for i := fn.BodyStart + 1; i+2 < before; i++ {
		if !functionValueTokenEquals(program, i, name) {
			continue
		}
		if functionValueTokenEquals(program, i+1, ":=") {
			rhs := i + 2
			if functionValueTokenEquals(program, rhs, "&") && rhs+1 < before && program.Tokens[rhs+1].Kind == unit.TokenIdent {
				return "*" + functionValueTokenText(program, rhs+1)
			}
			if program.Tokens[rhs].Kind == unit.TokenNumber {
				return "int"
			}
			if program.Tokens[rhs].Kind == unit.TokenString {
				return "string"
			}
			if program.Tokens[rhs].Kind == unit.TokenIdent {
				callName := ""
				if functionValueTokenEquals(program, rhs+1, "(") {
					callName = functionValueTokenText(program, rhs)
				} else if functionValueTokenEquals(program, rhs+1, ".") && rhs+3 < before && functionValueTokenEquals(program, rhs+3, "(") {
					callName = functionValueTokenText(program, rhs+2)
				}
				if callName != "" {
					if typ := functionValueDeclaredFunctionResultType(program, callName); typ != "" {
						return typ
					}
				}
				if typ := functionValueFunctionParamType(program, fn, functionValueTokenText(program, rhs)); typ != "" {
					return typ
				}
			}
		}
		if i > 0 && functionValueTokenEquals(program, i-1, "var") && i+1 < before {
			return functionValueTokenText(program, i+1)
		}
	}
	return functionValueFunctionParamType(program, fn, name)
}

func functionValueDeclaredFunctionResultType(program *unit.Program, name string) string {
	for i := 0; i < len(program.Funcs); i++ {
		fn := program.Funcs[i]
		if fn.ReceiverStart < fn.ReceiverEnd || functionValueTokenText(program, fn.NameTok) != name {
			continue
		}
		open := fn.NameTok + 1
		if !functionValueTokenEquals(program, open, "(") {
			continue
		}
		close := functionValueFindMatchingParen(program, open)
		resultStart := close + 1
		if resultStart >= fn.BodyStart {
			return ""
		}
		if functionValueTokenEquals(program, resultStart, "(") {
			resultEnd := functionValueFindMatchingParen(program, resultStart)
			if resultEnd <= resultStart {
				return ""
			}
			return functionValueTokensText(program, resultStart+1, resultEnd)
		}
		return functionValueTokensText(program, resultStart, fn.BodyStart)
	}
	return ""
}

func functionValueFunctionParamType(program *unit.Program, fn unit.Func, name string) string {
	open := fn.NameTok + 1
	if !functionValueTokenEquals(program, open, "(") {
		return ""
	}
	close := functionValueFindMatchingParen(program, open)
	for i := open + 1; i+1 < close; i++ {
		if functionValueTokenEquals(program, i, name) {
			end := i + 2
			for end < close && !functionValueTokenEquals(program, end, ",") {
				end++
			}
			return functionValueTokensText(program, i+1, end)
		}
	}
	return ""
}

func functionValueClosureBody(program *unit.Program, bodyOpen int, bodyClose int, captures []string) string {
	if bodyOpen+1 >= bodyClose {
		return ""
	}
	start := program.Tokens[bodyOpen].Start + program.Tokens[bodyOpen].Size
	end := program.Tokens[bodyClose].Start
	src := program.Text[start:end]
	var edits []functionValueEdit
	for i := bodyOpen + 1; i < bodyClose; i++ {
		name := functionValueTokenText(program, i)
		if !functionValueNameInList(captures, name) {
			continue
		}
		tok := program.Tokens[i]
		edits = append(edits, functionValueEdit{start: tok.Start - start, end: tok.Start + tok.Size - start, text: "env." + name})
	}
	out, ok := applyFunctionValueEdits(src, edits)
	if !ok {
		return ""
	}
	return string(out)
}

func appendFunctionValuePackageEdits(program *unit.Program, edits []functionValueEdit) []functionValueEdit {
	seen := false
	for i := 0; i+1 < len(program.Tokens); i++ {
		if !functionValueTokenEquals(program, i, "package") {
			continue
		}
		start := program.Tokens[i].Start
		end := program.Tokens[i+1].Start + program.Tokens[i+1].Size
		if !seen {
			edits = append(edits, functionValueEdit{start: start, end: end, text: "package main"})
			seen = true
		} else {
			edits = append(edits, functionValueEdit{start: start, end: end, text: ""})
		}
	}
	return edits
}

func reparseFunctionValueProgram(original *unit.Program, text []byte) bool {
	file := syntax.ParseFile(text)
	if !file.Ok {
		return false
	}
	out := unit.Program{Package: original.Package, ImportPath: original.ImportPath, Text: text}
	tokenMap := make([]int, len(file.Tokens)+1)
	for i := 0; i < len(file.Tokens); i++ {
		tok := file.Tokens[i]
		tokenMap[i] = len(out.Tokens)
		kind := functionValueUnitTokenKind(text, tok)
		if functionValueTokenIsEllipsis(text, tok) {
			for dot := 0; dot < 3; dot++ {
				out.Tokens = append(out.Tokens, unit.Token{Kind: kind, Start: tok.Start + dot, Size: 1, Line: tok.Line})
			}
		} else {
			out.Tokens = append(out.Tokens, unit.Token{Kind: kind, Start: tok.Start, Size: tok.End - tok.Start, Line: tok.Line})
		}
	}
	tokenMap[len(file.Tokens)] = len(out.Tokens)
	eof := len(out.Tokens) - 1
	for i := 0; i < len(file.Decls); i++ {
		decl := file.Decls[i]
		name := file.Tokens[decl.NameTok]
		out.Decls = append(out.Decls, unit.Decl{Kind: functionValueDeclKind(decl.Kind), NameStart: name.Start, NameEnd: name.End, StartTok: tokenMap[decl.StartTok], EndTok: tokenMap[decl.EndTok]})
	}
	for i := 0; i < len(file.Funcs); i++ {
		fn := file.Funcs[i]
		name := file.Tokens[fn.NameTok]
		receiverStart := fn.ReceiverStart
		receiverEnd := fn.ReceiverEnd
		if receiverStart < 0 {
			receiverStart = eof
			receiverEnd = eof
		} else {
			receiverStart = tokenMap[receiverStart]
			receiverEnd = tokenMap[receiverEnd]
		}
		out.Funcs = append(out.Funcs, unit.Func{NameStart: name.Start, NameEnd: name.End, StartTok: tokenMap[fn.StartTok], NameTok: tokenMap[fn.NameTok], ReceiverStart: receiverStart, ReceiverEnd: receiverEnd, BodyStart: tokenMap[fn.BodyStart], BodyEnd: tokenMap[fn.BodyEnd], EndTok: tokenMap[fn.EndTok]})
	}
	replaceFunctionValueProgram(original, &out)
	return true
}

func functionValueTokenIsEllipsis(src []byte, tok syntax.Token) bool {
	return tok.Kind == syntax.TokenOperator && tok.End-tok.Start == 3 && tok.Start >= 0 && tok.End <= len(src) && src[tok.Start] == '.' && src[tok.Start+1] == '.' && src[tok.Start+2] == '.'
}

func functionValueUnitTokenKind(src []byte, tok syntax.Token) int {
	if tok.Kind == syntax.TokenEOF {
		return unit.TokenEOF
	}
	if tok.Kind == syntax.TokenIdent {
		return unit.TokenIdent
	}
	if tok.Kind == syntax.TokenNumber {
		if syntax.NumberTokenIsFloat(src, tok) {
			return unit.TokenFloat
		}
		return unit.TokenNumber
	}
	if tok.Kind == syntax.TokenString {
		return unit.TokenString
	}
	if tok.Kind == syntax.TokenChar {
		return unit.TokenChar
	}
	if tok.Kind == syntax.TokenOperator {
		return unit.TokenOp
	}
	if tok.Kind == syntax.TokenPackage {
		return unit.TokenPackage
	}
	if tok.Kind == syntax.TokenConst {
		return unit.TokenConst
	}
	if tok.Kind == syntax.TokenVar {
		return unit.TokenVar
	}
	if tok.Kind == syntax.TokenType {
		return unit.TokenType
	}
	if tok.Kind == syntax.TokenFunc {
		return unit.TokenFunc
	}
	if tok.Kind == syntax.TokenStruct {
		return unit.TokenStruct
	}
	if tok.Kind == syntax.TokenReturn {
		return unit.TokenReturn
	}
	if tok.Kind == syntax.TokenIf {
		return unit.TokenIf
	}
	if tok.Kind == syntax.TokenElse {
		return unit.TokenElse
	}
	if tok.Kind == syntax.TokenFor {
		return unit.TokenFor
	}
	if tok.Kind == syntax.TokenBreak {
		return unit.TokenBreak
	}
	if tok.Kind == syntax.TokenContinue {
		return unit.TokenContinue
	}
	if tok.Kind == syntax.TokenGoto {
		return unit.TokenGoto
	}
	if tok.Kind == syntax.TokenSwitch {
		return unit.TokenSwitch
	}
	if tok.Kind == syntax.TokenCase {
		return unit.TokenCase
	}
	if tok.Kind == syntax.TokenDefault {
		return unit.TokenDefault
	}
	return unit.TokenIdent
}

func functionValueDeclKind(kind int) int {
	if kind == syntax.TokenConst {
		return unit.TokenConst
	}
	if kind == syntax.TokenVar {
		return unit.TokenVar
	}
	return unit.TokenType
}

func functionValueMethodReceiverType(program *unit.Program, method string) string {
	for i := 0; i < len(program.Funcs); i++ {
		fn := program.Funcs[i]
		if functionValueTokenText(program, fn.NameTok) != method || fn.ReceiverStart >= fn.ReceiverEnd {
			continue
		}
		start := fn.ReceiverStart
		end := fn.ReceiverEnd
		if functionValueTokenEquals(program, start, "(") {
			start++
		}
		if functionValueTokenEquals(program, end-1, ")") {
			end--
		}
		if end-start >= 2 && program.Tokens[start].Kind == unit.TokenIdent {
			start++
		}
		return functionValueTokensText(program, start, end)
	}
	return ""
}

func functionValueMethodReceiverTypeForBase(program *unit.Program, at int, base string, method string) string {
	baseType := functionValueEnclosingLocalType(program, at, base)
	if baseType == "" {
		return functionValueMethodReceiverType(program, method)
	}
	for i := 0; i < len(program.Funcs); i++ {
		fn := program.Funcs[i]
		if functionValueTokenText(program, fn.NameTok) != method || fn.ReceiverStart >= fn.ReceiverEnd {
			continue
		}
		receiverType := functionValueReceiverType(program, fn)
		if receiverType == baseType {
			return receiverType
		}
	}
	return functionValueMethodReceiverType(program, method)
}

func functionValueReceiverType(program *unit.Program, fn unit.Func) string {
	start := fn.ReceiverStart
	end := fn.ReceiverEnd
	if functionValueTokenEquals(program, start, "(") {
		start++
	}
	if functionValueTokenEquals(program, end-1, ")") {
		end--
	}
	if end-start >= 2 && program.Tokens[start].Kind == unit.TokenIdent {
		start++
	}
	return functionValueTokensText(program, start, end)
}

func functionValueSelectorFieldBefore(program *unit.Program, before int) int {
	if before < 3 || !functionValueTokenEquals(program, before-2, ".") {
		return -1
	}
	return before - 1
}

func functionValueSelectorStart(program *unit.Program, end int) int {
	start := end
	for start >= 2 && functionValueTokenEquals(program, start-1, ".") && program.Tokens[start-2].Kind == unit.TokenIdent {
		start -= 2
	}
	return start
}

func functionValueFieldByName(fields []functionValueField, name string) int {
	for i := 0; i < len(fields); i++ {
		if fields[i].name == name {
			return i
		}
	}
	return -1
}

func functionValueFieldByOwnerAndName(fields []functionValueField, owner string, name string) int {
	for i := 0; i < len(fields); i++ {
		if fields[i].owner == owner && fields[i].name == name {
			return i
		}
	}
	return -1
}

func functionValueFieldForSelector(program *unit.Program, fieldTok int, fields []functionValueField) int {
	if fieldTok < 2 {
		return -1
	}
	selectorStart := functionValueSelectorStart(program, fieldTok)
	if selectorStart < 0 || selectorStart >= fieldTok {
		return -1
	}
	baseType := functionValueEnclosingLocalType(program, fieldTok, functionValueTokenText(program, selectorStart))
	baseType = functionValueBareType(baseType)
	if baseType == "" {
		match := -1
		name := functionValueTokenText(program, fieldTok)
		for i := 0; i < len(fields); i++ {
			if fields[i].name == name {
				if match >= 0 {
					return -1
				}
				match = i
			}
		}
		return match
	}
	for i := selectorStart + 2; i < fieldTok; i += 2 {
		if !functionValueTokenEquals(program, i-1, ".") {
			return -1
		}
		baseType = functionValueBareType(functionValueStructFieldType(program, baseType, functionValueTokenText(program, i)))
		if baseType == "" {
			return -1
		}
	}
	name := functionValueTokenText(program, fieldTok)
	for i := 0; i < len(fields); i++ {
		if fields[i].name == name && functionValueTypeEmbeds(program, baseType, fields[i].owner, 0) {
			return i
		}
	}
	return -1
}

func functionValueStructFieldType(program *unit.Program, owner string, fieldName string) string {
	owner = functionValueBareType(owner)
	for i := 0; i < len(program.Decls); i++ {
		decl := program.Decls[i]
		if decl.Kind != unit.TokenType {
			continue
		}
		nameTok := functionValueTokenAtSpan(program, decl.NameStart, decl.NameEnd)
		if nameTok < 0 || functionValueTokenText(program, nameTok) != owner {
			continue
		}
		start := nameTok + 1
		if !functionValueTokenEquals(program, start, "struct") || !functionValueTokenEquals(program, start+1, "{") {
			return ""
		}
		close := functionValueFindMatchingBrace(program, start+1)
		j := start + 2
		for j < close {
			for j < close && functionValueTokenEquals(program, j, ";") {
				j++
			}
			if j >= close {
				break
			}
			lineEnd := j + 1
			for lineEnd < close && !functionValueTokenEquals(program, lineEnd, ";") && program.Tokens[lineEnd].Line == program.Tokens[j].Line {
				lineEnd++
			}
			if functionValueTokenEquals(program, j, fieldName) && j+1 < lineEnd {
				return functionValueTokensText(program, j+1, lineEnd)
			}
			j = lineEnd
		}
		return ""
	}
	return ""
}

func functionValueBareType(typ string) string {
	for len(typ) > 0 && typ[0] == '*' {
		typ = typ[1:]
	}
	lastDot := -1
	for i := 0; i < len(typ); i++ {
		if typ[i] == '.' {
			lastDot = i
		}
	}
	if lastDot >= 0 {
		return typ[lastDot+1:]
	}
	return typ
}

func functionValueTypeEmbeds(program *unit.Program, actual string, wanted string, depth int) bool {
	actual = functionValueBareType(actual)
	wanted = functionValueBareType(wanted)
	if actual == wanted {
		return true
	}
	if depth >= 8 {
		return false
	}
	for i := 0; i < len(program.Decls); i++ {
		decl := program.Decls[i]
		if decl.Kind != unit.TokenType {
			continue
		}
		nameTok := functionValueTokenAtSpan(program, decl.NameStart, decl.NameEnd)
		if nameTok < 0 || functionValueTokenText(program, nameTok) != actual {
			continue
		}
		start := nameTok + 1
		if !functionValueTokenEquals(program, start, "struct") || !functionValueTokenEquals(program, start+1, "{") {
			return false
		}
		close := functionValueFindMatchingBrace(program, start+1)
		for j := start + 2; j < close; j++ {
			if j > start+2 && !functionValueTokenEquals(program, j-1, ";") && program.Tokens[j-1].Line == program.Tokens[j].Line {
				continue
			}
			embeddedStart := j
			if functionValueTokenEquals(program, j, "*") {
				embeddedStart++
			}
			if program.Tokens[embeddedStart].Kind != unit.TokenIdent {
				continue
			}
			embeddedEnd := embeddedStart + 1
			if functionValueTokenEquals(program, embeddedEnd, ".") && embeddedEnd+1 < close && program.Tokens[embeddedEnd+1].Kind == unit.TokenIdent {
				embeddedEnd += 2
			}
			if embeddedEnd < close && !functionValueTokenEquals(program, embeddedEnd, ";") && program.Tokens[embeddedEnd].Line == program.Tokens[embeddedStart].Line {
				continue
			}
			embeddedType := functionValueBareType(functionValueTokensText(program, j, embeddedEnd))
			if functionValueTypeEmbeds(program, embeddedType, wanted, depth+1) {
				return true
			}
		}
		return false
	}
	return false
}

func functionValueSignatureByName(signatures []functionValueSignature, name string) int {
	for i := 0; i < len(signatures); i++ {
		if signatures[i].name == name {
			return i
		}
	}
	return -1
}

func functionValueImplIndex(sig functionValueSignature, receiverType string, method string, function string) int {
	for i := 0; i < len(sig.impls); i++ {
		impl := sig.impls[i]
		if impl.receiverType == receiverType && impl.method == method && impl.function == function {
			return i
		}
	}
	return -1
}

func functionValueTokenCanStartType(program *unit.Program, tok int) bool {
	if tok < 0 || tok >= len(program.Tokens) {
		return false
	}
	text := functionValueTokenText(program, tok)
	return program.Tokens[tok].Kind == unit.TokenIdent || text == "*" || text == "[" || text == "struct"
}

func functionValueTokenAtSpan(program *unit.Program, start int, end int) int {
	for i := 0; i < len(program.Tokens); i++ {
		tok := program.Tokens[i]
		if tok.Start == start && tok.Start+tok.Size == end {
			return i
		}
	}
	return -1
}

func functionValueTokensText(program *unit.Program, start int, end int) string {
	if start < 0 || start >= end || end > len(program.Tokens) {
		return ""
	}
	byteStart := program.Tokens[start].Start
	last := program.Tokens[end-1]
	byteEnd := last.Start + last.Size
	if byteStart < 0 || byteEnd > len(program.Text) {
		return ""
	}
	return string(program.Text[byteStart:byteEnd])
}

func functionValueTokenEdit(program *unit.Program, tok int, replacement string) functionValueEdit {
	item := program.Tokens[tok]
	return functionValueEdit{start: item.Start, end: item.Start + item.Size, text: replacement}
}

func functionValueTokenRangeEdit(program *unit.Program, start int, end int, replacement string) functionValueEdit {
	first := program.Tokens[start]
	if end <= start {
		return functionValueEdit{start: first.Start, end: first.Start, text: replacement}
	}
	last := program.Tokens[end-1]
	return functionValueEdit{start: first.Start, end: last.Start + last.Size, text: replacement}
}

func applyFunctionValueEdits(src []byte, edits []functionValueEdit) ([]byte, bool) {
	for i := 0; i < len(edits); i++ {
		best := i
		for j := i + 1; j < len(edits); j++ {
			if edits[j].start < edits[best].start || edits[j].start == edits[best].start && edits[j].end < edits[best].end {
				best = j
			}
		}
		edits[i], edits[best] = edits[best], edits[i]
	}
	var out []byte
	pos := 0
	for i := 0; i < len(edits); i++ {
		edit := edits[i]
		if edit.start < pos || edit.end < edit.start || edit.end > len(src) {
			return nil, false
		}
		out = append(out, src[pos:edit.start]...)
		out = appendFunctionValueString(out, edit.text)
		pos = edit.end
	}
	out = append(out, src[pos:]...)
	return out, true
}

func appendFunctionValueString(out []byte, text string) []byte {
	for i := 0; i < len(text); i++ {
		out = append(out, text[i])
	}
	return out
}

func functionValueJoin(items []string, separator string) string {
	out := ""
	for i := 0; i < len(items); i++ {
		if i > 0 {
			out = out + separator
		}
		out = out + items[i]
	}
	return out
}

func functionValueZero(result string) string {
	if result == "string" {
		return "\"\""
	}
	if result == "bool" {
		return "false"
	}
	if len(result) > 0 && result[0] == '*' {
		return "nil"
	}
	return "0"
}

func functionValueDecimal(value int) string {
	if value == 0 {
		return "0"
	}
	var digits []byte
	for value > 0 {
		digits = append(digits, byte('0'+value%10))
		value /= 10
	}
	var out []byte
	for i := len(digits) - 1; i >= 0; i-- {
		out = append(out, digits[i])
	}
	return string(out)
}

func functionValueNameInList(list []string, name string) bool {
	if name == "" {
		return false
	}
	for i := 0; i < len(list); i++ {
		if list[i] == name {
			return true
		}
	}
	return false
}

func functionValueFindMatchingParen(program *unit.Program, open int) int {
	return functionValueFindMatching(program, open, "(", ")")
}

func functionValueFindMatchingBrace(program *unit.Program, open int) int {
	return functionValueFindMatching(program, open, "{", "}")
}

func functionValueFindMatching(program *unit.Program, open int, left string, right string) int {
	if !functionValueTokenEquals(program, open, left) {
		return -1
	}
	depth := 0
	for i := open; i < len(program.Tokens); i++ {
		if functionValueTokenEquals(program, i, left) {
			depth++
		} else if functionValueTokenEquals(program, i, right) {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func functionValueTokenEquals(program *unit.Program, tok int, want string) bool {
	if tok < 0 || tok >= len(program.Tokens) {
		return false
	}
	token := program.Tokens[tok]
	if token.Start < 0 || token.Size != len(want) || token.Start+token.Size > len(program.Text) {
		return false
	}
	for i := 0; i < len(want); i++ {
		if program.Text[token.Start+i] != want[i] {
			return false
		}
	}
	return true
}

func functionValueTokenText(program *unit.Program, tok int) string {
	if tok < 0 || tok >= len(program.Tokens) {
		return ""
	}
	token := program.Tokens[tok]
	if token.Start < 0 || token.Start+token.Size > len(program.Text) {
		return ""
	}
	return string(program.Text[token.Start : token.Start+token.Size])
}
