package link

import (
	"renvo.dev/internal/arena"
	"renvo.dev/internal/syntax"
	"renvo.dev/internal/unit"
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
	zeroType    string
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

func renvo_runtime_ArenaDiscardLinkTokens(tokens []unit.Token) {}

func lowerFunctionValuesCore(program *unit.Program, transient bool) bool {
	functions, deferred, builtins := functionValueProgramNeedsLowering(program)
	if deferred {
		if !lowerDeferredBuiltins(program, transient) {
			return false
		}
		functions = true
	}
	if builtins {
		if !lowerOrdinaryBuiltins(program, transient) {
			return false
		}
		functions = true
	}
	if !functions {
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
	edits = lowerFunctionValueCallArguments(program, signatures, edits)
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
	originalLength := len(program.Text)
	if transient {
		renvo_runtime_ArenaDiscardLinkTokens(program.Tokens)
	}
	text, ok := applyFunctionValueEdits(program.Text, edits)
	if !ok {
		return false
	}
	if len(text) > 0 && text[len(text)-1] != '\n' {
		text = append(text, '\n')
	}
	generatedStart := len(text)
	text = appendFunctionValueString(text, generated)
	if transient {
		arena.DiscardBytes(program.Text)
	}
	return reparseFunctionValueProgram(program, text, edits, originalLength, generatedStart)
}

func lowerDeferredBuiltins(program *unit.Program, transient bool) bool {
	var edits []functionValueEdit
	for i := 0; i+2 < len(program.Tokens); i++ {
		if !functionValueTokenEquals(program, i, "defer") || !functionValueTokenEquals(program, i+2, "(") {
			continue
		}
		name := functionValueTokenText(program, i+1)
		if name != "copy" && name != "delete" && name != "panic" && name != "print" && name != "println" && name != "recover" {
			continue
		}
		if functionValueEnclosingLocalType(program, i, name) != "" || functionValueDeclaredFunction(program, name) {
			continue
		}
		close := functionValueFindMatchingParen(program, i+2)
		if close < 0 {
			return false
		}
		var starts []int
		var ends []int
		start := i + 3
		depth := 0
		for tok := start; tok <= close; tok++ {
			text := functionValueTokenText(program, tok)
			if tok < close && (text == "(" || text == "[" || text == "{") {
				depth++
			} else if tok < close && (text == ")" || text == "]" || text == "}") {
				depth--
			}
			if tok != close && !(depth == 0 && text == ",") {
				continue
			}
			if start < tok {
				starts = append(starts, start)
				ends = append(ends, tok)
			}
			start = tok + 1
		}
		if name == "recover" {
			edits = append(edits, functionValueTokenRangeEdit(program, i, close+1, "defer func(){}()"))
			i = close
			continue
		}
		params := ""
		args := ""
		bodyArgs := ""
		for arg := 0; arg < len(starts); arg++ {
			typ := deferredBuiltinArgumentType(program, i, starts[arg], ends[arg])
			if typ == "" {
				return false
			}
			if arg > 0 {
				params += ", "
				args += ", "
				bodyArgs += ", "
			}
			argName := "__renvo_defer_" + functionValueDecimal(i) + "_" + functionValueDecimal(arg)
			params += argName + " " + typ
			args += functionValueTokensText(program, starts[arg], ends[arg])
			bodyArgs += argName
		}
		replacement := "defer func(" + params + "){" + name + "(" + bodyArgs + ")}(" + args + ")"
		edits = append(edits, functionValueTokenRangeEdit(program, i, close+1, replacement))
		i = close
	}
	if len(edits) == 0 {
		return true
	}
	originalLength := len(program.Text)
	if transient {
		renvo_runtime_ArenaDiscardLinkTokens(program.Tokens)
	}
	text, ok := applyFunctionValueEdits(program.Text, edits)
	if !ok {
		return false
	}
	if transient {
		arena.DiscardBytes(program.Text)
	}
	return ok && reparseFunctionValueProgram(program, text, edits, originalLength, -1)
}

func deferredBuiltinArgumentType(program *unit.Program, before int, start int, end int) string {
	return ordinaryBuiltinExprType(program, before, start, end)
}

func functionValueDeclaredFunction(program *unit.Program, name string) bool {
	for i := 0; i < len(program.Funcs); i++ {
		if functionValueTokenText(program, program.Funcs[i].NameTok) == name {
			return true
		}
	}
	return false
}

func functionValueProgramNeedsLowering(program *unit.Program) (bool, bool, bool) {
	functions := false
	deferred := false
	builtins := false
	for i := 0; i+1 < len(program.Tokens); i++ {
		token := program.Tokens[i]
		start := token.Start
		valid := start >= 0 && start+token.Size <= len(program.Text)
		if token.KindLine&255 == unit.TokenFunc && functionValueTokenEquals(program, i+1, "(") && !functionValueIsDeclaredFunction(program, i) {
			functions = true
		}
		if i+2 < len(program.Tokens) && valid && token.Size == 5 && program.Text[start] == 'd' && program.Text[start+1] == 'e' && program.Text[start+2] == 'f' && program.Text[start+3] == 'e' && program.Text[start+4] == 'r' && functionValueTokenEquals(program, i+2, "(") {
			name := functionValueTokenText(program, i+1)
			if (name == "copy" || name == "delete" || name == "panic" || name == "print" || name == "println" || name == "recover") && functionValueEnclosingLocalType(program, i, name) == "" && !functionValueDeclaredFunction(program, name) {
				deferred = true
			}
		}
		name := ""
		if valid && token.KindLine&255 == unit.TokenIdent {
			if token.Size == 3 && program.Text[start] == 'm' && program.Text[start+2] == 'n' {
				if program.Text[start+1] == 'i' {
					name = "min"
				} else if program.Text[start+1] == 'a' {
					name = "max"
				}
			} else if token.Size == 5 && program.Text[start] == 'c' && program.Text[start+1] == 'l' && program.Text[start+2] == 'e' && program.Text[start+3] == 'a' && program.Text[start+4] == 'r' {
				name = "clear"
			}
		}
		if name != "" && functionValueTokenEquals(program, i+1, "(") && !ordinaryBuiltinShadowed(program, i, name) {
			builtins = true
		}
	}
	return functions, deferred, builtins
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
			if program.Tokens[j].KindLine&255 != unit.TokenIdent {
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
				name := "__renvo_function_" + functionValueDecimal(len(signatures))
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
			if program.Tokens[j].KindLine&255 != unit.TokenIdent {
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
		zeroType := functionValueSingleResultType(program, end+1, resultClose)
		if functionValueZero(zeroType) == "0" && !functionValueCanUseScalarZero(zeroType) {
			sig.zeroType = zeroType
		}
		end = resultClose + 1
	} else if functionValueTokenCanStartType(program, end) && program.Tokens[end].KindLine>>8 == program.Tokens[close].KindLine>>8 {
		resultEnd := functionValueTypeEnd(program, end)
		if resultEnd <= end {
			return sig, funcTok, false
		}
		sig.result = functionValueTokensText(program, end, resultEnd)
		if functionValueZero(sig.result) == "0" && !functionValueCanUseScalarZero(sig.result) {
			sig.zeroType = sig.result
		}
		end = resultEnd
	}
	return sig, end, true
}

func functionValueTypeEnd(program *unit.Program, start int) int {
	if start < 0 || start >= len(program.Tokens) {
		return start
	}
	text := functionValueTokenText(program, start)
	if text == "*" {
		return functionValueTypeEnd(program, start+1)
	}
	if text == "[" {
		close := functionValueFindMatching(program, start, "[", "]")
		if close < 0 {
			return start
		}
		return functionValueTypeEnd(program, close+1)
	}
	if text == "map" {
		if !functionValueTokenEquals(program, start+1, "[") {
			return start
		}
		close := functionValueFindMatching(program, start+1, "[", "]")
		if close < 0 {
			return start
		}
		return functionValueTypeEnd(program, close+1)
	}
	if text == "struct" || text == "interface" {
		if !functionValueTokenEquals(program, start+1, "{") {
			return start
		}
		close := functionValueFindMatchingBrace(program, start+1)
		if close < 0 {
			return start
		}
		return close + 1
	}
	if text == "func" {
		if !functionValueTokenEquals(program, start+1, "(") {
			return start
		}
		close := functionValueFindMatchingParen(program, start+1)
		if close < 0 {
			return start
		}
		end := close + 1
		if functionValueTokenEquals(program, end, "(") {
			resultClose := functionValueFindMatchingParen(program, end)
			if resultClose < 0 {
				return start
			}
			return resultClose + 1
		}
		resultEnd := functionValueTypeEnd(program, end)
		if resultEnd > end {
			return resultEnd
		}
		return end
	}
	if text == "(" {
		close := functionValueFindMatchingParen(program, start)
		if close < 0 {
			return start
		}
		return close + 1
	}
	if program.Tokens[start].KindLine&255 == unit.TokenIdent {
		end := start + 1
		if functionValueTokenEquals(program, end, ".") && end+1 < len(program.Tokens) && program.Tokens[end+1].KindLine&255 == unit.TokenIdent {
			end += 2
		}
		return end
	}
	return start
}

func functionValueSingleResultType(program *unit.Program, start int, end int) string {
	if start >= end {
		return ""
	}
	depth := 0
	for i := start; i < end; i++ {
		text := functionValueTokenText(program, i)
		if text == "(" || text == "[" || text == "{" {
			depth++
		} else if text == ")" || text == "]" || text == "}" {
			depth--
		} else if text == "," && depth == 0 {
			return ""
		}
	}
	typeStart := start
	if start+1 < end && program.Tokens[start].KindLine&255 == unit.TokenIdent && functionValueTypeEnd(program, start+1) == end {
		typeStart++
	}
	if functionValueTypeEnd(program, typeStart) != end {
		return ""
	}
	return functionValueTokensText(program, typeStart, end)
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
		if partLen >= 2 && program.Tokens[partStarts[i]].KindLine&255 == unit.TokenIdent {
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
	return lowerFunctionValueBoundMethod(program, op, rhs, sigIndex, signatures, edits), true
}

func lowerFunctionValueBoundMethod(program *unit.Program, at int, rhs int, sigIndex int, signatures []functionValueSignature, edits []functionValueEdit) []functionValueEdit {
	if rhs+2 >= len(program.Tokens) || !functionValueTokenEquals(program, rhs+1, ".") || program.Tokens[rhs].KindLine&255 != unit.TokenIdent || program.Tokens[rhs+2].KindLine&255 != unit.TokenIdent {
		return edits
	}
	method := functionValueTokenText(program, rhs+2)
	receiverType := functionValueMethodReceiverTypeForBase(program, at, functionValueTokenText(program, rhs), method)
	if receiverType == "" {
		return edits
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
	receiver := functionValueTokenText(program, rhs)
	baseType := functionValueEnclosingLocalType(program, at, receiver)
	if len(receiverType) > 0 && receiverType[0] == '*' && len(baseType) > 0 && baseType[0] != '*' {
		receiver = "&" + receiver
	}
	replacement := signatures[sigIndex].name + "{kind: " + functionValueDecimal(implIndex+1) + ", " + impl.receiverField + ": " + receiver + "}"
	edits = append(edits, functionValueTokenRangeEdit(program, rhs, rhs+3, replacement))
	return edits
}

func lowerFunctionValueCallArguments(program *unit.Program, signatures []functionValueSignature, edits []functionValueEdit) []functionValueEdit {
	for open := 0; open < len(program.Tokens); open++ {
		if !functionValueTokenEquals(program, open, "(") || functionValueCallIsDeclaration(program, open) {
			continue
		}
		fn, ok := functionValueCalledFunction(program, open)
		if !ok {
			continue
		}
		close := functionValueFindMatchingParen(program, open)
		if close <= open {
			continue
		}
		paramTypes := functionValueFunctionParamTypes(program, fn)
		argStarts, argEnds := functionValueCommaParts(program, open+1, close)
		if len(argStarts) != len(paramTypes) {
			continue
		}
		for i := 0; i < len(argStarts); i++ {
			if argEnds[i]-argStarts[i] != 3 {
				continue
			}
			sigIndex := functionValueSignatureByName(signatures, functionValueBareType(paramTypes[i]))
			if sigIndex < 0 {
				continue
			}
			edits = lowerFunctionValueBoundMethod(program, open, argStarts[i], sigIndex, signatures, edits)
		}
	}
	return edits
}

func functionValueCallIsDeclaration(program *unit.Program, open int) bool {
	for i := 0; i < len(program.Funcs); i++ {
		if program.Funcs[i].NameTok+1 == open {
			return true
		}
	}
	return false
}

func functionValueCalledFunction(program *unit.Program, open int) (unit.Func, bool) {
	var fallback unit.Func
	fallbackOK := false
	if open < 1 || program.Tokens[open-1].KindLine&255 != unit.TokenIdent {
		return fallback, false
	}
	name := functionValueTokenText(program, open-1)
	selector := open >= 3 && functionValueTokenEquals(program, open-2, ".")
	baseType := ""
	fallbackCount := 0
	if selector {
		selectorStart := functionValueSelectorStart(program, open-1)
		if selectorStart >= 0 {
			baseType = functionValueEnclosingLocalType(program, open, functionValueTokenText(program, selectorStart))
		}
	}
	for i := 0; i < len(program.Funcs); i++ {
		fn := program.Funcs[i]
		if functionValueTokenText(program, fn.NameTok) != name {
			continue
		}
		method := fn.ReceiverStart < fn.ReceiverEnd
		if method != selector {
			continue
		}
		if !fallbackOK {
			fallback = fn
			fallbackOK = true
		}
		fallbackCount++
		if !method || baseType != "" && functionValueTypeEmbeds(program, baseType, functionValueReceiverType(program, fn), 0) {
			return fn, true
		}
	}
	return fallback, fallbackOK && fallbackCount == 1
}

func functionValueCommaParts(program *unit.Program, start int, end int) ([]int, []int) {
	var starts []int
	var ends []int
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
			starts = append(starts, partStart)
			ends = append(ends, i)
		}
		partStart = i + 1
	}
	return starts, ends
}

func functionValueFunctionParamTypes(program *unit.Program, fn unit.Func) []string {
	open := fn.NameTok + 1
	if !functionValueTokenEquals(program, open, "(") {
		return nil
	}
	close := functionValueFindMatchingParen(program, open)
	starts, ends := functionValueCommaParts(program, open+1, close)
	types := make([]string, len(starts))
	carried := ""
	for i := len(starts) - 1; i >= 0; i-- {
		start := starts[i]
		end := ends[i]
		if start+1 < end && program.Tokens[start].KindLine&255 == unit.TokenIdent && functionValueTypeEnd(program, start+1) == end {
			carried = functionValueTokensText(program, start+1, end)
			types[i] = carried
		} else if end == start+1 && carried != "" {
			types[i] = carried
		} else {
			carried = ""
			types[i] = functionValueTokensText(program, start, end)
		}
	}
	return types
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
	edits = append(edits, functionValueEdit{start: start, end: start, text: "__renvo_call_" + functionValueDecimal(fields[fieldIndex].sig) + "(&"})
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
		out = out + "func __renvo_call_" + functionValueDecimal(i) + "(fn *" + sig.name
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
			if sig.zeroType != "" {
				out = out + "var __renvo_zero " + sig.zeroType + "\nreturn __renvo_zero\n"
			} else {
				out = out + "return " + functionValueZero(sig.result) + "\n"
			}
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
		fieldName := functionValueTokenText(program, fieldTok)
		fieldIndex := functionValueFieldByOwnerAndName(fields, functionValueCompositeOwner(program, fieldTok), fieldName)
		if fieldIndex < 0 {
			fieldIndex = functionValueUniqueFieldByName(fields, fieldName)
		}
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
		envName := "__renvo_closure_env_" + functionValueDecimal(closureIndex)
		funcName := "__renvo_closure_" + functionValueDecimal(closureIndex)
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
		if program.Tokens[i].KindLine&255 != unit.TokenIdent {
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
	fnIndex := functionValueEnclosingFunc(program, before)
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
			if functionValueTokenEquals(program, rhs, "&") && rhs+1 < before && program.Tokens[rhs+1].KindLine&255 == unit.TokenIdent {
				return "*" + functionValueTokenText(program, rhs+1)
			}
			typeEnd := functionValueTypeEnd(program, rhs)
			if typeEnd > rhs && functionValueTokenEquals(program, typeEnd, "{") {
				return functionValueTokensText(program, rhs, typeEnd)
			}
			if program.Tokens[rhs].KindLine&255 == unit.TokenNumber {
				return "int"
			}
			if program.Tokens[rhs].KindLine&255 == unit.TokenString {
				return "string"
			}
			if program.Tokens[rhs].KindLine&255 == unit.TokenIdent {
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
					if functionValueDeclaredType(program, callName) {
						return callName
					}
				}
				if typ := functionValueFunctionParamType(program, fn, functionValueTokenText(program, rhs)); typ != "" {
					return typ
				}
			}
		}
		if i > 0 && functionValueTokenEquals(program, i-1, "var") && i+1 < before {
			return functionValueTokensText(program, i+1, functionValueTypeEnd(program, i+1))
		}
	}
	return functionValueFunctionParamType(program, fn, name)
}

func functionValueDeclaredType(program *unit.Program, name string) bool {
	for i := 0; i < len(program.Decls); i++ {
		decl := program.Decls[i]
		if decl.Kind == unit.TokenType && functionValueTokenText(program, functionValueTokenAtSpan(program, decl.NameStart, decl.NameEnd)) == name {
			return true
		}
	}
	return false
}

func functionValueEnclosingFunc(program *unit.Program, token int) int {
	low := 0
	high := len(program.Funcs)
	for low < high {
		middle := low + (high-low)/2
		if program.Funcs[middle].BodyStart < token {
			low = middle + 1
		} else {
			high = middle
		}
	}
	if low > 0 {
		candidate := low - 1
		if token < program.Funcs[candidate].BodyEnd {
			return candidate
		}
	}
	return -1
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

func reparseFunctionValueProgram(original *unit.Program, text []byte, edits []functionValueEdit, originalLength int, generatedStart int) bool {
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
				out.Tokens = append(out.Tokens, unit.MakeToken(kind, tok.Start+dot, 1, syntax.TokenLine(tok)))
			}
		} else {
			out.Tokens = append(out.Tokens, unit.MakeToken(kind, tok.Start, tok.End-tok.Start, syntax.TokenLine(tok)))
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
	out.Packages = remapFunctionValuePackages(original, &out, edits, originalLength, generatedStart)
	replaceFunctionValueProgram(original, &out)
	return true
}

func remapFunctionValuePackages(original *unit.Program, reparsed *unit.Program, edits []functionValueEdit, originalLength int, generatedStart int) []unit.PackageInfo {
	if len(original.Packages) == 0 {
		return nil
	}
	packages := make([]unit.PackageInfo, 0, len(original.Packages)+1)
	root := -1
	for i := 0; i < len(original.Packages); i++ {
		item := original.Packages[i]
		item.TextStart = mapFunctionValueOffset(item.TextStart, edits, originalLength)
		item.TextEnd = mapFunctionValueOffset(item.TextEnd, edits, originalLength)
		setFunctionValuePackageTableRanges(&item, reparsed)
		if item.ImportPath == original.ImportPath {
			root = len(packages)
		}
		packages = append(packages, item)
	}
	if generatedStart >= 0 && generatedStart < len(reparsed.Text) && root >= 0 {
		if packages[root].TextEnd == generatedStart {
			packages[root].TextEnd = len(reparsed.Text)
			setFunctionValuePackageTableRanges(&packages[root], reparsed)
		} else {
			generated := packages[root]
			generated.TextStart = generatedStart
			generated.TextEnd = len(reparsed.Text)
			setFunctionValuePackageTableRanges(&generated, reparsed)
			packages = append(packages, generated)
		}
	}
	return packages
}

func mapFunctionValueOffset(position int, edits []functionValueEdit, sourceLength int) int {
	if position < 0 {
		return 0
	}
	if position > sourceLength {
		position = sourceLength
	}
	delta := 0
	for i := 0; i < len(edits); i++ {
		edit := edits[i]
		if position < edit.start {
			break
		}
		if position < edit.end {
			return edit.start + delta
		}
		delta += len(edit.text) - (edit.end - edit.start)
	}
	return position + delta
}

func setFunctionValuePackageTableRanges(item *unit.PackageInfo, program *unit.Program) {
	item.TokenStart, item.TokenEnd = functionValueTokenRangeForText(program.Tokens, item.TextStart, item.TextEnd)
	item.DeclStart, item.DeclEnd = functionValueDeclRangeForText(program.Decls, item.TextStart, item.TextEnd)
	item.FuncStart, item.FuncEnd = functionValueFuncRangeForText(program.Funcs, item.TextStart, item.TextEnd)
}

func functionValueTokenRangeForText(items []unit.Token, start int, end int) (int, int) {
	first := len(items)
	last := len(items)
	for i := 0; i < len(items); i++ {
		if items[i].Start >= start && items[i].Start < end {
			if first == len(items) {
				first = i
			}
			last = i + 1
		}
	}
	return first, last
}

func functionValueDeclRangeForText(items []unit.Decl, start int, end int) (int, int) {
	first := len(items)
	last := len(items)
	for i := 0; i < len(items); i++ {
		if items[i].NameStart >= start && items[i].NameStart < end {
			if first == len(items) {
				first = i
			}
			last = i + 1
		}
	}
	return first, last
}

func functionValueFuncRangeForText(items []unit.Func, start int, end int) (int, int) {
	first := len(items)
	last := len(items)
	for i := 0; i < len(items); i++ {
		if items[i].NameStart >= start && items[i].NameStart < end {
			if first == len(items) {
				first = i
			}
			last = i + 1
		}
	}
	return first, last
}

func functionValueTokenIsEllipsis(src []byte, tok syntax.Token) bool {
	return tok.KindLine&255 == syntax.TokenOperator && tok.End-tok.Start == 3 && tok.Start >= 0 && tok.End <= len(src) && src[tok.Start] == '.' && src[tok.Start+1] == '.' && src[tok.Start+2] == '.'
}

func functionValueUnitTokenKind(src []byte, tok syntax.Token) int {
	if tok.KindLine&255 == syntax.TokenEOF {
		return unit.TokenEOF
	}
	if tok.KindLine&255 == syntax.TokenIdent {
		return unit.TokenIdent
	}
	if tok.KindLine&255 == syntax.TokenNumber {
		if syntax.NumberTokenIsFloat(src, tok) {
			return unit.TokenFloat
		}
		return unit.TokenNumber
	}
	if tok.KindLine&255 == syntax.TokenString {
		return unit.TokenString
	}
	if tok.KindLine&255 == syntax.TokenChar {
		return unit.TokenChar
	}
	if tok.KindLine&255 == syntax.TokenOperator {
		return unit.TokenOp
	}
	if tok.KindLine&255 == syntax.TokenPackage {
		return unit.TokenPackage
	}
	if tok.KindLine&255 == syntax.TokenConst {
		return unit.TokenConst
	}
	if tok.KindLine&255 == syntax.TokenVar {
		return unit.TokenVar
	}
	if tok.KindLine&255 == syntax.TokenType {
		return unit.TokenType
	}
	if tok.KindLine&255 == syntax.TokenFunc {
		return unit.TokenFunc
	}
	if tok.KindLine&255 == syntax.TokenStruct {
		return unit.TokenStruct
	}
	if tok.KindLine&255 == syntax.TokenReturn {
		return unit.TokenReturn
	}
	if tok.KindLine&255 == syntax.TokenIf {
		return unit.TokenIf
	}
	if tok.KindLine&255 == syntax.TokenElse {
		return unit.TokenElse
	}
	if tok.KindLine&255 == syntax.TokenFor {
		return unit.TokenFor
	}
	if tok.KindLine&255 == syntax.TokenBreak {
		return unit.TokenBreak
	}
	if tok.KindLine&255 == syntax.TokenContinue {
		return unit.TokenContinue
	}
	if tok.KindLine&255 == syntax.TokenGoto {
		return unit.TokenGoto
	}
	if tok.KindLine&255 == syntax.TokenSwitch {
		return unit.TokenSwitch
	}
	if tok.KindLine&255 == syntax.TokenCase {
		return unit.TokenCase
	}
	if tok.KindLine&255 == syntax.TokenDefault {
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
		if end-start >= 2 && program.Tokens[start].KindLine&255 == unit.TokenIdent {
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
	if end-start >= 2 && program.Tokens[start].KindLine&255 == unit.TokenIdent {
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
	for start >= 2 && functionValueTokenEquals(program, start-1, ".") {
		start = functionValuePrimaryStart(program, start-2)
		if start < 0 {
			return -1
		}
	}
	return start
}

func functionValuePrimaryStart(program *unit.Program, end int) int {
	if end < 0 || end >= len(program.Tokens) {
		return -1
	}
	start := end
	if functionValueTokenEquals(program, end, "]") {
		open := functionValueFindMatchingBackward(program, end, "[", "]")
		if open <= 0 {
			return -1
		}
		start = functionValuePrimaryStart(program, open-1)
	} else if functionValueTokenEquals(program, end, ")") {
		open := functionValueFindMatchingBackward(program, end, "(", ")")
		if open < 0 {
			return -1
		}
		start = open
		if open > 0 && (program.Tokens[open-1].KindLine&255 == unit.TokenIdent || functionValueTokenEquals(program, open-1, "]") || functionValueTokenEquals(program, open-1, ")")) {
			start = functionValuePrimaryStart(program, open-1)
		}
	}
	for start >= 2 && functionValueTokenEquals(program, start-1, ".") {
		start = functionValuePrimaryStart(program, start-2)
		if start < 0 {
			return -1
		}
	}
	return start
}

func functionValueFindMatchingBackward(program *unit.Program, close int, openText string, closeText string) int {
	depth := 0
	for i := close; i >= 0; i-- {
		if functionValueTokenEquals(program, i, closeText) {
			depth++
		} else if functionValueTokenEquals(program, i, openText) {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func functionValueUniqueFieldByName(fields []functionValueField, name string) int {
	match := -1
	for i := 0; i < len(fields); i++ {
		if fields[i].name != name {
			continue
		}
		if match >= 0 {
			return -1
		}
		match = i
	}
	return match
}

func functionValueCompositeOwner(program *unit.Program, before int) string {
	depth := 0
	for i := before - 1; i >= 1; i-- {
		if functionValueTokenEquals(program, i, "}") {
			depth++
			continue
		}
		if !functionValueTokenEquals(program, i, "{") {
			continue
		}
		if depth > 0 {
			depth--
			continue
		}
		if program.Tokens[i-1].KindLine&255 == unit.TokenIdent {
			return functionValueTokenText(program, i-1)
		}
		return ""
	}
	return ""
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
	name := functionValueTokenText(program, fieldTok)
	possible := false
	for i := 0; i < len(fields); i++ {
		if fields[i].name == name {
			possible = true
			break
		}
	}
	if !possible {
		return -1
	}
	selectorStart := functionValueSelectorStart(program, fieldTok)
	if selectorStart < 0 || selectorStart >= fieldTok {
		return -1
	}
	baseType := functionValueEnclosingLocalType(program, fieldTok, functionValueTokenText(program, selectorStart))
	baseType = functionValueBareType(baseType)
	if baseType == "" {
		return functionValueUniqueFieldByName(fields, name)
	}
	for i := selectorStart + 1; i < fieldTok-1; {
		if functionValueTokenEquals(program, i, "[") {
			close := functionValueFindMatching(program, i, "[", "]")
			if close < 0 || close >= fieldTok {
				return -1
			}
			baseType = functionValueElementType(baseType)
			i = close + 1
			continue
		}
		if functionValueTokenEquals(program, i, ".") && i+1 < fieldTok {
			baseType = functionValueBareType(functionValueStructFieldType(program, baseType, functionValueTokenText(program, i+1)))
			if baseType == "" {
				return -1
			}
			i += 2
			continue
		}
		return -1
	}
	// A method declared on the selected type wins over a field promoted from
	// an embedded struct at a greater selector depth. Treating the promoted
	// callback as the selected member rewrites an ordinary method call into a
	// call through the embedded function field.
	if functionValueTypeHasDirectMethod(program, baseType, name) {
		return -1
	}
	for i := 0; i < len(fields); i++ {
		if fields[i].name == name && functionValueTypeEmbeds(program, baseType, fields[i].owner, 0) {
			return i
		}
	}
	return -1
}

func functionValueTypeHasDirectMethod(program *unit.Program, typ string, name string) bool {
	typ = functionValueBareType(typ)
	for i := 0; i < len(program.Funcs); i++ {
		fn := program.Funcs[i]
		if fn.ReceiverStart >= fn.ReceiverEnd || functionValueTokenText(program, fn.NameTok) != name {
			continue
		}
		if functionValueBareType(functionValueReceiverType(program, fn)) == typ {
			return true
		}
	}
	return false
}

func functionValueElementType(typ string) string {
	for len(typ) > 0 && typ[0] == '*' {
		typ = typ[1:]
	}
	if len(typ) >= 2 && typ[0] == '[' && typ[1] == ']' {
		return functionValueBareType(typ[2:])
	}
	if len(typ) > 1 && typ[0] == '[' {
		for i := 1; i < len(typ); i++ {
			if typ[i] == ']' {
				return functionValueBareType(typ[i+1:])
			}
		}
	}
	return ""
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
			for lineEnd < close && !functionValueTokenEquals(program, lineEnd, ";") && program.Tokens[lineEnd].KindLine>>8 == program.Tokens[j].KindLine>>8 {
				lineEnd++
			}
			if functionValueTokenEquals(program, j, fieldName) && j+1 < lineEnd {
				return functionValueTokensText(program, j+1, lineEnd)
			}
			typeEnd := functionValueTypeEnd(program, j)
			if typeEnd == lineEnd {
				nameTok := j
				if functionValueTokenEquals(program, nameTok, "*") {
					nameTok++
				}
				if functionValueTokenEquals(program, nameTok+1, ".") && nameTok+2 < lineEnd {
					nameTok += 2
				}
				if functionValueTokenEquals(program, nameTok, fieldName) {
					return functionValueTokensText(program, j, lineEnd)
				}
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
			if j > start+2 && !functionValueTokenEquals(program, j-1, ";") && program.Tokens[j-1].KindLine>>8 == program.Tokens[j].KindLine>>8 {
				continue
			}
			embeddedStart := j
			if functionValueTokenEquals(program, j, "*") {
				embeddedStart++
			}
			if program.Tokens[embeddedStart].KindLine&255 != unit.TokenIdent {
				continue
			}
			embeddedEnd := embeddedStart + 1
			if functionValueTokenEquals(program, embeddedEnd, ".") && embeddedEnd+1 < close && program.Tokens[embeddedEnd+1].KindLine&255 == unit.TokenIdent {
				embeddedEnd += 2
			}
			if embeddedEnd < close && !functionValueTokenEquals(program, embeddedEnd, ";") && program.Tokens[embeddedEnd].KindLine>>8 == program.Tokens[embeddedStart].KindLine>>8 {
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
	return program.Tokens[tok].KindLine&255 == unit.TokenIdent || text == "*" || text == "[" || text == "struct" || text == "interface" || text == "map" || text == "func"
}

func functionValueTokenAtSpan(program *unit.Program, start int, end int) int {
	low := 0
	high := len(program.Tokens)
	for low < high {
		middle := low + (high-low)/2
		if program.Tokens[middle].Start < start {
			low = middle + 1
		} else {
			high = middle
		}
	}
	for low < len(program.Tokens) && program.Tokens[low].Start == start {
		if program.Tokens[low].Start+program.Tokens[low].Size == end {
			return low
		}
		low++
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
	nilable := len(result) > 0 && result[0] == '*'
	if len(result) > 0 && result[0] == '[' {
		i := 1
		for i < len(result) && functionValueIsSpace(result[i]) {
			i++
		}
		nilable = i < len(result) && result[i] == ']'
	}
	if nilable || functionValueHasPrefix(result, "map[") || functionValueHasPrefix(result, "func(") || functionValueHasPrefix(result, "interface{") {
		return "nil"
	}
	return "0"
}

func functionValueCanUseScalarZero(result string) bool {
	scalar := []string{"int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr", "byte", "rune", "float32", "float64", "complex64", "complex128"}
	for i := 0; i < len(scalar); i++ {
		if result == scalar[i] {
			return true
		}
	}
	return false
}

func functionValueHasPrefix(value string, prefix string) bool {
	i := 0
	for p := 0; p < len(prefix); p++ {
		for i < len(value) && functionValueIsSpace(value[i]) {
			i++
		}
		if i >= len(value) || value[i] != prefix[p] {
			return false
		}
		i++
	}
	return true
}

func functionValueIsSpace(value byte) bool {
	return value == ' ' || value == '\t' || value == '\n' || value == '\r'
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
