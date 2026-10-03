package check

// Infer the callee and its generic function arguments together. Each argument
// gets fresh parameters: two uses of the same function may infer different
// instantiations in a single call.
func (c *genericExpressionContext) inferCall(d *genericDeclaration, explicit []int, actual []genericArgument, spread bool) ([]int, bool) {
	e := c.specializer.environment
	parameters, constraints, signature := c.inferenceDeclaration(d)
	parameters = append([]int(nil), parameters...)
	constraints = append([]genericConstraint(nil), constraints...)
	values := append([]genericArgument(nil), actual...)
	file := &e.graph.Packages[c.scope.pkg].Files[c.scope.file].File
	for i, value := range values {
		if value.function == 0 {
			continue
		}
		e.resolveDeclaration(value.function - 1)
		argument := &e.decls[value.function-1]
		_, next := c.declaration(value.start, value.end)
		var given []int
		if tokCharIs(file, next, '[') {
			for token := next + 1; token < value.end-1; {
				last := nextTopLevelComma(file, token, value.end-1)
				given = append(given, c.typeSpan(token, last))
				token = last + 1
			}
		}
		if len(given) > len(argument.parameters) {
			return nil, false
		}
		if len(given) < len(argument.parameters) && !e.requireVersion(c.scope, value.start, "1.21", "generic function argument inference") {
			return nil, false
		}
		replacements := append([]int(nil), given...)
		for j := len(given); j < len(argument.parameters); j++ {
			p := *e.types.get(argument.parameters[j])
			p.origin += "/inference/" + genericDecimal(c.scope.pkg) + "/" + genericDecimal(c.scope.file) + "/" + genericDecimal(value.start)
			replacements = append(replacements, e.types.intern(p))
		}
		for j := len(given); j < len(argument.parameters); j++ {
			parameters = append(parameters, replacements[j])
			constraints = append(constraints, e.types.substituteConstraint(argument.constraints[j], argument.parameters, replacements))
		}
		values[i].typ = e.types.substitute(argument.typ, argument.parameters, replacements)
		values[i].function = 0
	}
	arguments, ok := e.types.inferArgumentsVersion(parameters, constraints, signature, explicit, values, spread, 0, e.versionBefore(c.scope, "1.21"))
	if !ok {
		return nil, false
	}
	return arguments[:len(d.parameters)], true
}

// Recursive calls infer fresh callee parameters against the enclosing function's
// parameters. Sharing their identities would turn an equation such as E'=E into
// an unresolved self-reference, even though E is a known type in the caller.
func (c *genericExpressionContext) inferenceDeclaration(d *genericDeclaration) ([]int, []genericConstraint, int) {
	e := c.specializer.environment
	overlap := false
	for _, parameter := range d.parameters {
		for _, outer := range c.scope.parameters {
			overlap = overlap || parameter == outer
		}
	}
	if !overlap {
		return d.parameters, d.constraints, d.typ
	}
	parameters := make([]int, len(d.parameters))
	constraints := make([]genericConstraint, len(d.constraints))
	for i, id := range d.parameters {
		parameter := *e.types.get(id)
		parameter.origin += "/callee-inference"
		parameter.underlying = 0
		parameters[i] = e.types.intern(parameter)
	}
	for i, constraint := range d.constraints {
		constraints[i] = e.types.substituteConstraint(constraint, d.parameters, parameters)
		e.types.items[parameters[i]-1].underlying = e.types.coreType(constraints[i])
	}
	return parameters, constraints, e.types.substitute(d.typ, d.parameters, parameters)
}
