package link

import (
	"renvo.dev/internal/arena"
	"renvo.dev/internal/unit"
)

// Goroutines and channels are frontend features. This pass erases their
// syntax into the target-neutral x/runtime handler ABI before CoreProgram is
// serialized, keeping every compact backend on the ordinary source subset.

type concurrencyReceiveHelper struct {
	element string
	name    string
	open    bool
}

type concurrencyGoSite struct {
	entry       int
	context     string
	declaration string
	dispatch    string
}

func lowerConcurrencyCore(program *unit.Program, transient bool) bool {
	needed := len(program.ConcurrencySites) > 0
	if !needed {
		// Hand-built linker tests may not carry checked-package metadata. Normal
		// frontend builds use the compact table transported through RVFC2.
		for i := 0; i < len(program.Tokens); i++ {
			text := functionValueTokenText(program, i)
			if text == "chan" || text == "<-" || text == "go" || text == "select" {
				needed = true
				break
			}
		}
	}
	if !needed {
		return true
	}
	if !lowerChannelRangeCore(program, transient) {
		return false
	}
	if !lowerSelectSyntaxCore(program, transient) {
		return false
	}
	if !lowerChannelSyntaxCore(program, transient) {
		return false
	}
	if !lowerGoroutineSyntaxCore(program, transient) {
		return false
	}
	return true
}

func lowerChannelRangeCore(program *unit.Program, transient bool) bool {
	var edits []functionValueEdit
	count := 0
	for i := 0; i+3 < len(program.Tokens); i++ {
		if !functionValueTokenEquals(program, i, "for") {
			continue
		}
		open := concurrencyTopLevelToken(program, i+1, len(program.Tokens), "{")
		if open < 0 {
			return false
		}
		rangeTok := concurrencyTopLevelToken(program, i+1, open, "range")
		if rangeTok < 0 {
			continue
		}
		close := functionValueFindMatchingBrace(program, open)
		if close < 0 {
			return false
		}
		channelStart := rangeTok + 1
		channelType := ordinaryBuiltinExprType(program, rangeTok, channelStart, open)
		element := concurrencyChannelElement(channelType, program)
		if element == "" {
			continue
		}
		channelName := "__renvo_range_channel_" + functionValueDecimal(count)
		openName := "__renvo_range_open_" + functionValueDecimal(count)
		valueName := "__renvo_range_value_" + functionValueDecimal(count)
		channel := functionValueTokensText(program, channelStart, open)
		body := functionValueTokensText(program, open+1, close)
		prefix := valueName + ", " + openName + " := <-" + channelName + "; "
		lhs := functionValueTokensText(program, i+1, rangeTok)
		assign := concurrencyTopLevelAssignment(program, i+1, rangeTok)
		if lhs != "" {
			if assign < 0 {
				return false
			}
			name := functionValueTokensText(program, i+1, assign)
			operator := functionValueTokenText(program, assign)
			if concurrencyTextHasComma(name) {
				return false
			}
			if operator == ":=" {
				prefix = name + ", " + openName + " := <-" + channelName + "; "
			} else if operator == "=" {
				prefix += name + " = " + valueName + "; "
			} else {
				return false
			}
		}
		replacement := "for " + channelName + " := " + channel + "; ; { " + prefix + "if !" + openName + " { break }; " + body + " }"
		edits = append(edits, functionValueTokenRangeEdit(program, i, close+1, replacement))
		count++
		i = close
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
	return ok && reparseFunctionValueProgram(program, text, edits, originalLength, -1)
}

type concurrencySelectCase struct {
	index      int
	direction  string
	channel    string
	element    string
	send       string
	receiveLHS string
	receiveOp  string
	body       string
	isDefault  bool
}

func lowerSelectSyntaxCore(program *unit.Program, transient bool) bool {
	var edits []functionValueEdit
	generated := ""
	count := 0
	for i := 0; i+1 < len(program.Tokens); i++ {
		if !functionValueTokenEquals(program, i, "select") {
			continue
		}
		open := i + 1
		if !functionValueTokenEquals(program, open, "{") {
			return false
		}
		close := functionValueFindMatchingBrace(program, open)
		if close < 0 {
			return false
		}
		cases, ok := concurrencyParseSelectCases(program, open, close)
		if !ok {
			return false
		}
		name := "__renvo_select_" + functionValueDecimal(count)
		replacement, declaration := concurrencySelectText(name, cases)
		if replacement == "" || declaration == "" {
			return false
		}
		edits = append(edits, functionValueTokenRangeEdit(program, i, close+1, replacement))
		generated += declaration
		count++
		i = close
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

func concurrencyParseSelectCases(program *unit.Program, open int, close int) ([]concurrencySelectCase, bool) {
	var starts []int
	depth := 0
	for i := open + 1; i < close; i++ {
		text := functionValueTokenText(program, i)
		if text == "{" || text == "(" || text == "[" {
			depth++
		} else if text == "}" || text == ")" || text == "]" {
			depth--
		} else if depth == 0 && (text == "case" || text == "default") {
			starts = append(starts, i)
		}
	}
	var out []concurrencySelectCase
	operation := 0
	defaultSeen := false
	for i := 0; i < len(starts); i++ {
		start := starts[i]
		limit := close
		if i+1 < len(starts) {
			limit = starts[i+1]
		}
		colon := concurrencyTopLevelToken(program, start+1, limit, ":")
		if colon < 0 {
			return nil, false
		}
		item := concurrencySelectCase{index: operation, body: functionValueTokensText(program, colon+1, limit)}
		if functionValueTokenEquals(program, start, "default") {
			if defaultSeen {
				return nil, false
			}
			defaultSeen = true
			item.index = -1
			item.isDefault = true
			out = append(out, item)
			continue
		}
		arrow := concurrencyTopLevelToken(program, start+1, colon, "<-")
		if arrow < 0 {
			return nil, false
		}
		assign := concurrencyTopLevelAssignment(program, start+1, arrow)
		if arrow == start+1 || assign >= 0 {
			item.direction = "receive"
			item.channel = functionValueTokensText(program, arrow+1, colon)
			typ := ordinaryBuiltinExprType(program, arrow, arrow+1, colon)
			item.element = concurrencyChannelElement(typ, program)
			if item.element == "" {
				return nil, false
			}
			if assign >= 0 {
				item.receiveLHS = functionValueTokensText(program, start+1, assign)
				item.receiveOp = functionValueTokenText(program, assign)
			}
		} else {
			item.direction = "send"
			item.channel = functionValueTokensText(program, start+1, arrow)
			item.send = functionValueTokensText(program, arrow+1, colon)
			typ := ordinaryBuiltinExprType(program, arrow, start+1, arrow)
			item.element = concurrencyChannelElement(typ, program)
			if item.element == "" || item.send == "" {
				return nil, false
			}
		}
		operation++
		out = append(out, item)
	}
	return out, true
}

func concurrencySelectText(name string, cases []concurrencySelectCase) (string, string) {
	resultType := name + "_result"
	parameters := ""
	arguments := ""
	fields := "index int; "
	descriptors := ""
	switchCases := ""
	operation := 0
	hasDefault := false
	for i := 0; i < len(cases); i++ {
		item := cases[i]
		if item.isDefault {
			hasDefault = true
			switchCases += "case -1: " + item.body + "; "
			continue
		}
		if parameters != "" {
			parameters += ", "
			arguments += ", "
			descriptors += ", "
		}
		channelParam := "channel" + functionValueDecimal(operation)
		parameters += channelParam + " uintptr"
		arguments += "uintptr(" + item.channel + ")"
		valuePointer := "uintptr(&result.value" + functionValueDecimal(operation) + ")"
		okPointer := "uintptr(0)"
		direction := "renvo_runtime_SelectReceive"
		prefix := ""
		if item.direction == "send" {
			valueParam := "send" + functionValueDecimal(operation)
			parameters += ", " + valueParam + " " + item.element
			arguments += ", " + item.send
			valuePointer = "uintptr(&" + valueParam + ")"
			direction = "renvo_runtime_SelectSend"
		} else {
			fields += "value" + functionValueDecimal(operation) + " " + item.element + "; open" + functionValueDecimal(operation) + " bool; "
			okPointer = "uintptr(&result.open" + functionValueDecimal(operation) + ")"
			if item.receiveLHS != "" {
				prefix = item.receiveLHS + " " + item.receiveOp + " result.value" + functionValueDecimal(operation)
				// A two-target receive consumes the open result as its second value.
				if concurrencyTextHasComma(item.receiveLHS) {
					prefix += ", result.open" + functionValueDecimal(operation)
				}
				prefix += "; "
			}
		}
		descriptors += "renvo_runtime_ChanSelectValue{Channel: renvo_runtime_Channel(" + channelParam + "), Value: " + valuePointer + ", ReceiveOK: " + okPointer + ", Direction: " + direction + "}"
		switchCases += "case " + functionValueDecimal(item.index) + ": " + prefix + item.body + "; "
		operation++
	}
	boolText := "false"
	if hasDefault {
		boolText = "true"
	}
	declaration := "type " + resultType + " struct { " + fields + "}\n"
	declaration += "func " + name + "(" + parameters + ") " + resultType + " { var result " + resultType + "; cases := []renvo_runtime_ChanSelectValue{" + descriptors + "}; result.index = renvo_runtime_ChanSelect(cases, " + boolText + "); return result }\n"
	replacement := "switch result := " + name + "(" + arguments + "); result.index { " + switchCases + "}"
	return replacement, declaration
}

func concurrencyTopLevelToken(program *unit.Program, start int, end int, wanted string) int {
	paren, bracket, brace := 0, 0, 0
	for i := start; i < end; i++ {
		text := functionValueTokenText(program, i)
		if text == wanted && paren == 0 && bracket == 0 && brace == 0 {
			return i
		}
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
	return -1
}

func concurrencyTopLevelAssignment(program *unit.Program, start int, end int) int {
	for i := start; i < end; i++ {
		if functionValueTokenEquals(program, i, ":=") || functionValueTokenEquals(program, i, "=") {
			return i
		}
	}
	return -1
}

func concurrencyTextHasComma(text string) bool {
	for i := 0; i < len(text); i++ {
		if text[i] == ',' {
			return true
		}
	}
	return false
}

func lowerChannelSyntaxCore(program *unit.Program, transient bool) bool {
	covered := make([]bool, len(program.Tokens))
	var edits []functionValueEdit
	var helpers []concurrencyReceiveHelper

	// Construction must be discovered before channel type tokens are erased.
	for i := 0; i+2 < len(program.Tokens); i++ {
		if !functionValueTokenEquals(program, i, "make") || !functionValueTokenEquals(program, i+1, "(") {
			continue
		}
		close := functionValueFindMatchingParen(program, i+1)
		if close < 0 {
			return false
		}
		starts, ends := ordinaryBuiltinArguments(program, i+2, close)
		if len(starts) < 1 || len(starts) > 2 {
			continue
		}
		channelType := functionValueTokensText(program, starts[0], ends[0])
		typeEnd := concurrencyChanTypeEnd(program, starts[0])
		element := ""
		named := false
		if typeEnd == ends[0] {
			element = concurrencyChannelElementText(program, starts[0], ends[0])
		} else {
			element = concurrencyChannelElement(channelType, program)
			named = element != ""
		}
		if element == "" {
			continue
		}
		capacity := "0"
		if len(starts) == 2 {
			capacity = functionValueTokensText(program, starts[1], ends[1])
		}
		replacement := "renvo_runtime_ChanCreate(Sizeof(*new(" + element + ")), " + capacity + ")"
		if named {
			replacement = channelType + "(" + replacement + ")"
		}
		edits = append(edits, functionValueTokenRangeEdit(program, i, close+1, replacement))
		concurrencyCover(covered, i, close+1)
		i = close
	}

	// Sends are statements. Staging the value in an addressable temporary also
	// fixes evaluation order and lets the handler copy arbitrary element types.
	for i := 0; i < len(program.Tokens); i++ {
		if covered[i] || !functionValueTokenEquals(program, i, "<-") || functionValueTokenEquals(program, i+1, "chan") {
			continue
		}
		start := mapLowerAssignmentStart(program, i)
		end := mapLowerAssignmentEnd(program, i)
		if start >= i || i+1 >= end {
			continue
		}
		channelType := ordinaryBuiltinExprType(program, i, start, i)
		element := concurrencyChannelElement(channelType, program)
		if element == "" {
			continue
		}
		channel := functionValueTokensText(program, start, i)
		value := functionValueTokensText(program, i+1, end)
		name := "__renvo_chan_send_" + functionValueDecimal(i)
		channelName := name + "_channel"
		replacement := "{ " + channelName + " := " + channel + "; var " + name + " " + element + " = " + value + "; renvo_runtime_ChanSend(uintptr(" + channelName + "), uintptr(&" + name + ")) }"
		edits = append(edits, functionValueTokenRangeEdit(program, start, end, replacement))
		concurrencyCover(covered, start, end)
		i = end - 1
	}

	// Receives become typed helper calls. A single helper returns two values so
	// the same rewrite supports both ordinary and comma-ok receive forms.
	for i := 0; i < len(program.Tokens); i++ {
		if covered[i] || !functionValueTokenEquals(program, i, "<-") || functionValueTokenEquals(program, i+1, "chan") {
			continue
		}
		end := concurrencyPrimaryEnd(program, i+1)
		if end <= i+1 {
			continue
		}
		channelType := ordinaryBuiltinExprType(program, i, i+1, end)
		element := concurrencyChannelElement(channelType, program)
		if element == "" {
			continue
		}
		needsOpen := concurrencyReceiveNeedsOpen(program, i)
		helper := concurrencyReceiveHelperName(&helpers, element, needsOpen)
		channel := functionValueTokensText(program, i+1, end)
		edits = append(edits, functionValueTokenRangeEdit(program, i, end, helper+"(uintptr("+channel+"))"))
		concurrencyCover(covered, i, end)
		i = end - 1
	}

	// close, len, and cap retain their source evaluation but use the handler.
	for i := 0; i+1 < len(program.Tokens); i++ {
		if covered[i] || !functionValueTokenEquals(program, i+1, "(") {
			continue
		}
		name := functionValueTokenText(program, i)
		if name != "close" && name != "len" && name != "cap" {
			continue
		}
		if ordinaryBuiltinShadowed(program, i, name) {
			continue
		}
		close := functionValueFindMatchingParen(program, i+1)
		if close < 0 {
			return false
		}
		starts, ends := ordinaryBuiltinArguments(program, i+2, close)
		operandType := ""
		if len(starts) == 1 {
			operandType = ordinaryBuiltinExprType(program, i, starts[0], ends[0])
		}
		if len(starts) != 1 || concurrencyChannelElement(operandType, program) == "" {
			continue
		}
		replacement := "renvo_runtime_ChanClose"
		if name == "len" {
			replacement = "renvo_runtime_ChanLen"
		} else if name == "cap" {
			replacement = "renvo_runtime_ChanCap"
		}
		edits = append(edits, functionValueTokenEdit(program, i, replacement))
		covered[i] = true
	}

	// Channel nil comparisons use the opaque zero handle after type erasure.
	for i := 1; i+1 < len(program.Tokens); i++ {
		if covered[i] || (!functionValueTokenEquals(program, i, "==") && !functionValueTokenEquals(program, i, "!=")) {
			continue
		}
		if functionValueTokenEquals(program, i+1, "nil") {
			start := functionValuePrimaryStart(program, i-1)
			if start >= 0 && concurrencyChannelElement(ordinaryBuiltinExprType(program, i, start, i), program) != "" {
				edits = append(edits, functionValueTokenEdit(program, i+1, "0"))
				covered[i+1] = true
			}
		} else if functionValueTokenEquals(program, i-1, "nil") {
			end := concurrencyPrimaryEnd(program, i+1)
			if end > i+1 && concurrencyChannelElement(ordinaryBuiltinExprType(program, i, i+1, end), program) != "" {
				edits = append(edits, functionValueTokenEdit(program, i-1, "0"))
				covered[i-1] = true
			}
		}
	}

	// The backend sees channels as opaque word-sized handles.
	for i := 0; i < len(program.Tokens); i++ {
		if covered[i] {
			continue
		}
		end := concurrencyChanTypeEnd(program, i)
		if end <= i {
			continue
		}
		edits = append(edits, functionValueTokenRangeEdit(program, i, end, "uintptr"))
		concurrencyCover(covered, i, end)
		i = end - 1
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
	if !ok {
		return false
	}
	generatedStart := -1
	if len(helpers) > 0 {
		if len(text) > 0 && text[len(text)-1] != '\n' {
			text = append(text, '\n')
		}
		generatedStart = len(text)
		for i := 0; i < len(helpers); i++ {
			helper := helpers[i]
			declaration := "func " + helper.name + "(channel uintptr) " + helper.element + " { var value " + helper.element + "; renvo_runtime_ChanReceive(channel, uintptr(&value)); return value }\n"
			if helper.open {
				declaration = "func " + helper.name + "(channel uintptr) (" + helper.element + ", bool) { var value " + helper.element + "; open := renvo_runtime_ChanReceive(channel, uintptr(&value)); return value, open }\n"
			}
			text = appendFunctionValueString(text, declaration)
		}
	}
	if transient {
		arena.DiscardBytes(program.Text)
	}
	return reparseFunctionValueProgram(program, text, edits, originalLength, generatedStart)
}

func lowerGoroutineSyntaxCore(program *unit.Program, transient bool) bool {
	var edits []functionValueEdit
	var sites []concurrencyGoSite
	for i := 0; i < len(program.Tokens); i++ {
		if !functionValueTokenEquals(program, i, "go") {
			continue
		}
		end := mapLowerAssignmentEnd(program, i)
		if i+1 >= end {
			return false
		}
		context := "__renvo_go_context_" + functionValueDecimal(len(sites))
		spawn := "__renvo_go_spawn_" + functionValueDecimal(len(sites))
		entry := len(sites) + 1
		declaration, replacement, dispatch, ok := concurrencyDirectGoCall(program, i+1, end, context, spawn, entry)
		if !ok {
			return false
		}
		sites = append(sites, concurrencyGoSite{entry: entry, context: context, declaration: declaration, dispatch: dispatch})
		edits = append(edits, functionValueTokenRangeEdit(program, i, end, replacement))
		i = end - 1
	}
	if len(sites) == 0 {
		return true
	}
	callIndex := -1
	for i := 0; i < len(program.Funcs); i++ {
		if functionValueTokenText(program, program.Funcs[i].NameTok) == "renvo_runtime_Call" {
			callIndex = i
			break
		}
	}
	if callIndex < 0 {
		return false
	}
	call := program.Funcs[callIndex]
	dispatch := "{ switch int(entry) {"
	for i := 0; i < len(sites); i++ {
		dispatch += " case " + functionValueDecimal(sites[i].entry) + ": " + sites[i].dispatch + "; return;"
	}
	dispatch += " }; panic(\"runtime: invalid goroutine entry\") }"
	edits = append(edits, functionValueTokenRangeEdit(program, call.BodyStart, call.BodyEnd, dispatch))
	edits = appendFunctionValuePackageEdits(program, edits)
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
	for i := 0; i < len(sites); i++ {
		text = appendFunctionValueString(text, sites[i].declaration)
	}
	if transient {
		arena.DiscardBytes(program.Text)
	}
	return reparseFunctionValueProgram(program, text, edits, originalLength, generatedStart)
}

func concurrencyDirectGoCall(program *unit.Program, start int, end int, context string, spawn string, entry int) (string, string, string, bool) {
	if end-start < 3 || !functionValueTokenEquals(program, end-1, ")") {
		return "", "", "", false
	}
	open := functionValueFindMatchingBackward(program, end-1, "(", ")")
	if open <= start || open+1 > end-1 {
		return "", "", "", false
	}
	direct := open-start == 1 && program.Tokens[start].KindLine&255 == unit.TokenIdent &&
		(functionValueDeclaredFunction(program, functionValueTokenText(program, start)) || concurrencyDirectBuiltin(functionValueTokenText(program, start)))
	if !direct {
		if declaration, replacement, dispatch, ok := concurrencyGoLiteral(program, start, end, context, spawn, entry); ok {
			return declaration, replacement, dispatch, true
		}
		if callee := concurrencyInferredDirectFunction(program, start, open); callee != "" {
			return concurrencyNamedGoCall(program, callee, open, end, context, spawn, entry)
		}
		return concurrencyDynamicGoCall(program, start, open, end, context, spawn, entry)
	}
	return concurrencyNamedGoCall(program, functionValueTokenText(program, start), open, end, context, spawn, entry)
}

func concurrencyNamedGoCall(program *unit.Program, callee string, open int, end int, context string, spawn string, entry int) (string, string, string, bool) {
	starts, ends := ordinaryBuiltinArguments(program, open+1, end-1)
	parameterTypes := concurrencyFunctionParameterTypes(program, callee)
	declaration := "type " + context + " struct { "
	parameters := ""
	spawnArguments := ""
	setup := "__renvo_context_value := new(" + context + "); "
	arguments := ""
	for i := 0; i < len(starts); i++ {
		typ := ordinaryBuiltinExprType(program, open, starts[i], ends[i])
		if typ == "" && i < len(parameterTypes) {
			typ = parameterTypes[i]
		}
		if typ == "" {
			return "", "", "", false
		}
		field := "value" + functionValueDecimal(i)
		declaration += field + " " + typ + "; "
		if i > 0 {
			parameters += ", "
			spawnArguments += ", "
		}
		parameter := "argument" + functionValueDecimal(i)
		parameters += parameter + " " + typ
		spawnArguments += functionValueTokensText(program, starts[i], ends[i])
		setup += "__renvo_context_value." + field + " = " + parameter + "; "
		if i > 0 {
			arguments += ", "
		}
		arguments += "(*" + context + ")(context)." + field
	}
	declaration += "}\nfunc " + spawn + "(" + parameters + ") { " + setup + "renvo_runtime_Spawn(" + functionValueDecimal(entry) + ", uintptr(__renvo_context_value)) }\n"
	dispatch := callee + "(" + arguments + ")"
	return declaration, spawn + "(" + spawnArguments + ")", dispatch, true
}

func concurrencyInferredDirectFunction(program *unit.Program, start int, open int) string {
	if open-start != 1 || program.Tokens[start].KindLine&255 != unit.TokenIdent {
		return ""
	}
	name := functionValueTokenText(program, start)
	fnIndex := functionValueEnclosingFunc(program, start)
	if fnIndex < 0 {
		return ""
	}
	fn := program.Funcs[fnIndex]
	for i := start - 1; i > fn.BodyStart; i-- {
		if !functionValueTokenEquals(program, i, name) || !functionValueTokenEquals(program, i+1, ":=") || i+2 >= start {
			continue
		}
		candidate := functionValueTokenText(program, i+2)
		if program.Tokens[i+2].KindLine&255 == unit.TokenIdent && functionValueDeclaredFunction(program, candidate) {
			return candidate
		}
		return ""
	}
	return ""
}

func concurrencyDirectBuiltin(name string) bool {
	return name == "close" || name == "copy" || name == "delete" || name == "panic" || name == "print" || name == "println"
}

func concurrencyDynamicGoCall(program *unit.Program, start int, open int, end int, context string, spawn string, entry int) (string, string, string, bool) {
	callType := concurrencyCallType(program, start, open)
	if callType == "" {
		return "", "", "", false
	}
	starts, ends := ordinaryBuiltinArguments(program, open+1, end-1)
	parameterTypes := concurrencyCallParameterTypes(program, open)
	callName := callType
	declaration := ""
	if functionValueHasPrefix(callType, "func(") {
		callName = context + "_call_type"
		declaration = "type " + callName + " " + callType + "\n"
	}
	declaration += "type " + context + " struct { call " + callName + "; "
	parameters := "call " + callName
	spawnArguments := functionValueTokensText(program, start, open)
	setup := "value := new(" + context + "); value.call = call; "
	arguments := ""
	for i := 0; i < len(starts); i++ {
		typ := ordinaryBuiltinExprType(program, start, starts[i], ends[i])
		if typ == "" && i < len(parameterTypes) {
			typ = parameterTypes[i]
		}
		if typ == "" {
			return "", "", "", false
		}
		field := "value" + functionValueDecimal(i)
		parameter := "argument" + functionValueDecimal(i)
		declaration += field + " " + typ + "; "
		parameters += ", " + parameter + " " + typ
		spawnArguments += ", " + functionValueTokensText(program, starts[i], ends[i])
		setup += "value." + field + " = " + parameter + "; "
		if i > 0 {
			arguments += ", "
		}
		arguments += "value." + field
	}
	declaration += "}\nfunc " + spawn + "(" + parameters + ") { " + setup + "renvo_runtime_Spawn(" + functionValueDecimal(entry) + ", uintptr(value)) }\n"
	invoke := spawn + "_call"
	declaration += "func " + invoke + "(value *" + context + ") { value.call(" + arguments + ") }\n"
	dispatch := invoke + "((*" + context + ")(context))"
	return declaration, spawn + "(" + spawnArguments + ")", dispatch, true
}

func concurrencyCallParameterTypes(program *unit.Program, open int) []string {
	if fn, ok := functionValueCalledFunction(program, open); ok {
		return functionValueFunctionParamTypes(program, fn)
	}
	return nil
}

func concurrencyCallType(program *unit.Program, start int, open int) string {
	if functionValueTokenEquals(program, start, "func") {
		paramsClose := functionValueFindMatchingParen(program, start+1)
		bodyOpen := concurrencyTopLevelToken(program, paramsClose+1, open, "{")
		if paramsClose > start && bodyOpen > paramsClose {
			return functionValueTokensText(program, start, bodyOpen)
		}
	}
	if fn, ok := functionValueCalledFunction(program, open); ok {
		return concurrencyDeclaredCallType(program, fn)
	}
	typ := ordinaryBuiltinExprType(program, start, start, open)
	if functionValueHasPrefix(ordinaryUnderlyingType(program, typ, 0), "func(") {
		return typ
	}
	if open-start == 1 && program.Tokens[start].KindLine&255 == unit.TokenIdent {
		if typ := concurrencyInferredFunctionValueType(program, start, functionValueTokenText(program, start)); typ != "" {
			return typ
		}
	}
	return ""
}

func concurrencyDeclaredCallType(program *unit.Program, fn unit.Func) string {
	params := functionValueFunctionParamTypes(program, fn)
	text := "func("
	for i := 0; i < len(params); i++ {
		if i > 0 {
			text += ", "
		}
		text += params[i]
	}
	text += ")"
	close := functionValueFindMatchingParen(program, fn.NameTok+1)
	if close+1 < fn.BodyStart {
		result := functionValueTokensText(program, close+1, fn.BodyStart)
		if result != "" {
			text += " " + result
		}
	}
	return text
}

func concurrencyInferredFunctionValueType(program *unit.Program, before int, name string) string {
	fnIndex := functionValueEnclosingFunc(program, before)
	if fnIndex < 0 {
		return ""
	}
	fn := program.Funcs[fnIndex]
	for i := before - 1; i > fn.BodyStart; i-- {
		if !functionValueTokenEquals(program, i, name) || !functionValueTokenEquals(program, i+1, ":=") || i+2 >= before {
			continue
		}
		rhs := i + 2
		if rhs+2 < before && program.Tokens[rhs].KindLine&255 == unit.TokenIdent && functionValueTokenEquals(program, rhs+1, ".") && program.Tokens[rhs+2].KindLine&255 == unit.TokenIdent {
			method := functionValueTokenText(program, rhs+2)
			baseType := functionValueEnclosingLocalType(program, i, functionValueTokenText(program, rhs))
			for index := 0; index < len(program.Funcs); index++ {
				candidate := program.Funcs[index]
				if candidate.ReceiverStart < candidate.ReceiverEnd && functionValueTokenText(program, candidate.NameTok) == method && (baseType == "" || functionValueTypeEmbeds(program, baseType, functionValueReceiverType(program, candidate), 0)) {
					return concurrencyDeclaredCallType(program, candidate)
				}
			}
			return ""
		}
		if program.Tokens[rhs].KindLine&255 != unit.TokenIdent {
			return ""
		}
		callee := functionValueTokenText(program, rhs)
		for index := 0; index < len(program.Funcs); index++ {
			candidate := program.Funcs[index]
			if candidate.ReceiverStart >= candidate.ReceiverEnd && functionValueTokenText(program, candidate.NameTok) == callee {
				return concurrencyDeclaredCallType(program, candidate)
			}
		}
		return ""
	}
	return ""
}

func concurrencyGoLiteral(program *unit.Program, start int, end int, context string, spawn string, entry int) (string, string, string, bool) {
	if !functionValueTokenEquals(program, start, "func") || !functionValueTokenEquals(program, start+1, "(") {
		return "", "", "", false
	}
	paramsClose := functionValueFindMatchingParen(program, start+1)
	if paramsClose < 0 || paramsClose+1 >= end || !functionValueTokenEquals(program, paramsClose+1, "{") {
		return "", "", "", false
	}
	bodyOpen := paramsClose + 1
	bodyClose := functionValueFindMatchingBrace(program, bodyOpen)
	callOpen := bodyClose + 1
	if bodyClose < 0 || callOpen >= end || !functionValueTokenEquals(program, callOpen, "(") || functionValueFindMatchingParen(program, callOpen) != end-1 {
		return "", "", "", false
	}
	_, paramNames, ok := normalizedFunctionValueParams(program, start+2, paramsClose)
	if !ok {
		return "", "", "", false
	}
	paramTypes := concurrencyLiteralParameterTypes(program, start+2, paramsClose)
	argStarts, argEnds := ordinaryBuiltinArguments(program, callOpen+1, end-1)
	if len(paramNames) != len(paramTypes) || len(paramNames) != len(argStarts) {
		return "", "", "", false
	}
	captures, captureTypes := functionValueCaptures(program, start, bodyOpen, bodyClose, paramNames)
	if len(captures) != len(captureTypes) {
		return "", "", "", false
	}
	declaration := "type " + context + " struct { "
	parameters := ""
	arguments := ""
	setup := "__renvo_context_value := new(" + context + "); "
	for i := 0; i < len(captures); i++ {
		field := captures[i]
		declaration += field + " *" + captureTypes[i] + "; "
		if i > 0 {
			parameters += ", "
			arguments += ", "
		}
		parameters += field + " *" + captureTypes[i]
		arguments += "&" + field
		setup += "__renvo_context_value." + field + " = " + field + "; "
	}
	for i := 0; i < len(paramNames); i++ {
		field := paramNames[i]
		if parameters != "" {
			parameters += ", "
			arguments += ", "
		}
		declaration += field + " " + paramTypes[i] + "; "
		parameters += field + " " + paramTypes[i]
		arguments += functionValueTokensText(program, argStarts[i], argEnds[i])
		setup += "__renvo_context_value." + field + " = " + field + "; "
	}
	declaration += "}\n"
	declaration += "func " + spawn + "(" + parameters + ") { " + setup + "renvo_runtime_Spawn(" + functionValueDecimal(entry) + ", uintptr(__renvo_context_value)) }\n"
	call := spawn + "_call"
	body := concurrencyClosureBody(program, bodyOpen, bodyClose, captures, paramNames)
	declaration += "func " + call + "(env *" + context + ") { " + body + " }\n"
	replacement := spawn + "(" + arguments + ")"
	if len(captures) > 0 {
		marker := "{ if false { func(){ "
		for i := 0; i < len(captures); i++ {
			marker += "_ = " + captures[i] + "; "
		}
		marker += "}() }; "
		replacement = marker + replacement + " }"
	}
	return declaration, replacement, call + "((*" + context + ")(context))", true
}

func concurrencyClosureBody(program *unit.Program, bodyOpen int, bodyClose int, captures []string, params []string) string {
	if bodyOpen+1 >= bodyClose {
		return ""
	}
	start := program.Tokens[bodyOpen].Start + program.Tokens[bodyOpen].Size
	end := program.Tokens[bodyClose].Start
	src := program.Text[start:end]
	var edits []functionValueEdit
	for i := bodyOpen + 1; i < bodyClose; i++ {
		if !functionValueClosureValueReference(program, i) {
			continue
		}
		name := functionValueTokenText(program, i)
		replacement := ""
		if functionValueNameInList(captures, name) {
			replacement = "(*env." + name + ")"
		} else if functionValueNameInList(params, name) {
			replacement = "env." + name
		}
		if replacement == "" {
			continue
		}
		tok := program.Tokens[i]
		edits = append(edits, functionValueEdit{start: tok.Start - start, end: tok.Start + tok.Size - start, text: replacement})
	}
	out, ok := applyFunctionValueEdits(src, edits)
	if !ok {
		return ""
	}
	return string(out)
}

func concurrencyLiteralParameterTypes(program *unit.Program, start int, end int) []string {
	partStarts, partEnds := functionValueCommaParts(program, start, end)
	var out []string
	for i := 0; i < len(partStarts); i++ {
		partLen := partEnds[i] - partStarts[i]
		typ := functionValueTokensText(program, partStarts[i], partEnds[i])
		if partLen >= 2 && program.Tokens[partStarts[i]].KindLine&255 == unit.TokenIdent {
			typ = functionValueTokensText(program, partStarts[i]+1, partEnds[i])
		} else if partLen == 1 && i+1 < len(partStarts) && partEnds[i+1]-partStarts[i+1] >= 2 {
			typ = functionValueTokensText(program, partStarts[i+1]+1, partEnds[i+1])
		}
		out = append(out, typ)
	}
	return out
}

func concurrencyFunctionParameterTypes(program *unit.Program, name string) []string {
	for i := 0; i < len(program.Funcs); i++ {
		fn := program.Funcs[i]
		if fn.ReceiverStart >= fn.ReceiverEnd && functionValueTokenText(program, fn.NameTok) == name {
			return functionValueFunctionParamTypes(program, fn)
		}
	}
	return nil
}

func concurrencyChanTypeEnd(program *unit.Program, start int) int {
	if functionValueTokenEquals(program, start, "<-") {
		if !functionValueTokenEquals(program, start+1, "chan") {
			return start
		}
		start++
	}
	if !functionValueTokenEquals(program, start, "chan") {
		return start
	}
	element := start + 1
	if functionValueTokenEquals(program, element, "<-") {
		element++
	}
	end := functionValueTypeEnd(program, element)
	if end <= element {
		return start
	}
	return end
}

func concurrencyChannelElementText(program *unit.Program, start int, end int) string {
	if functionValueTokenEquals(program, start, "<-") {
		start++
	}
	if !functionValueTokenEquals(program, start, "chan") {
		return ""
	}
	start++
	if functionValueTokenEquals(program, start, "<-") {
		start++
	}
	return functionValueTokensText(program, start, end)
}

func concurrencyChannelElement(typ string, program *unit.Program) string {
	underlying := ordinaryUnderlyingType(program, typ, 0)
	start := 0
	for start < len(underlying) && functionValueIsSpace(underlying[start]) {
		start++
	}
	if start+2 <= len(underlying) && underlying[start:start+2] == "<-" {
		start += 2
		for start < len(underlying) && functionValueIsSpace(underlying[start]) {
			start++
		}
	}
	if start+4 > len(underlying) || underlying[start:start+4] != "chan" {
		return ""
	}
	start += 4
	for start < len(underlying) && functionValueIsSpace(underlying[start]) {
		start++
	}
	if start+2 <= len(underlying) && underlying[start:start+2] == "<-" {
		start += 2
	}
	for start < len(underlying) && functionValueIsSpace(underlying[start]) {
		start++
	}
	end := len(underlying)
	for end > start && functionValueIsSpace(underlying[end-1]) {
		end--
	}
	if end <= start {
		return ""
	}
	return underlying[start:end]
}

func concurrencyPrimaryEnd(program *unit.Program, start int) int {
	if start < 0 || start >= len(program.Tokens) {
		return start
	}
	end := start + 1
	if functionValueTokenEquals(program, start, "(") {
		close := functionValueFindMatchingParen(program, start)
		if close < 0 {
			return start
		}
		end = close + 1
	}
	for end < len(program.Tokens) {
		if functionValueTokenEquals(program, end, ".") && end+1 < len(program.Tokens) {
			end += 2
			continue
		}
		if functionValueTokenEquals(program, end, "[") {
			close := functionValueFindMatching(program, end, "[", "]")
			if close < 0 {
				return start
			}
			end = close + 1
			continue
		}
		if functionValueTokenEquals(program, end, "(") {
			close := functionValueFindMatchingParen(program, end)
			if close < 0 {
				return start
			}
			end = close + 1
			continue
		}
		break
	}
	return end
}

func concurrencyReceiveNeedsOpen(program *unit.Program, arrow int) bool {
	start := mapLowerAssignmentStart(program, arrow)
	assign := concurrencyTopLevelAssignment(program, start, arrow)
	if assign < 0 {
		return false
	}
	for i := start; i < assign; i++ {
		if functionValueTokenEquals(program, i, ",") {
			return true
		}
	}
	return false
}

func concurrencyReceiveHelperName(helpers *[]concurrencyReceiveHelper, element string, open bool) string {
	for i := 0; i < len(*helpers); i++ {
		if (*helpers)[i].element == element && (*helpers)[i].open == open {
			return (*helpers)[i].name
		}
	}
	name := "__renvo_chan_receive_" + functionValueDecimal(len(*helpers))
	*helpers = append(*helpers, concurrencyReceiveHelper{element: element, name: name, open: open})
	return name
}

func concurrencyCover(covered []bool, start int, end int) {
	if start < 0 {
		start = 0
	}
	if end > len(covered) {
		end = len(covered)
	}
	for i := start; i < end; i++ {
		covered[i] = true
	}
}
