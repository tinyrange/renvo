package link

import "renvo.dev/internal/arena"
import "renvo.dev/internal/unit"

// Alias anonymous aggregate types used in expressions and signatures so the
// compact backend receives its ordinary named-type syntax. Aliases preserve
// unnamed Go type identity rather than introducing distinct defined types.
func lowerAnonymousTypes(program *unit.Program, transient bool) bool {
	var edits []functionValueEdit
	var types []string
	var names []string
	generated := ""
	for i := 0; i+1 < len(program.Tokens); i++ {
		if (!functionValueTokenEquals(program, i, "struct") && !functionValueTokenEquals(program, i, "interface")) || !functionValueTokenEquals(program, i+1, "{") {
			continue
		}
		// The right-hand side of a type declaration is already supported.
		if functionValueTokenEquals(program, i-2, "type") || functionValueTokenEquals(program, i-3, "type") {
			continue
		}
		close := functionValueFindMatchingBrace(program, i+1)
		if close < 0 {
			return false
		}
		if close == i+2 {
			continue
		}
		text := functionValueTokensText(program, i, close+1)
		key := ""
		for tok := i; tok <= close; tok++ {
			if !functionValueTokenEquals(program, tok, ";") {
				key += functionValueTokenText(program, tok) + "\x00"
			}
		}
		index := -1
		for j := 0; j < len(types); j++ {
			if types[j] == key {
				index = j
				break
			}
		}
		if index < 0 {
			index = len(types)
			name := ordinaryBuiltinGeneratedName(program, "__renvo_anonymous_type_"+functionValueDecimal(index))
			types = append(types, key)
			names = append(names, name)
			generated += "type " + name + " = " + text + "\n"
		}
		edits = append(edits, functionValueTokenRangeEdit(program, i, close+1, names[index]))
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
