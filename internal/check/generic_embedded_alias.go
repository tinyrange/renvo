package check

import "renvo.dev/internal/syntax"

// Member and signature names live in different scopes from local type names.
// Concretizing an alias must leave their declarations and branch labels intact.
func genericTypeMemberName(file *syntax.File, token int) bool {
	if token > 0 {
		kind := file.Tokens[token-1].KindLine & 255
		if kind == syntax.TokenGoto || kind == syntax.TokenBreak || kind == syntax.TokenContinue {
			return true
		}
	}
	for start := 0; start < token; start++ {
		kind := file.Tokens[start].KindLine & 255
		if (kind == syntax.TokenStruct || kind == syntax.TokenInterface) && tokCharIs(file, start+1, '{') {
			close := findTypeMatching(file, start+1, '{', '}')
			if close <= token {
				continue
			}
			if kind == syntax.TokenStruct {
				for _, field := range parseStructFields(file, start+2, close-1) {
					if field.NameTok == token {
						return true
					}
				}
			} else {
				methods, _ := parseInterfaceElements(file, start+2, close-1)
				for _, method := range methods {
					if method.NameTok == token {
						return true
					}
					for _, field := range append(method.Signature.Params, method.Signature.Results...) {
						if field.NameTok == token {
							return true
						}
					}
				}
			}
		}
		if kind == syntax.TokenFunc && tokCharIs(file, start+1, '(') {
			close := findTypeMatching(file, start+1, '(', ')')
			if close <= start+1 {
				continue
			}
			for _, field := range parseFieldList(file, start+2, close-1) {
				if field.NameTok == token {
					return true
				}
			}
			if tokCharIs(file, close, '(') {
				end := findTypeMatching(file, close, '(', ')')
				if end > close {
					for _, field := range parseFieldList(file, close+1, end-1) {
						if field.NameTok == token {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

// An embedded alias contributes its own field name even though its type is
// identical to the alias target. Preserve that distinction when concrete type
// spellings replace generic names, including anonymous struct identities.
func (s *genericSpecializer) embeddedTypeText(typ int, name string, origin string, into int) string {
	e := s.environment
	prefix := ""
	if e.types.get(typ).kind == genericPointer {
		prefix = "*"
		typ = e.types.get(typ).elem
	}
	owner := 0
	for i, pkg := range e.graph.Packages {
		if pkg.Ref.ImportPath == origin {
			owner = i + 1
			break
		}
	}
	// Each field identity needs one bridge declaration for the whole graph.
	// Duplicating it in callers causes package symbol disambiguation to rename
	// the embedded field while its selectors and literal keys retain the old
	// spelling. Keep private fields in their declaration package; otherwise
	// prefer the embedded named type's owner.
	bridgePackage := into
	if owner > 0 {
		bridgePackage = owner - 1
	} else {
		base := e.types.get(typ)
		for _, decl := range e.decls {
			if decl.kind == SymbolType && !decl.alias && e.types.get(decl.typ).origin == base.origin && base.kind == genericNamed {
				bridgePackage = decl.pkg
				break
			}
		}
	}
	text := s.typeText(typ, bridgePackage)
	// Even an unchanged source spelling can be renamed later by package
	// linking. One graph-wide bridge keeps embedding, selectors and literal
	// keys on the same spelling through that namespace change.
	alias := "RenvoGenericEmbedded_" + name + "_" + genericDecimal(typ) + "_" + genericDecimal(owner)
	for {
		used := false
		for pkg := range e.graph.Packages {
			if s.nameUsed(pkg, alias) {
				used = true
				break
			}
		}
		if !used {
			break
		}
		alias += "_"
	}
	for _, bridge := range s.bridges {
		if bridge.name == alias {
			return prefix + s.qualify(into, bridge.pkg, alias)
		}
	}
	s.bridges = append(s.bridges, genericBridge{pkg: bridgePackage, name: alias, target: text, embeddedName: name})
	return prefix + s.qualify(into, bridgePackage, alias)
}

func (c *genericExpressionContext) rewriteEmbeddedType(token int) bool {
	e := c.specializer.environment
	file := &e.graph.Packages[c.scope.pkg].Files[c.scope.file].File
	first := token
	if tokCharIs(file, first-1, '*') {
		first--
	}
	// An embedded type starts a field, unlike named field types and ordinary
	// references elsewhere in a body. Avoid walking those scopes backwards.
	if !tokCharIs(file, first-1, '{') && !tokCharIs(file, first-1, ';') && (first <= 0 || (file.Tokens[first-1].KindLine>>syntax.TokenOperatorLineShift&syntax.TokenLineLimit) == (file.Tokens[first].KindLine>>syntax.TokenOperatorLineShift&syntax.TokenLineLimit)) {
		return false
	}
	depth := 0
	for open := token - 1; open >= 1; open-- {
		if tokCharIs(file, open, '}') {
			depth++
		}
		if !tokCharIs(file, open, '{') {
			continue
		}
		if depth > 0 {
			depth--
			continue
		}
		if file.Tokens[open-1].KindLine&255 != syntax.TokenStruct {
			return false
		}
		close := findTypeMatching(file, open, '{', '}')
		if close <= token {
			return false
		}
		for _, field := range parseStructFields(file, open+1, close-1) {
			first := field.TypeStart
			if tokCharIs(file, first, '*') {
				first++
			}
			if field.NameTok >= 0 || first != token {
				continue
			}
			nameToken := first
			if tokCharIs(file, first+1, '.') {
				nameToken += 2
			}
			name := tokenString(file, nameToken)
			origin := ""
			if !syntax.IdentifierExported([]byte(name), 0) {
				origin = e.graph.Packages[c.scope.pkg].Ref.ImportPath
			}
			text := c.specializer.embeddedTypeText(c.typeSpan(field.TypeStart, field.TypeEnd), name, origin, c.scope.pkg)
			c.replace(field.TypeStart, field.TypeEnd, text)
			return true
		}
		return false
	}
	return false
}
