package link

import "renvo.dev/internal/arena"
import "renvo.dev/internal/unit"

// Encode non-ASCII identifiers after source-level resolution. Use an unused
// prefix and the complete UTF-8 byte sequence, preserving identity without
// collisions while keeping the compact backend source subset ASCII-only.
func lowerUnicodeIdentifiers(program *unit.Program, transient bool) bool {
	maybeUnicode := false
	for i := 0; i < len(program.Text); i++ {
		if program.Text[i] >= 128 {
			maybeUnicode = true
			break
		}
	}
	if !maybeUnicode {
		return true
	}
	prefix := "__renvo_unicode_"
	for {
		used := false
		for i := 0; i < len(program.Tokens); i++ {
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
	const digits = "0123456789abcdef"
	for i := 0; i < len(program.Tokens); i++ {
		if program.Tokens[i].KindLine&255 != unit.TokenIdent {
			continue
		}
		name := functionValueTokenText(program, i)
		unicode := false
		for j := 0; j < len(name); j++ {
			if name[j] >= 128 {
				unicode = true
				break
			}
		}
		if !unicode {
			continue
		}
		encoded := []byte(prefix)
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
