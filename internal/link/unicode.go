package link

import "renvo.dev/internal/arena"
import "renvo.dev/internal/unit"
import "renvo.dev/internal/syntax"

// Encode non-ASCII identifiers after source-level resolution. Use an unused
// prefix and the complete UTF-8 byte sequence, preserving identity without
// collisions while keeping the compact backend source subset ASCII-only.
func lowerUnicodeIdentifiers(program *unit.Program, transient bool) bool {
	prefix := "__renvo_unicode_"
	var edits []functionValueEdit
	const digits = "0123456789abcdef"
	for i := 0; i < len(program.Tokens); i++ {
		token := &program.Tokens[i]
		if token.KindLine&255 != unit.TokenIdent || token.Start < 0 || token.Size <= 0 || token.Start > len(program.Text)-token.Size {
			continue
		}
		name := program.Text[token.Start : token.Start+token.Size]
		unicode := false
		for _, value := range name {
			if value >= 128 {
				unicode = true
				break
			}
		}
		if !unicode {
			continue
		}
		// Select a collision-free prefix only when an identifier needs an edit.
		// Comments and literals do not allocate names or cause a second walk.
		if len(edits) == 0 {
			for {
				used := false
				for i := 0; i < len(program.Tokens); i++ {
					if program.Tokens[i].KindLine&255 == unit.TokenIdent && (functionValueHasPrefix(functionValueTokenText(program, i), prefix) || functionValueHasPrefix(functionValueTokenText(program, i), "Renvo"+prefix)) {
						used = true
						break
					}
				}
				if !used {
					break
				}
				prefix += "_"
			}
		}
		encodedPrefix := prefix
		if syntax.IdentifierExported(name, 0) {
			encodedPrefix = "Renvo" + prefix
		}
		encoded := []byte(encodedPrefix)
		for j := 0; j < len(name); j++ {
			encoded = append(encoded, digits[name[j]>>4], digits[name[j]&15])
		}
		edits = append(edits, functionValueTokenRangeEdit(program, i, i+1, string(encoded)))
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
	return reparseFunctionValueProgram(program, text, edits, originalLength, len(text))
}
