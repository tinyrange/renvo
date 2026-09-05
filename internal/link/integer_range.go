package link

import "renvo.dev/internal/arena"
import "renvo.dev/internal/unit"

// Keep the bound evaluation in the loop initializer, and the declared range
// variable in the body: each iteration must introduce a fresh binding.
func lowerIntegerRangesCore(program *unit.Program, transient bool) bool {
	count := 0
	for {
		changed := false
		for i := len(program.Tokens) - 1; i >= 0; i-- {
			if !functionValueTokenEquals(program, i, "for") {
				continue
			}
			open := concurrencyTopLevelToken(program, i+1, len(program.Tokens), "{")
			if open < 0 {
				continue
			}
			rangeTok := concurrencyTopLevelToken(program, i+1, open, "range")
			if rangeTok < 0 {
				continue
			}
			typ := ordinaryBuiltinExprType(program, rangeTok, rangeTok+1, open)
			if !mapLowerIntegerKey(ordinaryUnderlyingType(program, typ, 0)) {
				continue
			}
			close := functionValueFindMatchingBrace(program, open)
			if close < 0 {
				return false
			}
			bound := "__renvo_integer_range_bound_" + functionValueDecimal(count)
			index := "__renvo_integer_range_index_" + functionValueDecimal(count)
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
			count++
			changed = true
			break
		}
		if !changed {
			return true
		}
	}
}
