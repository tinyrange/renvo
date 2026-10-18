package link

import "renvo.dev/internal/unit"

// The checker gives blank package variables no name binding. Give their emitted
// storage private identities so later callable/aggregate lowering cannot merge
// unrelated blank declarations or discard their initializer effects. This runs
// only after source-level name resolution; user code cannot refer to these names.
func lowerBlankPackageVariables(program *unit.Program, transient bool) bool {
	var tokens []int
	for _, decl := range program.Decls {
		if decl.Kind == unit.TokenVar && ordinarySpanEquals(program.Text, decl.NameStart, decl.NameEnd, "_") {
			token := coreTokenAt(*program, decl.NameStart, decl.NameEnd)
			if token < 0 {
				return false
			}
			tokens = append(tokens, token)
		}
	}
	if len(tokens) == 0 {
		return true
	}
	prefix := "__renvo_blank_"
	for {
		used := false
		for i := range program.Tokens {
			if program.Tokens[i].KindLine&255 == unit.TokenIdent && functionValueHasPrefix(functionValueTokenText(program, i), prefix) {
				used = true
				break
			}
		}
		if !used {
			break
		}
		prefix += "_"
	}
	var edits []functionValueEdit
	for _, token := range tokens {
		edits = append(edits, functionValueTokenEdit(program, token, prefix+functionValueDecimal(program.Tokens[token].Start)))
	}
	edits = appendFunctionValuePackageEdits(program, edits)
	oldLength := len(program.Text)
	text, ok := applyFunctionValueEdits(program.Text, edits)
	return ok && reparseFunctionValueProgramMode(program, text, edits, oldLength, -1, transient)
}
