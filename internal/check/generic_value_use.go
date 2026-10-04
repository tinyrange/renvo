package check

import "renvo.dev/internal/syntax"

// Ordinary bodies still need the distinction between value types and constraint
// interfaces, even when no generic declaration is instantiated. Inspect their
// expression spans without performing specialization or evaluating operands.
func (c *genericExpressionContext) validateValueTypeBody() {
	e := c.specializer.environment
	file := &e.graph.Packages[c.scope.pkg].Files[c.scope.file].File
	if !c.needsValueTypeCheck(c.function.BodyStart+1, c.function.BodyEnd-1) {
		return
	}
	body := syntax.ParseFuncBodyStatements(*file, c.function)
	for _, stmt := range body.Stmts {
		switch stmt.Kind {
		case syntax.StmtDecl, syntax.StmtBlock:
			// Type declarations are allowed to denote constraints. Variable
			// types and initializer expressions are handled by the bindings.
		case syntax.StmtIf, syntax.StmtFor, syntax.StmtSwitch:
			c.validateValueTypeExpression(stmt.StartTok+1, stmt.BodyStart)
		case syntax.StmtAssign:
			c.validateValueTypeExpression(stmt.StartTok, stmt.EndTok)
		default:
			c.validateValueTypeExpression(stmt.ExprStart, stmt.ExprEnd)
		}
	}
	for _, binding := range c.bindings {
		if binding.name < c.function.BodyStart || binding.name >= c.function.BodyEnd {
			continue
		}
		if binding.typeStart >= 0 && binding.typeEnd > binding.typeStart && !e.valueType(c.typeSpan(binding.typeStart, binding.typeEnd)) {
			e.fail(c.scope, binding.typeStart, "constraint interface cannot be used as a value type")
		}
		c.validateValueTypeExpression(binding.valueStart, binding.valueEnd)
	}
}

func (c *genericExpressionContext) validateValueTypeExpression(start, end int) {
	e := c.specializer.environment
	file := &e.graph.Packages[c.scope.pkg].Files[c.scope.file].File
	if !c.needsValueTypeCheck(start, end) {
		return
	}
	if start >= 0 && start < end {
		if op := findTopLevelAssignOp(file, start, end); op >= 0 && tokenTextIs(file, op, ":=") {
			start = op + 1
		}
	}
	for i := start; i >= 0 && i < end && e.errorText == ""; i++ {
		if file.Tokens[i].KindLine&255 == syntax.TokenFunc {
			fn := genericFunctionLiteral(file, i, end)
			if fn.BodyStart >= 0 {
				inner := newGenericExpressionContext(c.specializer, c.scope, fn, c.arguments)
				inner.specialize = false
				inner.bindings = append(append([]scopedTypeBinding(nil), c.bindings...), inner.bindings...)
				e.signature(c.scope, buildFuncSignature(file, &fn))
				inner.validateValueTypeBody()
				i = fn.BodyEnd - 1
				continue
			}
		}
		// Selectors and literal keys can share a spelling with a type without
		// referring to it. Qualified types are recognized from the package name.
		if tokCharIs(file, i-1, '.') || tokCharIs(file, i+1, ':') {
			continue
		}
		last := c.valueExpressionTypeEnd(i, end)
		if last <= i {
			continue
		}
		if !e.valueType(c.typeSpan(i, last)) {
			e.fail(c.scope, i, "constraint interface cannot be used as a value type")
		}
		i = last - 1
	}
}

func (c *genericExpressionContext) needsValueTypeCheck(start, end int) bool {
	e := c.specializer.environment
	if e.types.nonbasic || len(c.scope.parameters) != 0 {
		return true
	}
	file := &e.graph.Packages[c.scope.pkg].Files[c.scope.file].File
	for i := start; i >= 0 && i < end; i++ {
		if file.Tokens[i].KindLine&255 == syntax.TokenInterface || tokenTextIs(file, i, "comparable") && !tokCharIs(file, i-1, '.') && c.binding(i, i) < 0 {
			return true
		}
	}
	return false
}

// Recognize a type before parsing it, so ordinary function calls and values do
// not produce speculative undefined-type errors. This consumes the complete
// type, including constraints nested under pointer and container types.
func (c *genericExpressionContext) valueExpressionTypeEnd(start, end int) int {
	e := c.specializer.environment
	file := &e.graph.Packages[c.scope.pkg].Files[c.scope.file].File
	if start < 0 || start >= end {
		return -1
	}
	if tokCharIs(file, start, '(') {
		close := findTypeMatching(file, start, '(', ')')
		if close > start && close <= end && c.valueExpressionTypeEnd(start+1, close-1) == close-1 {
			return close
		}
		return -1
	}
	if tokCharIs(file, start, '*') {
		return c.valueExpressionTypeEnd(start+1, end)
	}
	if tokCharIs(file, start, '[') {
		// An inferred array literal is not a type until its elements have
		// supplied the length. Its element type is inspected separately.
		if tokenTextIs(file, start+1, "...") {
			return -1
		}
		close := findTypeMatching(file, start, '[', ']')
		if close > start && close < end {
			return c.valueExpressionTypeEnd(close, end)
		}
		return -1
	}
	kind := file.Tokens[start].KindLine & 255
	if kind == syntax.TokenMap {
		close := findTypeMatching(file, start+1, '[', ']')
		if close > start+2 && close < end && c.valueExpressionTypeEnd(start+2, close-1) == close-1 {
			return c.valueExpressionTypeEnd(close, end)
		}
		return -1
	}
	if kind == syntax.TokenChan || tokenTextIs(file, start, "<-") && start+1 < end && file.Tokens[start+1].KindLine&255 == syntax.TokenChan {
		next := start + 1
		if tokenTextIs(file, start, "<-") || tokenTextIs(file, next, "<-") {
			next++
		}
		return c.valueExpressionTypeEnd(next, end)
	}
	if kind == syntax.TokenInterface || kind == syntax.TokenStruct {
		if !tokCharIs(file, start+1, '{') {
			return -1
		}
		close := findTypeMatching(file, start+1, '{', '}')
		if close > start+1 && close <= end {
			return close
		}
		return -1
	}
	if kind == syntax.TokenFunc && tokCharIs(file, start+1, '(') {
		close := findTypeMatching(file, start+1, '(', ')')
		if close <= start+1 || close > end {
			return -1
		}
		if tokCharIs(file, close, '(') {
			results := findTypeMatching(file, close, '(', ')')
			if results > close && results <= end {
				return results
			}
		}
		if result := c.valueExpressionTypeEnd(close, end); result > close {
			return result
		}
		return close
	}
	if kind != syntax.TokenIdent || c.binding(start, start) >= 0 {
		return -1
	}
	declarationEnd := end
	if tokCharIs(file, start+1, '.') && e.imported(c.scope, tokenString(file, start)) < 0 {
		declarationEnd = start + 1
	}
	d, next := c.declaration(start, declarationEnd)
	if d >= 0 {
		if e.decls[d].kind != SymbolType {
			return -1
		}
		if len(e.decls[d].parameters) == 0 {
			return next
		}
		if tokCharIs(file, next, '[') {
			close := findTypeMatching(file, next, '[', ']')
			if close > next && close <= end {
				return close
			}
		}
		return -1
	}
	name := tokenString(file, start)
	for _, parameter := range c.scope.parameters {
		if e.types.get(parameter).name == name {
			return start + 1
		}
	}
	if genericBasicName(name) || name == "any" || name == "error" || name == "comparable" {
		for _, source := range e.graph.Packages[c.scope.pkg].Files {
			for _, decl := range source.File.Decls {
				if decl.Kind != syntax.TokenType && tokenStringEquals(&source.File, decl.NameTok, name) {
					return -1
				}
			}
		}
		return start + 1
	}
	return -1
}
