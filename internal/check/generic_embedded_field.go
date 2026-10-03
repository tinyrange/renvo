package check

import "renvo.dev/internal/syntax"

func genericPrimaryStart(file *syntax.File, end int) int {
	if end < 0 || end >= len(file.Tokens) {
		return -1
	}
	start := end
	if tokCharIs(file, end, ']') || tokCharIs(file, end, ')') || tokCharIs(file, end, '}') {
		open, close := byte('['), byte(']')
		if tokCharIs(file, end, ')') {
			open, close = '(', ')'
		} else if tokCharIs(file, end, '}') {
			open, close = '{', '}'
		}
		start = genericMatchingOpen(file, 0, end, open, close)
		if start <= 0 {
			return start
		}
		if open != '(' || (file.Tokens[start-1].KindLine>>syntax.TokenOperatorLineShift&syntax.TokenLineLimit) == (file.Tokens[start].KindLine>>syntax.TokenOperatorLineShift&syntax.TokenLineLimit) && (file.Tokens[start-1].KindLine&255 == syntax.TokenIdent || file.Tokens[start-1].KindLine&255 == syntax.TokenFunc || tokCharIs(file, start-1, ')') || tokCharIs(file, start-1, ']')) {
			start = genericPrimaryStart(file, start-1)
		}
	}
	for start >= 2 && tokCharIs(file, start-1, '.') {
		start = genericPrimaryStart(file, start-2)
	}
	return start
}

func (c *genericExpressionContext) embeddedFieldName(typ int, name string) string {
	e := c.specializer.environment
	for _, field := range e.types.promotedMembers(typ, true) {
		if field.name == name && field.embedded {
			text := c.specializer.embeddedTypeText(field.typ, field.name, field.pkg, c.scope.pkg)
			start := 0
			for i := 0; i < len(text); i++ {
				if text[i] == '.' || text[i] == '*' {
					start = i + 1
				}
			}
			return text[start:]
		}
	}
	return ""
}

// Concrete embedded type spellings acquire specialization names. Preserve
// source selectors and keyed literals by resolving the original field identity
// before emitting the corresponding concrete selector.
func (c *genericExpressionContext) rewriteEmbeddedField(token int) {
	e := c.specializer.environment
	file := &e.graph.Packages[c.scope.pkg].Files[c.scope.file].File
	name := tokenString(file, token)
	candidate := genericBasicName(name) || name == "any" || name == "error"
	for declIndex := range e.decls {
		d := &e.decls[declIndex]
		if d.kind == SymbolType && d.name == name {
			candidate = true
			break
		}
	}
	if !candidate {
		return
	}
	typ := 0
	if tokCharIs(file, token-1, '.') {
		start := genericPrimaryStart(file, token-2)
		if start >= 0 {
			typ = c.expression(start, token-1, token).typ
		}
	} else if tokCharIs(file, token+1, ':') {
		depth := 0
		for i := token - 1; i >= 0; i-- {
			if tokCharIs(file, i, '}') {
				depth++
			}
			if tokCharIs(file, i, '{') {
				if depth == 0 {
					start := genericPrimaryStart(file, i-1)
					if start >= 0 {
						typ = c.expressionType(start, i)
					}
					break
				}
				depth--
			}
		}
	}
	if typ != 0 {
		if replacement := c.embeddedFieldName(typ, name); replacement != "" && replacement != name {
			c.replace(token, token+1, replacement)
		}
	}
}
