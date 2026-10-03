package check

import "renvo.dev/internal/syntax"

func (c *genericExpressionContext) replacedToken(token int) bool {
	file := &c.specializer.environment.graph.Packages[c.scope.pkg].Files[c.scope.file].File
	for _, change := range c.changes {
		if change.start <= int(file.Tokens[token].Start) && int(file.Tokens[token].End) <= change.end {
			return true
		}
	}
	return false
}

func (c *genericExpressionContext) containsFunctionValue(start int, end int) bool {
	for i := start; i < end; i++ {
		if d, _ := c.declaration(i, end); d >= 0 && c.specializer.environment.decls[d].kind == SymbolFunc && len(c.specializer.environment.decls[d].parameters) > 0 {
			return true
		}
	}
	return false
}

func (c *genericExpressionContext) functionArguments(signature *genericType, actual []genericArgument, spread bool, token int) {
	e := c.specializer.environment
	if signature.kind != genericFunc {
		return
	}
	if !signature.variadic && (spread || len(actual) != len(signature.params)) || signature.variadic && (len(actual) < len(signature.params)-1 || spread && len(actual) != len(signature.params)) {
		e.fail(c.scope, token, "wrong number of call arguments")
		return
	}
	for i, arg := range actual {
		parameter := genericArgumentParameter(&e.types, signature, i, spread)
		if arg.function != 0 {
			arg = c.functionValue(arg.function-1, arg.start, arg.end, parameter)
		}
		if !e.argumentAssignable(arg, parameter) {
			e.fail(c.scope, token, "call argument is not assignable for every type in the constraint")
		}
		c.lowerExpectedConstant(arg, parameter)
	}
}

// Function values are instantiated against the assignment's expected signature.
// A call uses the same path after its other arguments constrain its parameters.
func (c *genericExpressionContext) functionValue(declaration int, start int, end int, expected int) genericArgument {
	e := c.specializer.environment
	if !e.requireVersion(c.scope, start, "1.18", "function instantiation") {
		return genericArgument{}
	}
	e.resolveDeclaration(declaration)
	d := &e.decls[declaration]
	file := &e.graph.Packages[c.scope.pkg].Files[c.scope.file].File
	_, next := c.declaration(start, end)
	var explicit []int
	if tokCharIs(file, next, '[') {
		for i := next + 1; i < end-1; {
			last := nextTopLevelComma(file, i, end-1)
			explicit = append(explicit, c.typeSpan(i, last))
			i = last + 1
		}
	}
	if len(explicit) < len(d.parameters) && !e.requireVersion(c.scope, start, "1.21", "generic function value inference") {
		return genericArgument{}
	}
	parameters, constraints, signature := c.inferenceDeclaration(d)
	arguments, ok := e.types.inferArgumentsVersion(parameters, constraints, signature, explicit, nil, false, expected, e.versionBefore(c.scope, "1.21"))
	if !ok {
		e.fail(c.scope, start, "cannot infer generic function value")
		return genericArgument{}
	}
	for i, arg := range arguments {
		if !e.argumentSatisfies(c.scope, arg, e.types.substituteConstraint(d.constraints[i], d.parameters, arguments)) {
			e.fail(c.scope, start, "function value type argument does not satisfy constraint")
			return genericArgument{}
		}
	}
	e.recordInstantiation(c.scope, start, d.parameters, arguments)
	if c.specialize {
		instance := c.specializer.instantiate(declaration, arguments)
		if instance < 0 {
			return genericArgument{}
		}
		name := c.specializer.qualify(c.scope.pkg, d.pkg, c.specializer.instances[instance].name)
		if d.pkg != c.scope.pkg && tokCharIs(file, start+1, '.') {
			name = tokenString(file, start) + "." + c.specializer.instances[instance].name
		}
		c.replace(start, end, name)
	}
	return genericArgument{typ: e.types.substitute(d.typ, d.parameters, arguments)}
}

func (c *genericExpressionContext) expectedValueType(start int, end int) int {
	for _, binding := range c.bindings {
		if binding.valueStart == start && binding.valueEnd == end && binding.typeEnd > binding.typeStart {
			return c.typeSpan(binding.typeStart, binding.typeEnd)
		}
	}
	e := c.specializer.environment
	file := &e.graph.Packages[c.scope.pkg].Files[c.scope.file].File
	for _, decl := range file.Decls {
		info := buildDeclInfo(file, c.scope.file, PackageInfo{}, nil, decl)
		if info.TypeEnd <= info.TypeStart {
			continue
		}
		for _, value := range info.Values {
			if value.StartTok == start && value.EndTok == end {
				return c.typeSpan(info.TypeStart, info.TypeEnd)
			}
		}
	}
	if c.function.BodyStart < start && end <= c.function.BodyEnd {
		body := syntax.ParseFuncBodyStatements(*file, c.function)
		for _, stmt := range body.Stmts {
			if start < stmt.StartTok || end > stmt.EndTok {
				continue
			}
			if stmt.Kind == syntax.StmtReturn {
				values := splitExprList(file, stmt.ExprStart, stmt.ExprEnd)
				results := buildFuncSignature(file, &c.function).Results
				for i, value := range values {
					if value.StartTok == start && value.EndTok == end && i < len(results) {
						return c.typeSpan(results[i].TypeStart, results[i].TypeEnd)
					}
				}
			}
			if stmt.Kind == syntax.StmtAssign {
				op := findTopLevelAssignOp(file, stmt.StartTok, stmt.EndTok)
				if op < 0 || tokenTextIs(file, op, ":=") {
					continue
				}
				values := splitExprList(file, op+1, stmt.EndTok)
				left := splitExprList(file, stmt.StartTok, op)
				for i, value := range values {
					if value.StartTok == start && value.EndTok == end && i < len(left) {
						return c.expression(left[i].StartTok, left[i].EndTok, stmt.StartTok).typ
					}
				}
			}
		}
	}
	return 0
}
