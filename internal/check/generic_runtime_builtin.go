package check

// These primitives are part of Renvo's existing runtime ABI. Checking every
// body in a generic graph must retain their concrete result types, including
// the descriptor inferred from open, before checking the ordinary Go builtins.
// The caller has already excluded package declarations and local bindings.
func (c *genericExpressionContext) runtimeBuiltin(start int, name string, spans []ExprSpan, before int) (genericArgument, bool) {
	if name != "open" && name != "read" && name != "write" && name != "chmod" {
		return genericArgument{}, false
	}
	e := c.specializer.environment
	integer := e.types.basic("int")
	params := []int{integer, integer}
	if name == "open" {
		params[0] = e.types.basic("string")
	}
	if name == "read" || name == "write" {
		params = []int{integer, e.types.intern(genericType{kind: genericSlice, elem: e.types.basic("byte")}), integer}
	}
	actual := c.expressionValues(spans, before)
	if name == "write" && len(actual) == 3 && e.argumentAssignable(actual[1], e.types.basic("string")) {
		params[1] = e.types.basic("string")
	}
	c.functionArguments(&genericType{kind: genericFunc, params: params}, actual, false, start)
	return genericArgument{typ: integer}, true
}
