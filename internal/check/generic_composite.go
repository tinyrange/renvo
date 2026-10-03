package check

func (c *genericExpressionContext) composite(typ int, open int, end int, before int) genericArgument {
	e := c.specializer.environment
	file := &e.graph.Packages[c.scope.pkg].Files[c.scope.file].File
	v := e.types.get(e.coreType(typ))
	if v.kind == genericPointer {
		v = e.types.get(e.coreType(v.elem))
	}
	if v.kind != genericStruct && v.kind != genericArray && v.kind != genericSlice && v.kind != genericMap {
		e.fail(c.scope, open, "composite literal requires a common composite type")
		return genericArgument{}
	}
	parts := splitExprList(file, open+1, end-1)
	keyed, unkeyed := false, false
	var names []string
	var indices []uint64
	index, extent := uint64(0), uint64(0)
	for i, part := range parts {
		colon := findTypeTopLevelChar(file, part.StartTok, part.EndTok, ':')
		target := v.elem
		start := part.StartTok
		if colon >= 0 {
			keyed = true
			start = colon + 1
		} else {
			unkeyed = true
		}
		if v.kind == genericStruct {
			target = 0
			if colon < 0 {
				if i < len(v.fields) {
					target = v.fields[i].typ
				}
			} else {
				name := tokenString(file, part.StartTok)
				for _, old := range names {
					if old == name {
						e.fail(c.scope, part.StartTok, "duplicate composite field")
					}
				}
				names = append(names, name)
				for _, field := range v.fields {
					if field.name == name && colon == part.StartTok+1 {
						target = field.typ
					}
				}
			}
		} else if v.kind == genericMap {
			key := c.expression(part.StartTok, colon, before)
			if colon < 0 || !e.argumentAssignable(key, v.key) {
				e.fail(c.scope, part.StartTok, "invalid map literal key")
			}
			c.lowerExpectedConstant(key, v.key)
		} else {
			if colon >= 0 {
				value := c.validateIndex(part.StartTok, colon, before)
				if value < 0 {
					e.fail(c.scope, part.StartTok, "literal index must be a constant within bounds")
					return genericArgument{}
				}
				index = uint64(value)
			}
			if v.kind == genericArray && v.length != ^uint64(0) && index >= v.length {
				e.fail(c.scope, part.StartTok, "literal index must be a constant within bounds")
			}
			for _, old := range indices {
				if old == index {
					e.fail(c.scope, part.StartTok, "duplicate literal index")
				}
			}
			indices = append(indices, index)
			// Explicit keys must fit int; inferred positions and lengths can
			// exceed that bound as following unkeyed elements are appended.
			index++
			if index > extent {
				extent = index
			}
		}
		if target == 0 {
			e.fail(c.scope, part.StartTok, "invalid composite field")
			continue
		}
		var value genericArgument
		if tokCharIs(file, start, '{') {
			value = c.composite(target, start, part.EndTok, before)
		} else {
			value = c.expression(start, part.EndTok, before)
		}
		if value.function != 0 {
			value = c.functionValue(value.function-1, value.start, value.end, target)
		}
		if !e.argumentAssignable(value, target) {
			e.fail(c.scope, start, "composite value is not valid for every type in the constraint")
		}
		c.lowerExpectedConstant(value, target)
	}
	if v.kind == genericStruct && (keyed && unkeyed || unkeyed && len(parts) != len(v.fields)) {
		e.fail(c.scope, open, "invalid number or mixture of struct literal fields")
	}
	if v.kind == genericArray && v.length == ^uint64(0) {
		typ = e.types.intern(genericType{kind: genericArray, elem: v.elem, length: extent})
	}
	return genericArgument{typ: typ}
}
