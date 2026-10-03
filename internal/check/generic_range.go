package check

import "renvo.dev/internal/syntax"

func (e *genericEnvironment) rangeTypes(value genericArgument) ([]int, bool) {
	id := value.typ
	if value.untyped != 0 {
		id = e.types.defaultType(value.untyped)
	}
	v := e.types.get(e.coreType(id))
	if v.kind == genericPointer {
		v = e.types.get(e.coreType(v.elem))
		if v.kind != genericArray {
			return nil, false
		}
	}
	switch v.kind {
	case genericSlice, genericArray:
		return []int{e.types.basic("int"), v.elem}, true
	case genericMap:
		return []int{v.key, v.elem}, true
	case genericChan:
		return []int{v.elem}, v.direction != ChanSendOnly
	case genericBasic:
		if v.name == "string" {
			return []int{e.types.basic("int"), e.types.basic("rune")}, true
		}
		return []int{id}, genericInteger(v.name)
	case genericFunc:
		if len(v.params) != 1 || len(v.results) != 0 || v.variadic {
			return nil, false
		}
		yield := e.types.get(e.coreType(v.params[0]))
		if yield.kind != genericFunc || len(yield.params) > 2 || len(yield.results) != 1 || yield.results[0] != e.types.basic("bool") {
			return nil, false
		}
		return yield.params, true
	}
	return nil, false
}

func (c *genericExpressionContext) validateFor(stmt syntax.Stmt) {
	e := c.specializer.environment
	file := &e.graph.Packages[c.scope.pkg].Files[c.scope.file].File
	// Keep empty init/post clauses: the syntax expression span trims their
	// leading and trailing semicolons.
	stmt.ExprStart, stmt.ExprEnd = stmt.StartTok+1, stmt.BodyStart
	if i := findTypeTopLevelKind(file, stmt.ExprStart, stmt.ExprEnd, syntax.TokenRange); i >= 0 {
		value := c.expression(i+1, stmt.ExprEnd, i+1)
		types, ok := e.rangeTypes(value)
		if !ok {
			e.fail(c.scope, i, "range is not valid for every type in the constraint")
			return
		}
		rangeType := value.typ
		if value.untyped != 0 {
			rangeType = e.types.defaultType(value.untyped)
		}
		core := e.types.get(e.coreType(rangeType))
		if core.kind == genericFunc {
			e.requireVersion(c.scope, i, "1.23", "range over functions")
		} else if core.kind == genericBasic && genericInteger(core.name) {
			e.requireVersion(c.scope, i, "1.22", "range over integers")
		}
		if i > stmt.ExprStart {
			left := splitExprList(file, stmt.ExprStart, i-1)
			if len(left) > len(types) {
				e.fail(c.scope, i, "too many range variables")
			}
			if tokenTextIs(file, i-1, ":=") {
				declares := false
				for j, span := range left {
					if span.EndTok != span.StartTok+1 || file.Tokens[span.StartTok].KindLine&255 != syntax.TokenIdent {
						e.fail(c.scope, span.StartTok, "range declaration must use identifiers")
						continue
					}
					if !tokenTextIs(file, span.StartTok, "_") {
						declares = true
						for k := 0; k < j; k++ {
							if tokenString(file, left[k].StartTok) == tokenString(file, span.StartTok) {
								e.fail(c.scope, span.StartTok, "duplicate range variable")
							}
						}
					}
				}
				if !declares {
					e.fail(c.scope, i-1, "range declaration has no new variables")
				}
			}
			if tokenTextIs(file, i-1, "=") {
				for j, span := range left {
					if j < len(types) && !tokenTextIs(file, span.StartTok, "_") && !e.argumentAssignable(genericArgument{typ: types[j]}, c.expression(span.StartTok, span.EndTok, i).typ) {
						e.fail(c.scope, i, "range variable is not assignable")
					}
				}
			}
		}
		return
	}
	start, end := stmt.ExprStart, stmt.ExprEnd
	if semi := findTypeTopLevelChar(file, start, end, ';'); semi >= 0 {
		c.validateSimpleExpression(start, semi)
		start = semi + 1
		end = findTypeTopLevelChar(file, start, end, ';')
		if end >= 0 {
			c.validateSimpleExpression(end+1, stmt.ExprEnd)
		}
	}
	if end > start {
		condition := c.expression(start, end, start)
		if condition.untyped != genericUntypedBool && !e.allowsOperation(condition.typ, "!") {
			e.fail(c.scope, start, "for condition must be boolean")
		}
	}
}
