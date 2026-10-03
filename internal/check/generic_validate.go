package check

import "renvo.dev/internal/syntax"

func (e *genericEnvironment) argumentAssignable(value genericArgument, target int) bool {
	if target == 0 {
		return false
	}
	if value.untyped != 0 {
		if value.untyped == genericUntypedNil {
			return e.allowsOperation(target, "nil")
		}
		if e.types.get(target).kind == genericParameter {
			constraint, ok := e.parameterConstraint(target)
			if !ok || constraint.all || len(constraint.terms) == 0 {
				return false
			}
			for _, term := range constraint.terms {
				if !e.argumentAssignable(value, term.typ) {
					return false
				}
			}
			return true
		}
		if len(value.shifted) != 0 {
			shiftType := target
			if e.types.get(e.types.underlying(target)).kind == genericInterface {
				shiftType = e.types.defaultType(value.untyped)
			}
			if !e.allowsOperation(shiftType, "<<") {
				return false
			}
			for _, constant := range value.shifted {
				if !e.constantFits(genericArgument{constant: constant}, shiftType) {
					return false
				}
			}
		}
		return e.untypedAssignable(value.untyped, target) && e.constantFits(value, target)
	}
	if value.typ == target {
		return true
	}
	if e.types.get(value.typ).kind == genericParameter {
		v := e.types.get(e.types.underlying(target))
		if v.kind == genericInterface {
			return e.satisfies(value.typ, e.types.interfaceConstraint(target))
		}
		if e.types.get(target).kind == genericNamed || e.types.get(target).kind == genericBasic || e.types.get(target).kind == genericParameter {
			return false
		}
		constraint, ok := e.parameterConstraint(value.typ)
		if !ok || constraint.all || len(constraint.terms) == 0 {
			return false
		}
		for _, term := range constraint.terms {
			if !e.types.assignable(term.typ, target) {
				return false
			}
		}
		return true
	}
	if e.types.get(target).kind == genericParameter {
		kind := e.types.get(value.typ).kind
		if kind == genericNamed || kind == genericBasic {
			return false
		}
		constraint, ok := e.parameterConstraint(target)
		if !ok || constraint.all || len(constraint.terms) == 0 {
			return false
		}
		for _, term := range constraint.terms {
			if !e.types.assignable(value.typ, term.typ) {
				return false
			}
		}
		return true
	}
	if e.types.get(e.types.underlying(target)).kind == genericInterface {
		return e.satisfies(value.typ, e.types.interfaceConstraint(target))
	}
	return e.types.assignable(value.typ, target)
}

func (e *genericEnvironment) untypedAssignable(kind int, target int) bool {
	v := e.types.get(e.types.underlying(target))
	if v.kind == genericInterface {
		return !v.restricted && !v.comparable && len(v.methods) == 0
	}
	if v.kind != genericBasic {
		return false
	}
	switch kind {
	case genericUntypedBool:
		return v.name == "bool"
	case genericUntypedString:
		return v.name == "string"
	case genericUntypedInt, genericUntypedRune, genericUntypedFloat, genericUntypedComplex:
		return genericNumeric(v.name)
	}
	return false
}

// Validate definitions before considering their concrete uses. In particular,
// an unused generic function must not hide invalid parameter operations or
// return values behind the absence of an instantiation.
func (s *genericSpecializer) validateDefinition(index int) {
	e := s.environment
	e.resolveDeclaration(index)
	d := &e.decls[index]
	if d.kind == SymbolType || len(d.parameters) == 0 {
		return
	}
	e.validateParameterNames(d)
	e.validateGenericScope(d)
	if d.name == "init" || d.name == "main" && e.graph.Packages[d.pkg].Name == "main" {
		e.fail(genericTypeScope{pkg: d.pkg, file: d.file}, d.token, "main and init cannot have type parameters")
		return
	}
	if d.function.BodyStart < 0 {
		e.fail(genericTypeScope{pkg: d.pkg, file: d.file}, d.token, "generic function requires a body")
		return
	}
	ctx := newGenericExpressionContext(s, genericTypeScope{pkg: d.pkg, file: d.file, parameters: d.parameters}, d.function, nil)
	ctx.specialize = false
	ctx.validateBody(e.types.get(d.typ))
}

func (ctx *genericExpressionContext) validateBody(signature *genericType) {
	e := ctx.specializer.environment
	ctx.validateValueTypeBody()
	file := &e.graph.Packages[ctx.scope.pkg].Files[ctx.scope.file].File
	body := syntax.ParseFuncBodyStatements(*file, ctx.function)
	if !body.Ok {
		e.fail(ctx.scope, body.ErrorTok, "invalid function body")
		return
	}
	if len(signature.results) > 0 && !returnBlockTerminates(file, &body, ctx.function.BodyStart+1, ctx.function.BodyEnd-1, e.lookup(ctx.scope.pkg, "panic") < 0) {
		e.fail(ctx.scope, ctx.function.BodyEnd-1, "missing return in function")
		return
	}
	for index, stmt := range body.Stmts {
		switch stmt.Kind {
		case syntax.StmtSwitch:
			ctx.validateSwitch(stmt)
		case syntax.StmtCase:
			owner := genericClauseOwner(&body, index)
			if owner >= 0 && body.Stmts[owner].Kind == syntax.StmtSwitch {
				ctx.validateSwitchCase(body.Stmts[owner], stmt)
			}
			if owner >= 0 && body.Stmts[owner].Kind == syntax.StmtSelect {
				start := stmt.ExprStart
				if op := findTopLevelAssignOp(file, start, stmt.ExprEnd); op >= 0 {
					start = op + 1
				}
				ctx.validateSimpleExpression(start, stmt.ExprEnd)
			}
		case syntax.StmtFor:
			ctx.validateFor(stmt)
		case syntax.StmtReturn:
			values := splitExprList(file, stmt.ExprStart, stmt.ExprEnd)
			if len(values) == 0 {
				for _, result := range buildFuncSignature(file, &ctx.function).Results {
					if result.Name == "" || result.Name == "_" {
						e.fail(ctx.scope, stmt.StartTok, "bare return requires named results")
					}
				}
				continue
			}
			actual := ctx.expressionValues(values, stmt.StartTok)
			if len(actual) != len(signature.results) {
				e.fail(ctx.scope, stmt.StartTok, "wrong number of return values")
				continue
			}
			for i, value := range actual {
				if value.function != 0 {
					value = ctx.functionValue(value.function-1, value.start, value.end, signature.results[i])
				}
				if !e.argumentAssignable(value, signature.results[i]) {
					e.fail(ctx.scope, stmt.StartTok, "return value is not valid for every type in the constraint")
				}
				ctx.lowerExpectedConstant(value, signature.results[i])
			}
		case syntax.StmtAssign:
			ctx.validateAssignment(stmt.StartTok, stmt.EndTok)
		case syntax.StmtIf:
			start := stmt.ExprStart
			if semi := findTypeTopLevelChar(file, start, stmt.ExprEnd, ';'); semi >= 0 {
				ctx.validateSimpleExpression(start, semi)
				start = semi + 1
			}
			condition := ctx.expression(start, stmt.ExprEnd, start)
			if condition.untyped != genericUntypedBool && !e.allowsOperation(condition.typ, "!") {
				e.fail(ctx.scope, start, "condition type parameter must be boolean")
			}
		case syntax.StmtExpr, syntax.StmtDefer, syntax.StmtGo:
			ctx.validateSimpleExpression(stmt.ExprStart, stmt.ExprEnd)
		}
	}
	for _, binding := range ctx.bindings {
		if binding.valueStart >= 0 && file.Tokens[binding.valueStart].KindLine&255 == syntax.TokenRange {
			// validateFor checks the operand and rangeTypes supplies these
			// bindings. The header is not an ordinary variable initializer.
			continue
		}
		if binding.valueStart < 0 || tokenKindIs(file, binding.valueEnd-2, syntax.TokenType) {
			ctx.validateTupleInitializer(binding)
			continue
		}
		value := ctx.expression(binding.valueStart, binding.valueEnd, binding.name)
		if binding.constant && value.constant == nil {
			e.fail(ctx.scope, binding.valueStart, "constant initializer must be a constant expression")
		}
		target := 0
		if binding.typeEnd > binding.typeStart {
			target = ctx.typeSpan(binding.typeStart, binding.typeEnd)
		} else if !binding.constant && value.untyped != 0 {
			target = e.types.defaultType(value.untyped)
		} else {
			continue
		}
		if value.function != 0 {
			value = ctx.functionValue(value.function-1, value.start, value.end, target)
		}
		if !e.argumentAssignable(value, target) {
			e.fail(ctx.scope, binding.valueStart, "initializer is not valid for every type in the constraint")
		}
		ctx.lowerExpectedConstant(value, target)
	}
	// Calls in control-flow headers and declarations still contribute to the
	// instantiation graph even when their result is not otherwise needed.
	ctx.scan(ctx.function.BodyStart+1, ctx.function.BodyEnd-1)
}

func (ctx *genericExpressionContext) validateTupleInitializer(binding scopedTypeBinding) {
	e := ctx.specializer.environment
	for _, tuple := range ctx.tuples {
		if tuple.name != binding.name {
			continue
		}
		value := ctx.expression(tuple.start, tuple.end, binding.name)
		count := len(value.results)
		if count == 0 && value.commaOK {
			count = 2
			if tuple.index == 1 {
				value = genericArgument{typ: e.types.basic("bool")}
			}
		} else if tuple.index < count {
			value = genericArgument{typ: value.results[tuple.index]}
		}
		if count != tuple.count {
			e.fail(ctx.scope, tuple.start, "initializer count mismatch")
			return
		}
		if binding.typeEnd > binding.typeStart && !e.argumentAssignable(value, ctx.typeSpan(binding.typeStart, binding.typeEnd)) {
			e.fail(ctx.scope, tuple.start, "initializer is not valid for every type in the constraint")
		}
		return
	}
}

func (ctx *genericExpressionContext) validateAssignment(start int, end int) {
	e := ctx.specializer.environment
	file := &e.graph.Packages[ctx.scope.pkg].Files[ctx.scope.file].File
	op := findTopLevelAssignOp(file, start, end)
	if op < 0 {
		return
	}
	values := splitExprList(file, op+1, end)
	left := splitExprList(file, start, op)
	actual := ctx.expressionValues(values, start)
	if len(left) == 2 && len(actual) == 1 && actual[0].commaOK {
		actual = append(actual, genericArgument{typ: e.types.basic("bool")})
	}
	if len(left) != len(actual) {
		e.fail(ctx.scope, start, "assignment count mismatch")
	}
	for i, value := range actual {
		if i >= len(left) {
			continue
		}
		newVariable := false
		if tokenTextIs(file, op, ":=") {
			for _, binding := range ctx.bindings {
				if binding.name == left[i].StartTok {
					newVariable = true
				}
			}
		}
		if newVariable || tokenTextIs(file, left[i].StartTok, "_") {
			if value.untyped != 0 && !e.argumentAssignable(value, e.types.defaultType(value.untyped)) {
				e.fail(ctx.scope, start, "value cannot use its default type")
			}
			if value.untyped != 0 {
				ctx.lowerExpectedConstant(value, e.types.defaultType(value.untyped))
			}
		} else {
			if !ctx.assignableLocation(left[i].StartTok, left[i].EndTok, start) {
				e.fail(ctx.scope, start, "assignment target is not writable")
			}
			target := ctx.expression(left[i].StartTok, left[i].EndTok, start)
			ctx.lowerExpectedConstant(value, target.typ)
			operation := tokenString(file, op)
			if operation != "=" && operation != ":=" && len(operation) > 1 {
				operation = operation[:len(operation)-1]
				var ok bool
				value, ok = e.binaryOperation(target, value, operation)
				if !ok {
					e.fail(ctx.scope, op, "compound assignment is not valid for the constraint")
				}
			}
			if value.function != 0 && target.typ != 0 {
				value = ctx.functionValue(value.function-1, value.start, value.end, target.typ)
			}
			if target.typ != 0 && !e.argumentAssignable(value, target.typ) {
				e.fail(ctx.scope, start, "assignment is not valid for every type in the constraint")
			}
		}
	}
}
