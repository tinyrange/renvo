package link

import "renvo.dev/internal/arena"
import "renvo.dev/internal/unit"

// Keep the bound evaluation in the loop initializer, and the declared range
// variable in the body: each iteration must introduce a fresh binding.
func lowerIntegerRangesCore(program *unit.Program, transient bool) bool {
	return lowerRangesCore(program, transient, false, true)
}

func lowerIntegerRangeAt(program *unit.Program, transient bool, i int, rangeTok int, open int, typ string, count int) bool {
	close := functionValueFindMatchingBrace(program, open)
	if close < 0 {
		return false
	}
	bound := integerRangeName(program, "__renvo_integer_range_bound_"+functionValueDecimal(count))
	index := integerRangeName(program, "__renvo_integer_range_index_"+functionValueDecimal(count))
	prefix := ""
	if rangeTok != i+1 {
		assign := concurrencyTopLevelAssignment(program, i+1, rangeTok)
		if assign < 0 {
			return false
		}
		name := functionValueTokensText(program, i+1, assign)
		if concurrencyTextHasComma(name) {
			return false
		}
		if name != "_" {
			prefix = name + " " + functionValueTokenText(program, assign) + " " + index + "; "
		}
		if functionValueTokenEquals(program, assign, "=") && ordinaryUntypedExpression(program, rangeTok+1, open) {
			candidate := ordinaryBuiltinExprType(program, i, i+1, assign)
			if candidate != "" {
				typ = candidate
			}
		}
	}
	replacement := "for " + bound + ", " + index + " := " + functionValueTokensText(program, rangeTok+1, open) + ", " + typ + "(0); " + index + " < " + bound + "; " + index + "++ { " + prefix + functionValueTokensText(program, open+1, close) + " }"
	edits := []functionValueEdit{functionValueTokenRangeEdit(program, i, close+1, replacement)}
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
	return true
}

// Generated loop variables must not capture any authored binding or reference.
func integerRangeName(program *unit.Program, base string) string {
	name := base
	for {
		found := false
		for i := 0; i < len(program.Tokens); i++ {
			if functionValueTokenEquals(program, i, name) {
				found = true
				break
			}
		}
		if !found {
			return name
		}
		name += "_"
	}
}
