package link

import "renvo.dev/internal/unit"

// Removing an import qualifier must not let the reference resolve to an
// authored local binding. Package-level duplicate detection alone misses
// parameters, local types, constants, and short declarations. Only inspect
// symbols actually referenced from another package and not already aliased.
func coreAliasImportedBindings(programs []unit.Program, offsets []int, aliases []string, buckets, next []int, names []string) {
	candidates := make([]int, len(aliases))
	for pkg := range programs {
		program := &programs[pkg]
		var touched []int
		for _, selector := range program.Selectors {
			if selector.BaseKind == unit.RefImport {
				coreAliasImportCandidate(candidates, aliases, offsets, pkg, selector.Package, selector.Symbol, &touched)
			}
		}
		for _, ref := range program.TypeRefs {
			if ref.Kind == unit.TypeRefImportSelector || ref.Kind == unit.TypeRefPackage {
				coreAliasImportCandidate(candidates, aliases, offsets, pkg, ref.Package, ref.Symbol, &touched)
			}
		}
		for _, ref := range program.Refs {
			if ref.Kind == unit.RefPackage {
				coreAliasImportCandidate(candidates, aliases, offsets, pkg, ref.Package, ref.Index, &touched)
			}
		}
		if len(touched) == 0 {
			continue
		}
		// Semantic package references are safe and must not be mistaken for
		// local declarations. Declaration tokens and selector names are also
		// separate namespaces. Unresolved tokens include unused parameters,
		// whose names still shadow an imported call after qualifier removal.
		skip := make([]bool, len(program.Tokens))
		for _, symbol := range program.Symbols {
			coreAliasSkipBinding(skip, symbol.Token)
		}
		for _, ref := range program.Refs {
			if ref.Kind == unit.RefPackage {
				coreAliasSkipBinding(skip, ref.Token)
			}
		}
		for _, selector := range program.Selectors {
			coreAliasSkipBinding(skip, selector.NameTok)
			if selector.BaseKind == unit.RefImport {
				coreAliasSkipBinding(skip, selector.BaseTok)
			}
		}
		for _, ref := range program.TypeRefs {
			if ref.Kind == unit.TypeRefPackage || ref.Kind == unit.TypeRefImportSelector {
				coreAliasSkipBinding(skip, ref.Token)
				if ref.Kind == unit.TypeRefImportSelector {
					coreAliasSkipBinding(skip, ref.BaseTok)
				}
			}
		}
		for tok, token := range program.Tokens {
			if token.KindLine&255 != unit.TokenIdent || skip[tok] ||
				(tok > 0 && (program.Tokens[tok-1].KindLine&255 == unit.TokenPackage || functionValueTokenCharIs(program, tok-1, '.'))) {
				continue
			}
			bucket := coreAliasTokenHash(program, tok) % len(buckets)
			for index := buckets[bucket]; index >= 0; index = next[index] {
				if candidates[index] != 0 && aliases[index] == "" && coreTokenTextEquals(program, tok, names[index]) && !coreSymbolKeepsRuntimeName(names[index]) && coreAliasLocalBinding(program, tok) {
					aliases[index] = coreSymbolAliasName(candidates[index]-1, names[index])
				}
			}
		}
		for _, index := range touched {
			candidates[index] = 0
		}
	}
}

func coreAliasLocalBinding(program *unit.Program, token int) bool {
	first, last := token, token
	for first >= 2 && functionValueTokenCharIs(program, first-1, ',') && program.Tokens[first-2].KindLine&255 == unit.TokenIdent {
		first -= 2
	}
	for last+2 < len(program.Tokens) && functionValueTokenCharIs(program, last+1, ',') && program.Tokens[last+2].KindLine&255 == unit.TokenIdent {
		last += 2
	}
	if coreTokenTextEquals(program, last+1, ":=") {
		return true
	}
	if first > 0 {
		kind := program.Tokens[first-1].KindLine & 255
		if kind == unit.TokenVar || kind == unit.TokenConst || kind == unit.TokenType {
			return true
		}
	}
	// Names in keyed literals, selectors and dot-import uses do not introduce
	// bindings. A parameter name is followed by its type; unnamed parameter
	// types instead end at a comma or closing parenthesis.
	typed := functionValueTokenCanStartType(program, last+1)
	if !typed && !functionValueTokenCharIs(program, last+1, '=') {
		return false
	}
	depth := 0
	for i := first - 1; i >= 0; i-- {
		if functionValueTokenCharIs(program, i, ')') {
			depth++
		} else if functionValueTokenCharIs(program, i, '(') {
			if depth > 0 {
				depth--
				continue
			}
			if i > 0 {
				kind := program.Tokens[i-1].KindLine & 255
				if kind == unit.TokenVar || kind == unit.TokenConst || kind == unit.TokenType {
					return true
				}
			}
			return typed && coreAliasParameterParen(program, i)
		} else if depth == 0 && (functionValueTokenCharIs(program, i, '{') || functionValueTokenCharIs(program, i, '}')) {
			return false
		}
	}
	return false
}

func coreAliasParameterParen(program *unit.Program, open int) bool {
	for open > 0 {
		if program.Tokens[open-1].KindLine&255 == unit.TokenFunc {
			return true
		}
		for _, fn := range program.Funcs {
			if fn.NameTok == open-1 {
				return true
			}
		}
		if !functionValueTokenCharIs(program, open-1, ')') {
			return false
		}
		open = functionValueFindMatchingBackward(program, open-1, "(", ")")
	}
	return false
}

func coreAliasImportCandidate(candidates []int, aliases []string, offsets []int, own, pkg, symbol int, touched *[]int) {
	if pkg == own || pkg < 0 || pkg >= len(offsets) || symbol < 0 {
		return
	}
	index := offsets[pkg] + symbol
	if index >= len(aliases) || aliases[index] != "" || candidates[index] != 0 {
		return
	}
	candidates[index] = pkg + 1
	*touched = append(*touched, index)
}

func coreAliasSkipBinding(skip []bool, token int) {
	if token >= 0 && token < len(skip) {
		skip[token] = true
	}
}

// Hash token bytes directly: most identifiers do not match any imported
// symbol, so copying every spelling into an arena string would waste memory.
func coreAliasTokenHash(program *unit.Program, tok int) int {
	token := program.Tokens[tok]
	hash := 5381
	for i := token.Start; i < token.Start+token.Size; i++ {
		hash = ((hash << 5) + hash) ^ int(program.Text[i])
	}
	return hash & 2147483647
}
