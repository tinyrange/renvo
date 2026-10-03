package link

import (
	"renvo.dev/internal/arena"
	"renvo.dev/internal/unit"
)

func functionValueCanonicalCallableFields(program *unit.Program, start int, end int) []string {
	var types []string
	for _, field := range functionValueSemanticFields(program, start, end) {
		first := field.start
		prefix := ""
		if functionValueTokenCharIs(program, first, '.') && functionValueTokenCharIs(program, first+1, '.') && functionValueTokenCharIs(program, first+2, '.') {
			prefix, first = "...", first+3
		}
		types = append(types, prefix+functionValueCanonicalSignatureType(program, first, field.end, 0))
	}
	return types
}

// Resolve aliases in function signatures before selecting their shared storage.
// Definitions retain their names. Parameter/result names stay in their original
// scopes, and universe byte/rune aliases apply only when no declaration hides them.
func lowerFunctionSignatureAliases(program *unit.Program, transient bool) bool {
	var edits []functionValueEdit
	for token := 0; token+1 < len(program.Tokens); token++ {
		if !functionValueTokenKindIs(program, token, unit.TokenFunc) || !functionValueTokenCharIs(program, token+1, '(') || functionValueIsDeclaredFunction(program, token) {
			continue
		}
		end := functionValueTypeEnd(program, token)
		if end <= token {
			continue
		}
		mark := arena.Mark()
		original := functionValueTokensText(program, token, end)
		replacement := functionValueCanonicalSignature(program, token, end, 0)
		if replacement != original {
			edits = append(edits, functionValueTokenRangeEdit(program, token, end, replacement))
		} else {
			arena.Rewind(mark)
		}
		token = end - 1
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
	if !ok {
		return false
	}
	if transient {
		arena.DiscardBytes(program.Text)
	}
	return reparseFunctionValueProgramMode(program, text, edits, originalLength, -1, transient)
}

func functionValueCanonicalSignature(program *unit.Program, start int, end int, depth int) string {
	var edits []functionValueEdit
	close := functionValueFindMatchingParen(program, start+1)
	if close < 0 || close >= end {
		return functionValueTokensText(program, start, end)
	}
	edits = functionValueCanonicalFields(program, start+2, close, depth, edits)
	result := close + 1
	if functionValueTokenCharIs(program, result, '(') {
		resultClose := functionValueFindMatchingParen(program, result)
		if resultClose >= 0 && resultClose < end {
			edits = functionValueCanonicalFields(program, result+1, resultClose, depth, edits)
		}
	} else if result < end {
		typ := functionValueCanonicalSignatureType(program, result, end, depth+1)
		if typ != functionValueTokensText(program, result, end) {
			edits = append(edits, functionValueTokenRangeEdit(program, result, end, typ))
		}
	}
	original := functionValueTokensText(program, start, end)
	if len(edits) == 0 {
		return original
	}
	offset := program.Tokens[start].Start
	for i := range edits {
		edits[i].start -= offset
		edits[i].end -= offset
	}
	text, ok := applyFunctionValueEdits([]byte(original), edits)
	if !ok {
		return original
	}
	return string(text)
}

func functionValueCanonicalFields(program *unit.Program, start int, end int, depth int, edits []functionValueEdit) []functionValueEdit {
	starts, ends := functionValueCommaParts(program, start, end)
	for i := 0; i < len(starts); i++ {
		typ := starts[i]
		if program.Tokens[typ].KindLine&255 == unit.TokenIdent && functionValueTypeEnd(program, typ) != ends[i] {
			typ++
		}
		element := typ
		if functionValueTokenCharIs(program, typ, '.') && functionValueTokenCharIs(program, typ+1, '.') && functionValueTokenCharIs(program, typ+2, '.') {
			element += 3
		}
		if functionValueTypeEnd(program, element) != ends[i] && !functionValueTokenEquals(program, typ, "...") {
			continue
		}
		// A grouped name has no type of its own; its group's final field owns it.
		if typ+1 == ends[i] && i+1 < len(starts) {
			grouped := false
			for j := i + 1; j < len(starts); j++ {
				if starts[j]+1 == ends[j] {
					continue
				}
				grouped = program.Tokens[starts[j]].KindLine&255 == unit.TokenIdent && functionValueTypeEnd(program, starts[j]) != ends[j]
				break
			}
			if grouped {
				continue
			}
		}
		replacement := functionValueCanonicalSignatureType(program, typ, ends[i], depth+1)
		if replacement != functionValueTokensText(program, typ, ends[i]) {
			edits = append(edits, functionValueTokenRangeEdit(program, typ, ends[i], replacement))
		}
	}
	return edits
}

func functionValueCanonicalSignatureType(program *unit.Program, start int, end int, depth int) string {
	return functionValueCanonicalSignatureTypeAt(program, start, end, depth, start)
}

func functionValueCanonicalSignatureTypeAt(program *unit.Program, start int, end int, depth int, use int) string {
	original := functionValueTokensText(program, start, end)
	if start < 0 || start >= end || depth > len(program.Tokens) {
		return original
	}
	token := functionValueTokenText(program, start)
	if token == "." && functionValueTokenCharIs(program, start+1, '.') && functionValueTokenCharIs(program, start+2, '.') {
		return "..." + functionValueCanonicalSignatureTypeAt(program, start+3, end, depth+1, use)
	}
	if token == "*" || token == "..." {
		return token + functionValueCanonicalSignatureTypeAt(program, start+1, end, depth+1, use)
	}
	if token == "[" || token == "map" {
		open := start
		if token == "map" {
			open++
		}
		close := functionValueFindMatching(program, open, "[", "]")
		if close < 0 || close >= end {
			return original
		}
		prefix := functionValueTokensText(program, start, close+1)
		if token == "map" {
			prefix = "map[" + functionValueCanonicalSignatureTypeAt(program, open+1, close, depth+1, use) + "]"
		}
		return prefix + functionValueCanonicalSignatureTypeAt(program, close+1, end, depth+1, use)
	}
	if token == "chan" || token == "<-" {
		next := start + 1
		if token == "<-" {
			next++
		} else if functionValueTokenEquals(program, next, "<-") {
			next++
		}
		return functionValueTokensText(program, start, next) + " " + functionValueCanonicalSignatureTypeAt(program, next, end, depth+1, use)
	}
	if token == "(" && functionValueFindMatchingParen(program, start) == end-1 {
		return functionValueCanonicalSignatureTypeAt(program, start+1, end-1, depth+1, use)
	}
	if token == "func" {
		return functionValueCanonicalSignature(program, start, end, depth+1)
	}
	if program.Tokens[start].KindLine&255 != unit.TokenIdent || start+1 != end {
		return original
	}
	if name := functionValueSignatureLocalType(program, use, token); name >= 0 {
		if !functionValueTokenCharIs(program, name+1, '=') {
			return original
		}
		finish := functionValueTypeEnd(program, name+2)
		if finish > name+2 && functionValueSignatureAliasCanExpand(program, name+2, finish, use) {
			return functionValueCanonicalSignatureTypeAt(program, name+2, finish, depth+1, use)
		}
		return original
	}
	for i := 0; i < len(program.Decls); i++ {
		decl := &program.Decls[i]
		if decl.Kind != unit.TokenType || !ordinarySpanEquals(program.Text, decl.NameStart, decl.NameEnd, token) {
			continue
		}
		name := functionValueTokenAtSpan(program, decl.NameStart, decl.NameEnd)
		if !functionValueTokenCharIs(program, name+1, '=') {
			return original
		}
		target := name + 2
		// Keep aggregate aliases at their declaration site, where private field and
		// method identities belong to the original package.
		if functionValueTokenKindIs(program, target, unit.TokenStruct) || functionValueTokenEquals(program, target, "interface") {
			finish := functionValueTypeEnd(program, target)
			if finish > target {
				if name := functionValueMatchingAggregateAlias(program, target, finish, use); name != "" {
					return name
				}
			}
			return original
		}
		finish := functionValueTypeEnd(program, target)
		if finish <= target {
			return original
		}
		if !functionValueSignatureAliasCanExpand(program, target, finish, use) {
			return original
		}
		return functionValueCanonicalSignatureTypeAt(program, target, finish, depth+1, use)
	}
	if token == "byte" {
		if functionValueSignatureUniverseNameHidden(program, use, "uint8") {
			return original
		}
		return "uint8"
	}
	if token == "rune" {
		if functionValueSignatureUniverseNameHidden(program, use, "int32") {
			return original
		}
		return "int32"
	}
	return original
}

func functionValueSignatureUniverseNameHidden(program *unit.Program, use int, name string) bool {
	if functionValueSignatureLocalType(program, use, name) >= 0 || functionValueLexicalLocalType(program, use, name) != "" || functionValueDeclaredFunction(program, name) {
		return true
	}
	for i := 0; i < len(program.Decls); i++ {
		if ordinarySpanEquals(program.Text, program.Decls[i].NameStart, program.Decls[i].NameEnd, name) {
			return true
		}
	}
	return false
}

// An alias can carry a type through a scope that hides its spelling. Keep that
// alias when expansion would capture a local type or value instead. Check each
// step of an alias chain at the use site, rather than at the declaration site.
func functionValueSignatureAliasCanExpand(program *unit.Program, start int, end int, use int) bool {
	for token := start; token < end; token++ {
		if program.Tokens[token].KindLine&255 != unit.TokenIdent {
			continue
		}
		name := functionValueTokenText(program, token)
		if functionValueSignatureLocalType(program, use, name) >= 0 || functionValueLexicalLocalType(program, use, name) != "" {
			return false
		}
	}
	return true
}

func functionValueSignatureLocalType(program *unit.Program, before int, name string) int {
	enclosing := functionValueEnclosingFunc(program, before)
	if enclosing < 0 {
		return -1
	}
	fn := &program.Funcs[enclosing]
	for token := before - 1; token > fn.BodyStart; token-- {
		if !functionValueTokenEquals(program, token, name) || !functionValueLocalTypeName(program, token) {
			continue
		}
		if functionValueBindingInScope(program, fn, token, before) {
			return token
		}
	}
	return -1
}
