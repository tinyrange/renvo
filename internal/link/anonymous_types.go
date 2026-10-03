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
	var owners []unit.PackageInfo
	var lengths []int
	generated := ""
	for i := 0; i+1 < len(program.Tokens); i++ {
		if (program.Tokens[i].KindLine&255 != unit.TokenStruct && !functionValueTokenEquals(program, i, "interface")) || !functionValueTokenCharIs(program, i+1, '{') {
			continue
		}
		// The right-hand side of a type declaration is already supported.
		if functionValueTokenKindIs(program, i-2, unit.TokenType) || functionValueTokenKindIs(program, i-3, unit.TokenType) {
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
			if !functionValueTokenCharIs(program, tok, ';') {
				key += functionValueTokenText(program, tok) + "\x00"
			}
		}
		owner := unit.PackageInfo{}
		for _, pkg := range program.Packages {
			if program.Tokens[i].Start >= pkg.TextStart && program.Tokens[i].Start < pkg.TextEnd {
				owner = pkg
				break
			}
		}
		index := -1
		for j := 0; j < len(types); j++ {
			if types[j] == key && owners[j].ImportPath == owner.ImportPath {
				index = j
				break
			}
		}
		if index < 0 {
			index = len(types)
			name := ordinaryBuiltinGeneratedName(program, "__renvo_anonymous_type_"+functionValueDecimal(index))
			types = append(types, key)
			names = append(names, name)
			declaration := "type " + name + " = " + text + "\n"
			generated += declaration
			owners = append(owners, owner)
			lengths = append(lengths, len(declaration))
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
	// Each alias keeps the package of its source type. In particular, private
	// fields and interface methods must not acquire the root package's identity.
	if !reparseFunctionValueProgram(program, text, edits, originalLength, -1) {
		return false
	}
	for i, owner := range owners {
		if owner.ImportPath != "" {
			owner.TextStart, owner.TextEnd = generatedStart, generatedStart+lengths[i]
			setFunctionValuePackageTableRanges(&owner, program)
			program.Packages = append(program.Packages, owner)
		}
		generatedStart += lengths[i]
	}
	return true
}

func anonymousTypeLocalVariable(program *unit.Program, start int) bool {
	if functionValueEnclosingFunc(program, start) < 0 {
		return false
	}
	name := start - 1
	// Walk type constructors, not initializer expressions, back to the name
	// list. Matching brackets keeps identifiers inside array bounds separate.
	for name >= 0 {
		if functionValueTokenCharIs(program, name, '*') {
			name--
			continue
		}
		if functionValueTokenCharIs(program, name, ']') {
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
		if functionValueTokenKindIs(program, name-1, unit.TokenVar) {
			return true
		}
		if !functionValueTokenCharIs(program, name-1, ',') {
			// A grouped VarSpec has no repeated var keyword. Its nearest
			// containing parenthesis must belong to var, not a call/signature.
			depth := 0
			for tok := name - 1; tok >= 0; tok-- {
				if functionValueTokenCharIs(program, tok, ')') {
					depth++
				} else if functionValueTokenCharIs(program, tok, '(') {
					if depth == 0 {
						return functionValueTokenKindIs(program, tok-1, unit.TokenVar)
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
