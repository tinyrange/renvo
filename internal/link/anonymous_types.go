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
		// Local variable declarations already accept anonymous types in the
		// backend. Hoisting their type can detach array lengths and field types
		// from local declarations, or merge identically spelled distinct types.
		if anonymousTypeLocalVariable(program, i) {
			i = close
			continue
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

func anonymousTypeLocalVariable(program *unit.Program, start int) bool {
	if functionValueEnclosingFunc(program, start) < 0 {
		return false
	}
	name := start - 1
	// Walk type constructors, not initializer expressions, back to the name
	// list. Matching brackets keeps identifiers inside array bounds separate.
	for name >= 0 {
		if functionValueTokenEquals(program, name, "*") {
			name--
			continue
		}
		if functionValueTokenEquals(program, name, "]") {
			open := functionValueFindMatchingBackward(program, name, "[", "]")
			if open < 0 {
				return false
			}
			name = open - 1
			if functionValueTokenEquals(program, name, "map") {
				name--
			}
			continue
		}
		break
	}
	for name >= 0 && program.Tokens[name].KindLine&255 == unit.TokenIdent {
		if functionValueTokenEquals(program, name-1, "var") {
			return true
		}
		if !functionValueTokenEquals(program, name-1, ",") {
			// A grouped VarSpec has no repeated var keyword. Its nearest
			// containing parenthesis must belong to var, not a call/signature.
			depth := 0
			for tok := name - 1; tok >= 0; tok-- {
				if functionValueTokenEquals(program, tok, ")") {
					depth++
				} else if functionValueTokenEquals(program, tok, "(") {
					if depth == 0 {
						return functionValueTokenEquals(program, tok-1, "var")
					}
					depth--
				}
			}
			return false
		}
		name -= 2
	}
	return false
}
