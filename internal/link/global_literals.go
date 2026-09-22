package link

import "renvo.dev/internal/arena"
import "renvo.dev/internal/unit"

// A package-level function literal has no enclosing local environment. Give
// it an ordinary function declaration so global initialization and immediate
// calls share the backend's normal function-value representation.
func lowerGlobalFunctionLiterals(program *unit.Program, transient bool) bool {
	var edits []functionValueEdit
	generated := ""
	count := 0
	for i := 0; i+1 < len(program.Tokens); i++ {
		if !functionValueTokenEquals(program, i, "func") || !functionValueTokenEquals(program, i+1, "(") || functionValueEnclosingFunc(program, i) >= 0 || functionValueIsDeclaredFunction(program, i) || functionValueTokenInDeclaredSignature(program, i) {
			continue
		}
		if functionValueTokenEquals(program, i-1, "]") {
			continue
		}
		sig, open, ok := parseFunctionValueSignature(program, i, "")
		if !ok || !functionValueTokenEquals(program, open, "{") {
			continue
		}
		close := functionValueFindMatchingBrace(program, open)
		if close < 0 {
			return false
		}
		name := ordinaryBuiltinGeneratedName(program, "__renvo_global_literal_"+functionValueDecimal(count))
		generated += "func " + name + "(" + sig.params + ") " + sig.result + " " + functionValueTokensText(program, open, close+1) + "\n"
		edits = append(edits, functionValueTokenRangeEdit(program, i, close+1, name))
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
	if !ok {
		return false
	}
	text = append(text, '\n')
	generatedStart := len(text)
	text = appendFunctionValueString(text, generated)
	return reparseFunctionValueProgram(program, text, edits, originalLength, generatedStart)
}
