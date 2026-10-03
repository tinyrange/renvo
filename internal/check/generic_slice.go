package check

func (e *genericEnvironment) arrayIndexLimit(id int) uint64 {
	if e.types.get(id).kind == genericParameter {
		constraint, _ := e.parameterConstraint(id)
		limit := ^uint64(0)
		for _, term := range constraint.terms {
			length := e.arrayIndexLimit(term.typ)
			if length < limit {
				limit = length
			}
		}
		return limit
	}
	v := e.types.get(e.types.underlying(id))
	if v.kind == genericPointer {
		v = e.types.get(e.types.underlying(v.elem))
	}
	if v.kind == genericArray {
		return v.length
	}
	return ^uint64(0)
}

func (e *genericEnvironment) sliceType(id int, full bool) (int, bool) {
	if e.types.get(id).kind == genericParameter {
		constraint, ok := e.parameterConstraint(id)
		if !ok || constraint.all || len(constraint.terms) == 0 {
			return 0, false
		}
		core := 0
		stringOrBytes := true
		for _, term := range constraint.terms {
			underlying := e.types.underlying(term.typ)
			v := e.types.get(underlying)
			if v.kind != genericBasic || v.name != "string" {
				stringOrBytes = stringOrBytes && v.kind == genericSlice && v.elem == e.types.basic("byte")
			}
			if _, valid := e.sliceType(term.typ, full); !valid {
				return 0, false
			}
			if core == 0 {
				core = underlying
			} else if core != underlying {
				core = -1
			}
		}
		if core < 0 && !stringOrBytes {
			return 0, false
		}
		if core > 0 {
			v := e.types.get(core)
			if v.kind == genericArray || v.kind == genericPointer {
				return e.sliceType(core, full)
			}
		}
		return id, true
	}
	v := e.types.get(e.types.underlying(id))
	if v.kind == genericPointer {
		v = e.types.get(e.types.underlying(v.elem))
		if v.kind != genericArray {
			return 0, false
		}
	}
	if v.kind == genericArray {
		return e.types.intern(genericType{kind: genericSlice, elem: v.elem}), true
	}
	return id, v.kind == genericSlice || !full && v.kind == genericBasic && v.name == "string"
}

// validateIndex returns the index for a constant, or -1 for a runtime value.
func (c *genericExpressionContext) validateIndex(start int, end int, before int) int64 {
	e := c.specializer.environment
	value := c.expression(start, end, before)
	valid := e.allowsOperation(value.typ, "<<")
	if value.untyped != 0 {
		valid = e.argumentAssignable(value, e.types.basic("int"))
	}
	index := int64(-1)
	if value.constant != nil {
		integer := genericConstantInteger(value.constant)
		var fits bool
		index, fits = wideInt64(integer)
		valid = valid && fits && index >= 0 && e.constantFits(value, e.types.basic("int"))
	}
	if !valid {
		e.fail(c.scope, start, "index must be a nonnegative integer")
	}
	c.lowerExpectedConstant(value, e.types.basic("int"))
	return index
}

// Constant bounds must be ordered even when another bound is dynamic. A
// missing high bound is known only for an array or a constant string.
func (c *genericExpressionContext) sliceExpression(base genericArgument, start int, open int, colon int, end int, before int) genericArgument {
	e := c.specializer.environment
	file := &e.graph.Packages[c.scope.pkg].Files[c.scope.file].File
	second := findTypeTopLevelChar(file, colon+1, end-1, ':')
	if base.untyped == genericUntypedString {
		base.typ = e.types.basic("string")
	}
	result, ok := e.sliceType(base.typ, second >= 0)
	if !ok {
		e.fail(c.scope, start, "slice operation is not valid for the constraint")
	}
	if e.types.get(e.coreType(base.typ)).kind == genericArray && !c.addressable(start, open, before) {
		e.fail(c.scope, start, "sliced array is not addressable")
	}
	limit := e.arrayIndexLimit(base.typ)
	if base.constant != nil && base.constant.text != nil {
		limit = uint64(len(*base.constant.text))
	}
	ends := []int{colon, end - 1}
	if second >= 0 {
		ends = []int{colon, second, end - 1}
	}
	bound, previous := open+1, uint64(0)
	for i, stop := range ends {
		value := ^uint64(0)
		if bound < stop {
			if index := c.validateIndex(bound, stop, before); index >= 0 {
				value = uint64(index)
			}
		} else if i == 0 {
			value = 0
		} else if second >= 0 {
			e.fail(c.scope, stop, "full slice requires high and max bounds")
		} else {
			value = limit
		}
		if value != ^uint64(0) {
			if value < previous || value > limit {
				e.fail(c.scope, bound, "slice bounds are out of order or exceed length")
			}
			previous = value
		}
		bound = stop + 1
	}
	return genericArgument{typ: result}
}
