package check

func (c *genericExpressionContext) minmax(start int, spans []ExprSpan, before int, minimum bool) genericArgument {
	e := c.specializer.environment
	file := &e.graph.Packages[c.scope.pkg].Files[c.scope.file].File
	actual := c.expressionValues(spans, before)
	result := genericArgument{}
	if len(actual) == 0 {
		e.fail(c.scope, start, "min and max require arguments")
		return result
	}
	for _, value := range actual {
		if value.untyped == 0 {
			result.typ = value.typ
			break
		}
	}
	for i, value := range actual {
		if result.typ != 0 {
			if !e.argumentAssignable(value, result.typ) {
				e.fail(c.scope, start, "incompatible min or max argument")
			}
			c.lowerExpectedConstant(value, result.typ)
			value.constant = e.roundConstant(value.constant, result.typ)
		} else {
			kind, ok := genericMergeUntyped(result.untyped, value.untyped)
			if !ok {
				e.fail(c.scope, start, "incompatible min or max constants")
			}
			result.untyped = kind
			result.shifted = append(result.shifted, value.shifted...)
		}
		if i == 0 {
			result.constant = value.constant
		} else if result.constant != nil && value.constant != nil {
			less := false
			if result.constant.text != nil && value.constant.text != nil {
				less = *value.constant.text < *result.constant.text
			} else {
				difference := genericRationalAdd(value.constant.real, genericRationalNegate(result.constant.real))
				less = difference.numerator.negative
			}
			if less == minimum {
				result.constant = value.constant
			}
		} else {
			result.constant = nil
		}

	}
	if len(result.shifted) != 0 {
		for _, value := range actual {
			if value.constant != nil {
				result.shifted = append(result.shifted, value.constant)
			}
		}
	}
	typ := result.typ
	if typ == 0 {
		typ = e.types.defaultType(result.untyped)
	}
	if !e.allowsOperation(typ, "<") || tokenTextIs(file, spans[len(spans)-1].EndTok-1, "...") {
		e.fail(c.scope, start, "min and max require ordered arguments without expansion")
	}
	return result
}
