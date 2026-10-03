package check

import "renvo.dev/internal/syntax"

func genericClauseOwner(body *syntax.Body, index int) int {
	for i := index - 1; i >= 0; i-- {
		stmt := body.Stmts[i]
		if (stmt.Kind == syntax.StmtSwitch || stmt.Kind == syntax.StmtSelect) && stmt.BodyStart < body.Stmts[index].StartTok && body.Stmts[index].StartTok < stmt.BodyEnd {
			return i
		}
	}
	return -1
}

func (c *genericExpressionContext) switchBindings(body *syntax.Body) {
	file := &c.specializer.environment.graph.Packages[c.scope.pkg].Files[c.scope.file].File
	ends := localRuleScopeEnds(body)
	for i, clause := range body.Stmts {
		if clause.Kind != syntax.StmtCase && clause.Kind != syntax.StmtDefault {
			continue
		}
		owner := genericClauseOwner(body, i)
		if owner < 0 {
			continue
		}
		stmt := body.Stmts[owner]
		if stmt.Kind != syntax.StmtSwitch || !tokenTextIs(file, stmt.ExprEnd-2, "type") {
			continue
		}
		start := stmt.ExprStart
		if semi := findTypeTopLevelChar(file, start, stmt.ExprEnd, ';'); semi >= 0 {
			start = semi + 1
		}
		op := findTopLevelAssignOp(file, start, stmt.ExprEnd)
		if op != start+1 || !tokenTextIs(file, op, ":=") {
			continue
		}
		binding := scopedTypeBinding{name: start, visible: clause.EndTok, end: ends[i], typeStart: -1, typeEnd: -1, valueStart: op + 1, valueEnd: stmt.ExprEnd - 4, writable: true}
		if clause.Kind == syntax.StmtCase {
			cases := splitExprList(file, clause.ExprStart, clause.ExprEnd)
			if len(cases) == 1 && !tokenTextIs(file, cases[0].StartTok, "nil") {
				binding.typeStart, binding.typeEnd = cases[0].StartTok, cases[0].EndTok
				binding.valueStart, binding.valueEnd = -1, -1
			}
		}
		c.bindings = append(c.bindings, binding)
	}
}

func (c *genericExpressionContext) validateSwitch(stmt syntax.Stmt) {
	e := c.specializer.environment
	file := &e.graph.Packages[c.scope.pkg].Files[c.scope.file].File
	start, end := stmt.ExprStart, stmt.ExprEnd
	if semi := findTypeTopLevelChar(file, start, end, ';'); semi >= 0 {
		start = semi + 1
	}
	if tokenTextIs(file, end-2, "type") {
		if op := findTopLevelAssignOp(file, start, end); op >= 0 {
			start = op + 1
		}
		value := c.expression(start, end-4, start)
		if e.types.get(value.typ).kind == genericParameter || e.types.get(e.coreType(value.typ)).kind != genericInterface {
			e.fail(c.scope, start, "type switch requires an interface value")
		}
	} else if start < end {
		value := c.expression(start, end, start)
		if value.untyped != 0 {
			c.lowerExpectedConstant(value, e.types.defaultType(value.untyped))
		}
		if e.types.get(value.typ).kind == genericParameter && !e.allowsOperation(value.typ, "==") {
			e.fail(c.scope, start, "switch expression must be comparable")
		}
	}
}

func (c *genericExpressionContext) validateSwitchCase(owner syntax.Stmt, clause syntax.Stmt) {
	e := c.specializer.environment
	file := &e.graph.Packages[c.scope.pkg].Files[c.scope.file].File
	start, end := owner.ExprStart, owner.ExprEnd
	if tokenTextIs(file, end-2, "type") {
		return
	}
	if semi := findTypeTopLevelChar(file, start, end, ';'); semi >= 0 {
		start = semi + 1
	}
	target := e.types.basic("bool")
	if start < end {
		value := c.expression(start, end, start)
		target = value.typ
		if target == 0 {
			target = e.types.defaultType(value.untyped)
		}
	}
	for _, span := range splitExprList(file, clause.ExprStart, clause.ExprEnd) {
		value := c.expression(span.StartTok, span.EndTok, span.StartTok)
		if !e.argumentAssignable(value, target) {
			e.fail(c.scope, span.StartTok, "switch case is not assignable to the switch expression")
		}
		c.lowerExpectedConstant(value, target)
	}
}
