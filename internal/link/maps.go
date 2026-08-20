package link

import (
	"renvo.dev/internal/arena"
	"renvo.dev/internal/unit"
)

// Maps are a frontend language feature. This pass replaces them with typed
// pointer-owned entry slices before the unit reaches the size-constrained
// backend. Unlike the historical simple-map rewrite, discovery is based on
// parsed type and expression boundaries and each operation preserves map
// identity and operand evaluation.

type mapLowerSpec struct {
	typ       string
	key       string
	value     string
	storage   string
	entry     string
	find      string
	get       string
	lookup    string
	ref       string
	set       string
	remove    string
	length    string
	makeValue string
	entries   string
}

type mapLowerLiteral struct {
	spec   int
	name   string
	result string
	keys   []string
	values []string
}

type mapLowerMake struct {
	spec   int
	name   string
	result string
}

func lowerMapsCore(program *unit.Program, transient bool) bool {
	specs := discoverMapLowerSpecs(program)
	if len(specs) == 0 {
		return true
	}
	edits, literals, makes, ok := mapLowerEdits(program, specs)
	if !ok {
		return false
	}
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
	text = appendFunctionValueString(text, mapLowerGeneratedText(specs, literals, makes))
	if transient {
		arena.DiscardBytes(program.Text)
	}
	return reparseFunctionValueProgram(program, text, edits, originalLength, generatedStart)
}

func discoverMapLowerSpecs(program *unit.Program) []mapLowerSpec {
	var specs []mapLowerSpec
	for i := 0; i+3 < len(program.Tokens); i++ {
		if !functionValueTokenEquals(program, i, "map") || !functionValueTokenEquals(program, i+1, "[") {
			continue
		}
		close := functionValueFindMatching(program, i+1, "[", "]")
		end := functionValueTypeEnd(program, i)
		if close <= i+1 || end <= close+1 {
			continue
		}
		typ := compactMapLowerType(functionValueTokensText(program, i, end))
		if mapLowerSpecIndex(specs, typ) >= 0 {
			continue
		}
		index := len(specs)
		prefix := "__renvo_map_" + functionValueDecimal(index)
		specs = append(specs, mapLowerSpec{
			typ:       typ,
			key:       functionValueTokensText(program, i+2, close),
			value:     functionValueTokensText(program, close+1, end),
			storage:   prefix,
			entry:     prefix + "_entry",
			find:      prefix + "_find",
			get:       prefix + "_get",
			lookup:    prefix + "_lookup",
			ref:       prefix + "_ref",
			set:       prefix + "_set",
			remove:    prefix + "_delete",
			length:    prefix + "_len",
			makeValue: prefix + "_make",
			entries:   prefix + "_entries",
		})
	}
	return specs
}

func compactMapLowerType(value string) string {
	out := ""
	for i := 0; i < len(value); i++ {
		if !functionValueIsSpace(value[i]) {
			out += value[i : i+1]
		}
	}
	return out
}

func mapLowerSpecIndex(specs []mapLowerSpec, typ string) int {
	typ = compactMapLowerType(typ)
	for i := 0; i < len(specs); i++ {
		if specs[i].typ == typ {
			return i
		}
	}
	return -1
}

func mapLowerExprSpec(program *unit.Program, specs []mapLowerSpec, before int, start int, end int) int {
	typ := ordinaryBuiltinExprType(program, before, start, end)
	underlying := ordinaryUnderlyingType(program, typ, 0)
	if spec := mapLowerSpecIndex(specs, underlying); spec >= 0 {
		return spec
	}
	if end-start == 1 && program.Tokens[start].KindLine&255 == unit.TokenIdent {
		name := functionValueTokenText(program, start)
		typ = mapLowerShortLocalType(program, before, name)
		underlying = ordinaryUnderlyingType(program, typ, 0)
		if spec := mapLowerSpecIndex(specs, underlying); spec >= 0 {
			return spec
		}
		typ = mapLowerTypeSwitchLocalType(program, before, name)
	}
	underlying = ordinaryUnderlyingType(program, typ, 0)
	return mapLowerSpecIndex(specs, underlying)
}

// A variable introduced by a type switch has the selected case's type inside
// a single-type case. Core local metadata deliberately keeps only authored
// declarations, so recover that narrow fact from the parsed token structure
// for lowering passes which run before the backend sees the program.
func mapLowerTypeSwitchLocalType(program *unit.Program, before int, name string) string {
	caseTok := -1
	closed := 0
	for i := before - 1; i >= 0; i-- {
		text := functionValueTokenText(program, i)
		if text == "}" {
			closed++
			continue
		}
		if text == "{" && closed > 0 {
			closed--
			continue
		}
		if closed == 0 && (text == "case" || text == "default") {
			caseTok = i
			break
		}
	}
	if caseTok < 0 || functionValueTokenEquals(program, caseTok, "default") {
		return ""
	}
	colon := mapLowerTopLevelToken(program, caseTok+1, before, ":")
	if colon <= caseTok+1 || mapLowerTopLevelToken(program, caseTok+1, colon, ",") >= 0 {
		return ""
	}

	open := -1
	closed = 0
	for i := caseTok - 1; i >= 0; i-- {
		text := functionValueTokenText(program, i)
		if text == "}" {
			closed++
		} else if text == "{" {
			if closed > 0 {
				closed--
			} else {
				open = i
				break
			}
		}
	}
	if open < 0 {
		return ""
	}
	switchTok := open - 1
	for switchTok >= 0 && !functionValueTokenEquals(program, switchTok, "switch") && !functionValueTokenEquals(program, switchTok, "{") && !functionValueTokenEquals(program, switchTok, "}") && !functionValueTokenEquals(program, switchTok, ";") {
		switchTok--
	}
	if switchTok < 0 || !functionValueTokenEquals(program, switchTok, "switch") {
		return ""
	}
	define := mapLowerTopLevelToken(program, switchTok+1, open, ":=")
	if define < 0 || define != switchTok+2 || functionValueTokenText(program, switchTok+1) != name {
		return ""
	}
	typeTok := -1
	for i := define + 1; i < open; i++ {
		if functionValueTokenEquals(program, i, "type") {
			typeTok = i
			break
		}
	}
	if typeTok < define+3 || typeTok+1 >= open ||
		!functionValueTokenEquals(program, typeTok-1, "(") ||
		!functionValueTokenEquals(program, typeTok-2, ".") ||
		!functionValueTokenEquals(program, typeTok+1, ")") {
		return ""
	}
	return functionValueTokensText(program, caseTok+1, colon)
}

func mapLowerShortLocalType(program *unit.Program, before int, name string) string {
	fnIndex := functionValueEnclosingFunc(program, before)
	if fnIndex < 0 {
		return ""
	}
	fn := program.Funcs[fnIndex]
	for assign := before - 1; assign > fn.BodyStart; assign-- {
		if !functionValueTokenEquals(program, assign, ":=") {
			continue
		}
		line := program.Tokens[assign].KindLine >> 8
		start := assign - 1
		for start > fn.BodyStart && program.Tokens[start-1].KindLine>>8 == line && !functionValueTokenEquals(program, start-1, ";") && !functionValueTokenEquals(program, start-1, "{") && !functionValueTokenEquals(program, start-1, "}") {
			start--
		}
		end := assign + 1
		// Braces on the right hand side usually belong to composite literals.
		// The line boundary is the statement boundary here; stopping at the
		// first literal brace loses every item in a multi-value declaration.
		for end < fn.BodyEnd && program.Tokens[end].KindLine>>8 == line && !functionValueTokenEquals(program, end, ";") {
			end++
		}
		leftStarts, leftEnds := ordinaryBuiltinArguments(program, start, assign)
		rightStarts, rightEnds := ordinaryBuiltinArguments(program, assign+1, end)
		for item := 0; item < len(leftStarts) && item < len(rightStarts); item++ {
			if leftEnds[item]-leftStarts[item] != 1 || functionValueTokenText(program, leftStarts[item]) != name {
				continue
			}
			rhsStart := rightStarts[item]
			rhsEnd := rightEnds[item]
			typeEnd := functionValueTypeEnd(program, rhsStart)
			if typeEnd > rhsStart && typeEnd < rhsEnd && functionValueTokenEquals(program, typeEnd, "{") {
				return functionValueTokensText(program, rhsStart, typeEnd)
			}
			if typ := ordinaryBuiltinExprType(program, assign, rhsStart, rhsEnd); typ != "" {
				return typ
			}
		}
	}
	return ""
}

func mapLowerEdits(program *unit.Program, specs []mapLowerSpec) ([]functionValueEdit, []mapLowerLiteral, []mapLowerMake, bool) {
	var edits []functionValueEdit
	edits = appendFunctionValuePackageEdits(program, edits)
	var literals []mapLowerLiteral
	var makes []mapLowerMake
	covered := make([]bool, len(program.Tokens))
	var ok bool
	edits, covered = mapLowerAssignmentEdits(program, specs, edits, covered, &literals)
	edits, covered = mapLowerRangeEdits(program, specs, edits, covered)
	edits, covered, ok = mapLowerConstructionEdits(program, specs, edits, covered, &literals, &makes)
	if !ok {
		return nil, nil, nil, false
	}
	edits, covered = mapLowerBuiltinEdits(program, specs, edits, covered)
	edits, covered, ok = mapLowerIndexEdits(program, specs, edits, covered)
	if !ok {
		return nil, nil, nil, false
	}
	for i := 0; i < len(program.Tokens); i++ {
		if covered[i] || !functionValueTokenEquals(program, i, "map") || !functionValueTokenEquals(program, i+1, "[") {
			continue
		}
		end := functionValueTypeEnd(program, i)
		if end <= i {
			return nil, nil, nil, false
		}
		spec := mapLowerSpecIndex(specs, functionValueTokensText(program, i, end))
		if spec < 0 {
			return nil, nil, nil, false
		}
		edits = append(edits, functionValueTokenRangeEdit(program, i, end, mapLowerRepresentation(specs[spec])))
		mapLowerCover(covered, i, end)
		i = end - 1
	}
	return edits, literals, makes, true
}

func mapLowerAssignmentEdits(program *unit.Program, specs []mapLowerSpec, edits []functionValueEdit, covered []bool, literals *[]mapLowerLiteral) ([]functionValueEdit, []bool) {
	for assign := 0; assign < len(program.Tokens); assign++ {
		operator := functionValueTokenText(program, assign)
		compound := mapLowerCompoundOperator(operator)
		if covered[assign] || operator != "=" && compound == "" {
			continue
		}
		start := mapLowerAssignmentStart(program, assign)
		end := mapLowerAssignmentEnd(program, assign)
		leftStarts, leftEnds := ordinaryBuiltinArguments(program, start, assign)
		rightStarts, rightEnds := ordinaryBuiltinArguments(program, assign+1, end)
		if len(leftStarts) == 0 || len(rightStarts) == 0 || len(rightStarts) != len(leftStarts) && len(rightStarts) != 1 {
			continue
		}
		targetSpecs := make([]int, len(leftStarts))
		targetOpens := make([]int, len(leftStarts))
		anyMap := false
		for i := 0; i < len(leftStarts); i++ {
			targetSpecs[i], targetOpens[i] = mapLowerAssignmentTarget(program, specs, leftStarts[i], leftEnds[i])
			if targetSpecs[i] >= 0 {
				anyMap = true
			}
		}
		if !anyMap {
			continue
		}
		name := functionValueDecimal(assign)
		if operator != "=" {
			if len(leftStarts) != 1 || len(rightStarts) != 1 || targetSpecs[0] < 0 {
				continue
			}
			open := targetOpens[0]
			base := mapLowerReadText(program, specs, leftStarts[0], open)
			key := mapLowerReadText(program, specs, open+1, leftEnds[0]-1)
			rhs, ok := mapLowerLiteralValue(program, specs, rightStarts[0], rightEnds[0], specs[targetSpecs[0]].value, literals)
			if !ok {
				continue
			}
			target := "__renvo_map_target_" + name
			keyName := "__renvo_map_key_" + name
			old := "__renvo_map_old_" + name
			value := "__renvo_map_value_" + name
			valueType := specs[targetSpecs[0]].value
			replacement := "if true {" + target + " := " + base + ";" + keyName + " := " + key + ";" + old + " := " + specs[targetSpecs[0]].get + "(" + target + "," + keyName + ");var " + value + " " + valueType + " = " + rhs + ";" + specs[targetSpecs[0]].set + "(" + target + "," + keyName + "," + old + compound + value + ");}"
			edits = append(edits, functionValueTokenRangeEdit(program, start, end, replacement))
			mapLowerCover(covered, start, end)
			assign = end - 1
			continue
		}
		replacement := "if true {"
		for i := 0; i < len(leftStarts); i++ {
			if targetSpecs[i] >= 0 {
				open := targetOpens[i]
				base := mapLowerReadText(program, specs, leftStarts[i], open)
				key := mapLowerReadText(program, specs, open+1, leftEnds[i]-1)
				replacement += "__renvo_map_target_" + name + "_" + functionValueDecimal(i) + " := " + base + ";"
				replacement += "__renvo_map_key_" + name + "_" + functionValueDecimal(i) + " := " + key + ";"
			} else if compactMapLowerType(functionValueTokensText(program, leftStarts[i], leftEnds[i])) != "_" {
				replacement += "__renvo_map_address_" + name + "_" + functionValueDecimal(i) + " := &(" + functionValueTokensText(program, leftStarts[i], leftEnds[i]) + ");"
			}
		}
		for i := 0; i < len(leftStarts); i++ {
			if i > 0 {
				replacement += ","
			}
			replacement += "__renvo_map_value_" + name + "_" + functionValueDecimal(i)
		}
		replacement += " := "
		for i := 0; i < len(rightStarts); i++ {
			if i > 0 {
				replacement += ","
			}
			expected := ""
			if i < len(targetSpecs) && targetSpecs[i] >= 0 {
				expected = specs[targetSpecs[i]].value
			}
			value, ok := mapLowerLiteralValue(program, specs, rightStarts[i], rightEnds[i], expected, literals)
			if !ok {
				value = mapLowerReadText(program, specs, rightStarts[i], rightEnds[i])
			}
			sourceType := ordinaryBuiltinExprType(program, assign, rightStarts[i], rightEnds[i])
			if expected != "" && sourceType != "" && len(rightStarts) == len(leftStarts) && compactMapLowerType(sourceType) != compactMapLowerType(expected) {
				value = expected + "(" + value + ")"
			}
			replacement += value
		}
		replacement += ";"
		for i := 0; i < len(leftStarts); i++ {
			value := "__renvo_map_value_" + name + "_" + functionValueDecimal(i)
			if targetSpecs[i] >= 0 {
				replacement += specs[targetSpecs[i]].set + "(__renvo_map_target_" + name + "_" + functionValueDecimal(i) + ",__renvo_map_key_" + name + "_" + functionValueDecimal(i) + "," + value + ");"
			} else if compactMapLowerType(functionValueTokensText(program, leftStarts[i], leftEnds[i])) == "_" {
				replacement += "_ = " + value + ";"
			} else {
				replacement += "*__renvo_map_address_" + name + "_" + functionValueDecimal(i) + " = " + value + ";"
			}
		}
		replacement += "}"
		edits = append(edits, functionValueTokenRangeEdit(program, start, end, replacement))
		mapLowerCover(covered, start, end)
		assign = end - 1
	}
	return edits, covered
}

func mapLowerAssignmentStart(program *unit.Program, assign int) int {
	start := assign
	depth := 0
	for i := assign - 1; i >= 0; i-- {
		text := functionValueTokenText(program, i)
		if depth == 0 && (text == ";" || text == "{" || text == "}") {
			break
		}
		start = i
		if text == ")" || text == "]" || text == "}" {
			depth++
		} else if text == "(" || text == "[" || text == "{" {
			depth--
		}
		if i > 0 && program.Tokens[i-1].KindLine>>8 != program.Tokens[i].KindLine>>8 && depth == 0 && !functionValueTokenEquals(program, i-1, ",") {
			break
		}
	}
	return start
}

func mapLowerAssignmentEnd(program *unit.Program, assign int) int {
	end := assign + 1
	depth := 0
	for end < len(program.Tokens) {
		text := functionValueTokenText(program, end)
		if depth == 0 && (text == ";" || text == "}") {
			break
		}
		if text == "(" || text == "[" || text == "{" {
			depth++
		} else if text == ")" || text == "]" || text == "}" {
			depth--
		}
		end++
		if end < len(program.Tokens) && program.Tokens[end-1].KindLine>>8 != program.Tokens[end].KindLine>>8 && depth == 0 && !mapLowerLineContinues(functionValueTokenText(program, end-1)) {
			break
		}
	}
	return end
}

func mapLowerLineContinues(text string) bool {
	return text == "," || text == "+" || text == "-" || text == "*" || text == "/" || text == "%" || text == "&" || text == "|" || text == "^" || text == "&^" || text == "<<" || text == ">>" || text == "&&" || text == "||"
}

func mapLowerCompoundOperator(operator string) string {
	if operator == "+=" || operator == "-=" || operator == "*=" || operator == "/=" || operator == "%=" || operator == "&=" || operator == "|=" || operator == "^=" || operator == "<<=" || operator == ">>=" {
		return operator[:len(operator)-1]
	}
	if operator == "&^=" {
		return "&^"
	}
	return ""
}

func mapLowerAssignmentTarget(program *unit.Program, specs []mapLowerSpec, start int, end int) (int, int) {
	if end-start < 4 || !functionValueTokenEquals(program, end-1, "]") {
		return -1, -1
	}
	open := functionValueFindMatchingBackward(program, end-1, "[", "]")
	if open <= start || functionValuePrimaryStart(program, open-1) != start {
		return -1, -1
	}
	return mapLowerExprSpec(program, specs, open, start, open), open
}

func mapLowerRangeEdits(program *unit.Program, specs []mapLowerSpec, edits []functionValueEdit, covered []bool) ([]functionValueEdit, []bool) {
	for rangeTok := 0; rangeTok < len(program.Tokens); rangeTok++ {
		if !functionValueTokenEquals(program, rangeTok, "range") || covered[rangeTok] {
			continue
		}
		forTok := rangeTok - 1
		for forTok >= 0 && !functionValueTokenEquals(program, forTok, "for") && !functionValueTokenEquals(program, forTok, "{") && !functionValueTokenEquals(program, forTok, "}") && !functionValueTokenEquals(program, forTok, ";") {
			forTok--
		}
		if forTok < 0 || !functionValueTokenEquals(program, forTok, "for") {
			continue
		}
		bodyOpen := rangeTok + 1
		for bodyOpen < len(program.Tokens) && !functionValueTokenEquals(program, bodyOpen, "{") {
			bodyOpen++
		}
		if bodyOpen >= len(program.Tokens) {
			continue
		}
		spec := mapLowerExprSpec(program, specs, rangeTok, rangeTok+1, bodyOpen)
		if spec < 0 {
			continue
		}
		bodyClose := functionValueFindMatchingBrace(program, bodyOpen)
		if bodyClose < 0 {
			continue
		}
		expr := mapLowerReadText(program, specs, rangeTok+1, bodyOpen)
		mappingName := "__renvo_map_range_mapping_" + functionValueDecimal(rangeTok)
		mapping := "(" + expr + ")"
		indexName := "__renvo_map_range_index_" + functionValueDecimal(rangeTok)
		entryName := "__renvo_map_range_entry_" + functionValueDecimal(rangeTok)
		replacement := "{ " + mappingName + " := " + mapping + "; for " + indexName + "," + entryName + " := range " + specs[spec].entries + "(" + mappingName + ") { _ = " + indexName + "; if " + specs[spec].find + "(" + mappingName + "," + entryName + ".key) < 0 { continue };"
		op := mapLowerTopLevelToken(program, forTok+1, rangeTok, ":=")
		define := true
		if op < 0 {
			op = mapLowerTopLevelToken(program, forTok+1, rangeTok, "=")
			define = false
		}
		if op >= 0 {
			comma := mapLowerTopLevelToken(program, forTok+1, op, ",")
			firstEnd := op
			if comma >= 0 {
				firstEnd = comma
			}
			first := functionValueTokensText(program, forTok+1, firstEnd)
			second := ""
			if comma >= 0 {
				second = functionValueTokensText(program, comma+1, op)
			}
			assign := " = "
			if define {
				assign = " := "
			}
			if compactMapLowerType(first) != "_" {
				replacement += first + assign + entryName + ".key;"
			}
			if second != "" && compactMapLowerType(second) != "_" {
				replacement += second + assign + specs[spec].get + "(" + mappingName + "," + entryName + ".key);"
			}
		}
		edits = append(edits, functionValueTokenRangeEdit(program, forTok, bodyOpen+1, replacement))
		closeToken := program.Tokens[bodyClose]
		closeEnd := closeToken.Start + closeToken.Size
		edits = append(edits, functionValueEdit{start: closeEnd, end: closeEnd, text: "}"})
		mapLowerCover(covered, forTok, bodyOpen+1)
		rangeTok = bodyOpen
	}
	return edits, covered
}

func mapLowerConstructionEdits(program *unit.Program, specs []mapLowerSpec, edits []functionValueEdit, covered []bool, literals *[]mapLowerLiteral, makes *[]mapLowerMake) ([]functionValueEdit, []bool, bool) {
	for i := 0; i < len(program.Tokens); i++ {
		if covered[i] {
			continue
		}
		if functionValueTokenEquals(program, i, "make") && functionValueTokenEquals(program, i+1, "(") {
			close := functionValueFindMatchingParen(program, i+1)
			starts, ends := ordinaryBuiltinArguments(program, i+2, close)
			if close < 0 || len(starts) < 1 || len(starts) > 2 {
				continue
			}
			result := functionValueTokensText(program, starts[0], ends[0])
			spec := mapLowerSpecIndex(specs, ordinaryUnderlyingType(program, result, 0))
			if spec < 0 {
				continue
			}
			hint := "0"
			if len(starts) == 2 {
				hint = mapLowerReadText(program, specs, starts[1], ends[1])
			}
			makeName := specs[spec].makeValue
			if compactMapLowerType(result) != specs[spec].typ {
				makeIndex := mapLowerMakeIndex(*makes, spec, result)
				if makeIndex < 0 {
					makeIndex = len(*makes)
					*makes = append(*makes, mapLowerMake{spec: spec, name: "__renvo_map_named_make_" + functionValueDecimal(makeIndex), result: result})
				}
				makeName = (*makes)[makeIndex].name
			}
			edits = append(edits, functionValueTokenRangeEdit(program, i, close+1, makeName+"("+hint+")"))
			mapLowerCover(covered, i, close+1)
			i = close
			continue
		}
		if !functionValueTokenEquals(program, i, "{") {
			continue
		}
		typeStart := mapLowerLiteralTypeStart(program, i)
		if typeStart < 0 || functionValueTypeEnd(program, typeStart) != i {
			continue
		}
		close := functionValueFindMatchingBrace(program, i)
		if close < 0 {
			return nil, nil, false
		}
		typeText := functionValueTokensText(program, typeStart, i)
		spec := mapLowerSpecIndex(specs, ordinaryUnderlyingType(program, typeText, 0))
		if spec < 0 {
			replacement, transformed, ok := mapLowerSequenceLiteral(program, specs, typeText, i, close, literals)
			if !ok {
				return nil, nil, false
			}
			if transformed {
				edits = append(edits, functionValueTokenRangeEdit(program, typeStart, close+1, replacement))
				mapLowerCover(covered, typeStart, close+1)
				i = close
			}
			continue
		}
		call, ok := mapLowerCreateLiteral(program, specs, spec, typeText, i, close, literals)
		if !ok {
			return nil, nil, false
		}
		edits = append(edits, functionValueTokenRangeEdit(program, typeStart, close+1, call))
		mapLowerCover(covered, typeStart, close+1)
		i = close
	}
	return edits, covered, true
}

func mapLowerSequenceLiteral(program *unit.Program, specs []mapLowerSpec, typ string, open int, close int, literals *[]mapLowerLiteral) (string, bool, bool) {
	element := ordinaryIndexedElementType(program, typ)
	if mapLowerSpecIndex(specs, ordinaryUnderlyingType(program, element, 0)) < 0 {
		return "", false, true
	}
	out := mapLowerGeneratedType(specs, typ) + "{"
	start := open + 1
	first := true
	for start < close {
		end := mapLowerNextComma(program, start, close)
		colon := mapLowerTopLevelToken(program, start, end, ":")
		valueStart := start
		item := ""
		if colon >= 0 {
			item = mapLowerReadText(program, specs, start, colon) + ":"
			valueStart = colon + 1
		}
		value, ok := mapLowerLiteralValue(program, specs, valueStart, end, element, literals)
		if !ok {
			return "", false, false
		}
		if !first {
			out += ","
		}
		out += item + value
		first = false
		start = end + 1
	}
	return out + "}", true, true
}

func mapLowerLiteralTypeStart(program *unit.Program, open int) int {
	match := -1
	for i := open - 1; i >= 0; i-- {
		if functionValueTypeEnd(program, i) == open {
			// Nested types end at the same literal brace. Keep walking so the
			// complete outer type owns the literal; in []map[K]V{...}, for
			// example, the slice rather than its map element owns the brace.
			match = i
		}
	}
	if match >= 0 {
		return match
	}
	return functionValuePrimaryStart(program, open-1)
}

func mapLowerBuiltinEdits(program *unit.Program, specs []mapLowerSpec, edits []functionValueEdit, covered []bool) ([]functionValueEdit, []bool) {
	for i := 0; i+1 < len(program.Tokens); i++ {
		if covered[i] || !functionValueTokenEquals(program, i+1, "(") {
			continue
		}
		name := functionValueTokenText(program, i)
		if name != "delete" && name != "len" && name != "clear" {
			continue
		}
		close := functionValueFindMatchingParen(program, i+1)
		if close < 0 {
			continue
		}
		starts, ends := ordinaryBuiltinArguments(program, i+2, close)
		if len(starts) < 1 {
			continue
		}
		spec := mapLowerExprSpec(program, specs, i, starts[0], ends[0])
		if spec < 0 {
			continue
		}
		mapping := "(" + mapLowerReadText(program, specs, starts[0], ends[0]) + ")"
		replacement := ""
		if name == "len" && len(starts) == 1 {
			replacement = specs[spec].length + "(" + mapping + ")"
		} else if name == "delete" && len(starts) == 2 {
			replacement = specs[spec].remove + "(" + mapping + "," + mapLowerReadText(program, specs, starts[1], ends[1]) + ")"
		} else if name == "clear" && len(starts) == 1 {
			replacement = specs[spec].storage + "_clear(" + mapping + ")"
		}
		if replacement == "" {
			continue
		}
		edits = append(edits, functionValueTokenRangeEdit(program, i, close+1, replacement))
		mapLowerCover(covered, i, close+1)
		i = close
	}
	return edits, covered
}

func mapLowerIndexEdits(program *unit.Program, specs []mapLowerSpec, edits []functionValueEdit, covered []bool) ([]functionValueEdit, []bool, bool) {
	for open := len(program.Tokens) - 1; open >= 0; open-- {
		if covered[open] || !functionValueTokenEquals(program, open, "[") {
			continue
		}
		close := functionValueFindMatching(program, open, "[", "]")
		if close < 0 || close >= len(covered) || covered[close] {
			continue
		}
		baseStart := functionValuePrimaryStart(program, open-1)
		if baseStart < 0 || baseStart >= open || functionValueTokenEquals(program, baseStart, "map") {
			continue
		}
		spec := mapLowerExprSpec(program, specs, open, baseStart, open)
		if spec < 0 {
			continue
		}
		base := mapLowerReadText(program, specs, baseStart, open)
		key := mapLowerReadText(program, specs, open+1, close)
		mapping := "(" + base + ")"
		replacement := specs[spec].get + "(" + mapping + "," + key + ")"
		if mapLowerIndexIsAssignmentTarget(program, close) {
			replacement = "(*" + specs[spec].ref + "(" + mapping + "," + key + "))"
		} else if mapLowerIndexIsCommaOK(program, baseStart, close) {
			replacement = specs[spec].lookup + "(" + mapping + "," + key + ")"
		}
		edits = append(edits, functionValueTokenRangeEdit(program, baseStart, close+1, replacement))
		mapLowerCover(covered, baseStart, close+1)
		open = baseStart
	}
	return edits, covered, true
}

func mapLowerCreateLiteral(program *unit.Program, specs []mapLowerSpec, spec int, result string, open int, close int, literals *[]mapLowerLiteral) (string, bool) {
	index := len(*literals)
	literal := mapLowerLiteral{spec: spec, name: "__renvo_map_literal_" + functionValueDecimal(index), result: result}
	*literals = append(*literals, literal)
	start := open + 1
	for start < close {
		end := mapLowerNextComma(program, start, close)
		colon := mapLowerTopLevelToken(program, start, end, ":")
		if colon < 0 {
			return "", false
		}
		key, ok := mapLowerLiteralValue(program, specs, start, colon, specs[spec].key, literals)
		if !ok {
			return "", false
		}
		value, ok := mapLowerLiteralValue(program, specs, colon+1, end, specs[spec].value, literals)
		if !ok {
			return "", false
		}
		literal.keys = append(literal.keys, key)
		literal.values = append(literal.values, value)
		start = end + 1
	}
	(*literals)[index] = literal
	args := ""
	for item := 0; item < len(literal.keys); item++ {
		if item > 0 {
			args += ","
		}
		args += literal.keys[item] + "," + literal.values[item]
	}
	return literal.name + "(" + args + ")", true
}

func mapLowerLiteralValue(program *unit.Program, specs []mapLowerSpec, start int, end int, expected string, literals *[]mapLowerLiteral) (string, bool) {
	if start >= end {
		return "", false
	}
	if functionValueTokenEquals(program, start, "{") {
		close := functionValueFindMatchingBrace(program, start)
		spec := mapLowerSpecIndex(specs, ordinaryUnderlyingType(program, expected, 0))
		if close == end-1 && spec >= 0 {
			return mapLowerCreateLiteral(program, specs, spec, expected, start, close, literals)
		}
		return expected + mapLowerReadText(program, specs, start, end), true
	}
	typeEnd := functionValueTypeEnd(program, start)
	if typeEnd > start && typeEnd < end && functionValueTokenEquals(program, typeEnd, "{") {
		close := functionValueFindMatchingBrace(program, typeEnd)
		spec := mapLowerSpecIndex(specs, ordinaryUnderlyingType(program, functionValueTokensText(program, start, typeEnd), 0))
		if close == end-1 && spec >= 0 {
			return mapLowerCreateLiteral(program, specs, spec, functionValueTokensText(program, start, typeEnd), typeEnd, close, literals)
		}
	}
	return mapLowerReadText(program, specs, start, end), true
}

type mapLowerReadCandidate struct {
	start  int
	open   int
	close  int
	spec   int
	kind   int
	argEnd int
}

func mapLowerReadText(program *unit.Program, specs []mapLowerSpec, start int, end int) string {
	if start < 0 || start >= end || end > len(program.Tokens) {
		return ""
	}
	var candidates []mapLowerReadCandidate
	for open := start; open < end; open++ {
		if !functionValueTokenEquals(program, open, "[") {
			continue
		}
		close := functionValueFindMatching(program, open, "[", "]")
		if close < open || close >= end {
			continue
		}
		baseStart := functionValuePrimaryStart(program, open-1)
		if baseStart < start || baseStart >= open || functionValueTokenEquals(program, baseStart, "map") {
			continue
		}
		spec := mapLowerExprSpec(program, specs, open, baseStart, open)
		if spec >= 0 {
			candidates = append(candidates, mapLowerReadCandidate{start: baseStart, open: open, close: close, spec: spec})
		}
	}
	for call := start; call+2 < end; call++ {
		if !functionValueTokenEquals(program, call, "len") || !functionValueTokenEquals(program, call+1, "(") {
			continue
		}
		close := functionValueFindMatchingParen(program, call+1)
		if close >= end {
			continue
		}
		starts, ends := ordinaryBuiltinArguments(program, call+2, close)
		if len(starts) != 1 {
			continue
		}
		spec := mapLowerExprSpec(program, specs, call, starts[0], ends[0])
		if spec >= 0 {
			candidates = append(candidates, mapLowerReadCandidate{start: call, open: starts[0], close: close, spec: spec, kind: 1, argEnd: ends[0]})
		}
	}
	byteStart := program.Tokens[start].Start
	last := program.Tokens[end-1]
	byteEnd := last.Start + last.Size
	var edits []functionValueEdit
	for i := 0; i < len(candidates); i++ {
		candidate := candidates[i]
		contained := false
		for j := 0; j < len(candidates); j++ {
			other := candidates[j]
			if i != j && other.start <= candidate.start && other.close >= candidate.close && (other.start < candidate.start || other.close > candidate.close) {
				contained = true
				break
			}
		}
		if contained {
			continue
		}
		replacement := ""
		if candidate.kind == 1 {
			mapping := mapLowerReadText(program, specs, candidate.open, candidate.argEnd)
			replacement = specs[candidate.spec].length + "((" + mapping + "))"
		} else {
			base := mapLowerReadText(program, specs, candidate.start, candidate.open)
			key := mapLowerReadText(program, specs, candidate.open+1, candidate.close)
			replacement = specs[candidate.spec].get + "((" + base + ")," + key + ")"
		}
		first := program.Tokens[candidate.start]
		last := program.Tokens[candidate.close]
		edits = append(edits, functionValueEdit{
			start: first.Start - byteStart,
			end:   last.Start + last.Size - byteStart,
			text:  replacement,
		})
	}
	if len(edits) == 0 {
		return string(program.Text[byteStart:byteEnd])
	}
	text, ok := applyFunctionValueEdits(program.Text[byteStart:byteEnd], edits)
	if !ok {
		return ""
	}
	return string(text)
}

func mapLowerIndexIsAssignmentTarget(program *unit.Program, close int) bool {
	for i := close + 1; i < len(program.Tokens); i++ {
		if program.Tokens[i].KindLine>>8 != program.Tokens[close].KindLine>>8 {
			return false
		}
		text := functionValueTokenText(program, i)
		if text == "=" || text == ":=" || text == "+=" || text == "-=" || text == "*=" || text == "/=" || text == "%=" || text == "&=" || text == "|=" || text == "^=" || text == "&^=" || text == "<<=" || text == ">>=" {
			return true
		}
		if text == "++" || text == "--" {
			return true
		}
		if text == ";" || text == "{" || text == "}" {
			return false
		}
	}
	return false
}

func mapLowerIndexIsCommaOK(program *unit.Program, start int, close int) bool {
	assign := -1
	for i := start - 1; i >= 0; i-- {
		text := functionValueTokenText(program, i)
		if text == "=" || text == ":=" {
			assign = i
			break
		}
		if text == ";" || text == "{" || text == "}" {
			break
		}
		if i+1 < start && program.Tokens[i].KindLine>>8 != program.Tokens[i+1].KindLine>>8 {
			break
		}
	}
	if assign < 0 {
		return false
	}
	comma := false
	for i := assign - 1; i >= 0; i-- {
		text := functionValueTokenText(program, i)
		if text == "," {
			comma = true
		}
		if text == ";" || text == "{" || text == "}" || i+1 < assign && program.Tokens[i].KindLine>>8 != program.Tokens[i+1].KindLine>>8 {
			break
		}
	}
	if !comma {
		return false
	}
	for i := assign + 1; i < start; i++ {
		if functionValueTokenText(program, i) != "(" {
			return false
		}
	}
	for i := close + 1; i < len(program.Tokens) && program.Tokens[i].KindLine>>8 == program.Tokens[close].KindLine>>8; i++ {
		text := functionValueTokenText(program, i)
		if text == ")" {
			continue
		}
		if text == ";" || text == "}" {
			break
		}
		return false
	}
	return true
}

func mapLowerCover(covered []bool, start int, end int) {
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

func mapLowerNextComma(program *unit.Program, start int, end int) int {
	comma := mapLowerTopLevelToken(program, start, end, ",")
	if comma < 0 {
		return end
	}
	return comma
}

func mapLowerTopLevelToken(program *unit.Program, start int, end int, wanted string) int {
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
		} else if paren == 0 && bracket == 0 && brace == 0 && text == wanted {
			return i
		}
	}
	return -1
}

func mapLowerGeneratedText(specs []mapLowerSpec, literals []mapLowerLiteral, makes []mapLowerMake) string {
	out := ""
	for i := 0; i < len(specs); i++ {
		spec := specs[i]
		key := mapLowerGeneratedType(specs, spec.key)
		value := mapLowerGeneratedType(specs, spec.value)
		representation := mapLowerRepresentation(spec)
		out += "type " + spec.entry + " struct { key " + key + "; value " + value + " }\n"
		out += "type " + spec.storage + " struct { entries []" + spec.entry + " }\n"
		out += "func " + spec.find + "(mapping " + representation + ", key " + key + ") int { if key != key { return -1 }; if mapping == nil { return -1 }; entries := mapping[0].entries; for index := 0; index < len(entries); index++ { if entries[index].key == key { return index } }; return -1 }\n"
		out += "func " + spec.get + "(mapping " + representation + ", key " + key + ") " + value + " { index := " + spec.find + "(mapping,key); if index < 0 { var zero " + value + "; return zero }; return mapping[0].entries[index].value }\n"
		out += "func " + spec.lookup + "(mapping " + representation + ", key " + key + ") (" + value + ",bool) { index := " + spec.find + "(mapping,key); if index < 0 { var zero " + value + "; return zero,false }; return mapping[0].entries[index].value,true }\n"
		out += "func " + spec.ref + "(mapping " + representation + ", key " + key + ") *" + value + " { if mapping == nil { panic(\"assignment to entry in nil map\") }; index := " + spec.find + "(mapping,key); entries := &mapping[0].entries; if index < 0 { var added " + spec.entry + "; added.key = key; *entries = append(*entries,added); index = len(*entries)-1 }; return &(*entries)[index].value }\n"
		out += "func " + spec.set + "(mapping " + representation + ", key " + key + ", value " + value + ") { *" + spec.ref + "(mapping,key) = value }\n"
		out += "func " + spec.remove + "(mapping " + representation + ", key " + key + ") { index := " + spec.find + "(mapping,key); if index < 0 { return }; entries := &mapping[0].entries; last := len(*entries)-1; (*entries)[index] = (*entries)[last]; *entries = (*entries)[:last] }\n"
		out += "func " + spec.length + "(mapping " + representation + ") int { if mapping == nil { return 0 }; return len(mapping[0].entries) }\n"
		out += "func " + spec.entries + "(mapping " + representation + ") []" + spec.entry + " { if mapping == nil { return nil }; return mapping[0].entries }\n"
		out += "func " + spec.storage + "_clear(mapping " + representation + ") { if mapping != nil { mapping[0].entries = mapping[0].entries[:0] } }\n"
		out += "func " + spec.makeValue + "(hint int) " + representation + " { if hint < 0 { panic(\"make map: negative size\") }; mapping := make(" + representation + ",1); mapping[0] = &" + spec.storage + "{}; return mapping }\n"
	}
	for i := 0; i < len(makes); i++ {
		made := makes[i]
		out += "func " + made.name + "(hint int) " + mapLowerGeneratedType(specs, made.result) + " { return " + specs[made.spec].makeValue + "(hint) }\n"
	}
	for i := 0; i < len(literals); i++ {
		literal := literals[i]
		spec := specs[literal.spec]
		key := mapLowerGeneratedType(specs, spec.key)
		value := mapLowerGeneratedType(specs, spec.value)
		result := mapLowerGeneratedType(specs, literal.result)
		out += "func " + literal.name + "("
		for item := 0; item < len(literal.keys); item++ {
			if item > 0 {
				out += ","
			}
			out += "key" + functionValueDecimal(item) + " " + key + ",value" + functionValueDecimal(item) + " " + value
		}
		out += ") " + result + " { mapping := " + spec.makeValue + "(0);"
		for item := 0; item < len(literal.keys); item++ {
			out += "*" + spec.ref + "(mapping,key" + functionValueDecimal(item) + ") = value" + functionValueDecimal(item) + ";"
		}
		out += "return mapping }\n"
	}
	return out
}

func mapLowerMakeIndex(makes []mapLowerMake, spec int, result string) int {
	result = compactMapLowerType(result)
	for i := 0; i < len(makes); i++ {
		if makes[i].spec == spec && compactMapLowerType(makes[i].result) == result {
			return i
		}
	}
	return -1
}

func mapLowerGeneratedType(specs []mapLowerSpec, typ string) string {
	compact := compactMapLowerType(typ)
	if spec := mapLowerSpecIndex(specs, compact); spec >= 0 {
		return mapLowerRepresentation(specs[spec])
	}
	// Recursively rewrite maps nested in value types. The token-level pass
	// handles authored source; generated helper signatures need the same shape.
	out := ""
	for i := 0; i < len(typ); {
		if i+4 <= len(typ) && compactMapLowerType(typ[i:i+4]) == "map[" {
			end := mapLowerStringTypeEnd(typ, i)
			if end > i {
				part := compactMapLowerType(typ[i:end])
				if spec := mapLowerSpecIndex(specs, part); spec >= 0 {
					out += mapLowerRepresentation(specs[spec])
					i = end
					continue
				}
			}
		}
		out += typ[i : i+1]
		i++
	}
	return out
}

func mapLowerRepresentation(spec mapLowerSpec) string {
	return "[]*" + spec.storage
}

func mapLowerStringTypeEnd(typ string, start int) int {
	i := start + 3
	depth := 0
	for i < len(typ) {
		if typ[i] == '[' {
			depth++
		} else if typ[i] == ']' {
			depth--
			if depth == 0 {
				i++
				break
			}
		}
		i++
	}
	if depth != 0 || i >= len(typ) {
		return -1
	}
	// Generated nested type discovery is limited to type strings already seen
	// in the authored program, so scan until the surrounding delimiter.
	for i < len(typ) && typ[i] != ',' && typ[i] != ';' && typ[i] != '}' && typ[i] != ')' {
		i++
	}
	return i
}
