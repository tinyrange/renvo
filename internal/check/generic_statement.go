package check

// Sends and increments are statements rather than value expressions.
func (c *genericExpressionContext) validateSimpleExpression(start int, end int) {
	e := c.specializer.environment
	file := &e.graph.Packages[c.scope.pkg].Files[c.scope.file].File
	if end <= start {
		return
	}
	if findTopLevelAssignOp(file, start, end) >= 0 {
		c.validateAssignment(start, end)
		return
	}
	if tokenTextIs(file, end-1, "++") || tokenTextIs(file, end-1, "--") {
		value := c.expression(start, end-1, start)
		if !e.allowsOperation(value.typ, "unary+") || !c.assignableLocation(start, end-1, start) {
			e.fail(c.scope, start, "increment requires a numeric operand")
		}
		return
	}
	depth := 0
	for i := start; i < end; i++ {
		if tokCharIs(file, i, '(') || tokCharIs(file, i, '[') || tokCharIs(file, i, '{') {
			depth++
		} else if tokCharIs(file, i, ')') || tokCharIs(file, i, ']') || tokCharIs(file, i, '}') {
			depth--
		} else if depth == 0 && i > start && tokenTextIs(file, i, "<-") {
			channel := c.expression(start, i, start)
			value := c.expression(i+1, end, start)
			v := e.types.get(e.coreType(channel.typ))
			if v.kind != genericChan || v.direction == ChanReceiveOnly || !e.argumentAssignable(value, v.elem) {
				e.fail(c.scope, i, "send does not match channel constraint")
			}
			c.lowerExpectedConstant(value, v.elem)
			return
		}
	}
	c.expression(start, end, start)
}

func (c *genericExpressionContext) assignableLocation(start int, end int, before int) bool {
	if c.addressable(start, end, before) {
		return true
	}
	e := c.specializer.environment
	file := &e.graph.Packages[c.scope.pkg].Files[c.scope.file].File
	start, end = stripOuterParens(file, start, end)
	if tokCharIs(file, end-1, ']') {
		open := genericMatchingOpen(file, start, end-1, '[', ']')
		if open > start {
			return e.types.get(e.coreType(c.expression(start, open, before).typ)).kind == genericMap
		}
	}
	return false
}
