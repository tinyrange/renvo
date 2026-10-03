package link

import "renvo.dev/internal/unit"

// VarSpec groups share no types or initializers between specifications. Emit
// each specification as its own declaration so all subsequent lowering uses
// the same declaration boundaries for grouped and ungrouped source.
func lowerVariableGroups(program *unit.Program, transient bool) bool {
	var edits []functionValueEdit
	for token := 0; token+1 < len(program.Tokens); token++ {
		if program.Tokens[token].KindLine&255 != unit.TokenVar || !functionValueTokenEquals(program, token+1, "(") {
			continue
		}
		close := functionValueFindMatchingParen(program, token+1)
		if close < 0 {
			return false
		}
		edits = append(edits, functionValueTokenEdit(program, token, ""), functionValueTokenEdit(program, token+1, ""), functionValueTokenEdit(program, close, "\n"))
		if functionValueTokenEquals(program, close+1, ";") {
			edits = append(edits, functionValueTokenEdit(program, close+1, ""))
		}
		for pos := token + 2; pos < close; {
			if functionValueTokenEquals(program, pos, ";") {
				pos++
				continue
			}
			start := program.Tokens[pos].Start
			edits = append(edits, functionValueEdit{start: start, end: start, text: "var "})
			pos = variableGroupSpecEnd(program, pos, close)
		}
	}
	if len(edits) == 0 {
		return true
	}
	edits = appendFunctionValuePackageEdits(program, edits)
	originalLength := len(program.Text)
	text, ok := applyFunctionValueEdits(program.Text, edits)
	return ok && reparseFunctionValueProgramMode(program, text, edits, originalLength, -1, transient)
}

func variableGroupSpecEnd(program *unit.Program, start int, end int) int {
	depth := 0
	for token := start; token < end; token++ {
		if depth == 0 {
			if functionValueTokenEquals(program, token, ";") {
				return token + 1
			}
			if token > start && program.Tokens[token].KindLine>>8 != program.Tokens[token-1].KindLine>>8 && variableGroupTokenEndsSpec(program, token-1) {
				return token
			}
		}
		if functionValueTokenEquals(program, token, "(") || functionValueTokenEquals(program, token, "[") || functionValueTokenEquals(program, token, "{") {
			depth++
		} else if functionValueTokenEquals(program, token, ")") || functionValueTokenEquals(program, token, "]") || functionValueTokenEquals(program, token, "}") {
			depth--
		}
	}
	return end
}

func variableGroupTokenEndsSpec(program *unit.Program, token int) bool {
	kind := program.Tokens[token].KindLine & 255
	return kind == unit.TokenIdent || kind == unit.TokenNumber || kind == unit.TokenString || kind == unit.TokenChar ||
		functionValueTokenEquals(program, token, ")") || functionValueTokenEquals(program, token, "]") || functionValueTokenEquals(program, token, "}")
}
