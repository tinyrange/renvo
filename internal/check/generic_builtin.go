package check

func (c *genericExpressionContext) builtinIfVisible(start int, name string, spans []ExprSpan, before int) (genericArgument, bool) {
	if c.specializer.environment.packageValueName(c.scope.pkg, name) {
		return genericArgument{}, false
	}
	if value, ok := c.runtimeBuiltin(start, name, spans, before); ok {
		return value, true
	}
	return c.builtin(start, name, spans, before)
}

func (c *genericExpressionContext) builtin(start int, name string, spans []ExprSpan, before int) (genericArgument, bool) {
	e := c.specializer.environment
	file := &e.graph.Packages[c.scope.pkg].Files[c.scope.file].File
	if name == "min" || name == "max" {
		e.requireVersion(c.scope, start, "1.21", "predeclared "+name)
		return c.minmax(start, spans, before, name == "min"), true
	}
	if name == "print" || name == "println" || name == "panic" || name == "recover" {
		if name == "panic" && len(spans) != 1 || name == "recover" && len(spans) != 0 {
			e.fail(c.scope, start, "wrong number of builtin arguments")
		}
		for _, span := range spans {
			value := c.expression(span.StartTok, span.EndTok, before)
			if value.untyped != 0 && value.untyped != genericUntypedNil {
				c.lowerExpectedConstant(value, e.types.defaultType(value.untyped))
			}
		}
		if name == "recover" {
			return genericArgument{typ: e.types.intern(genericType{kind: genericInterface})}, true
		}
		return genericArgument{}, true
	}
	if name == "new" || name == "make" {
		if len(spans) == 0 {
			e.fail(c.scope, start, "missing builtin type argument")
			return genericArgument{}, true
		}
		id := c.typeSpan(spans[0].StartTok, spans[0].EndTok)
		if name == "new" {
			if len(spans) != 1 {
				e.fail(c.scope, start, "new requires one type argument")
			}
			return genericArgument{typ: e.types.intern(genericType{kind: genericPointer, elem: id})}, true
		}
		v := e.types.get(e.coreType(id))
		if v.kind != genericSlice && v.kind != genericMap && v.kind != genericChan || v.kind == genericSlice && (len(spans) < 2 || len(spans) > 3) || v.kind != genericSlice && len(spans) > 2 {
			e.fail(c.scope, start, "invalid make arguments for constraint")
		}
		length := int64(-1)
		for i, span := range spans[1:] {
			bound := c.validateIndex(span.StartTok, span.EndTok, before)
			if i == 0 {
				length = bound
			} else if bound >= 0 && length > bound {
				e.fail(c.scope, span.StartTok, "make length exceeds capacity")
			}
		}
		return genericArgument{typ: id}, true
	}
	if name != "append" && name != "copy" && name != "len" && name != "cap" && name != "clear" && name != "delete" && name != "close" && name != "real" && name != "imag" && name != "complex" {
		return genericArgument{}, false
	}
	if name == "clear" {
		e.requireVersion(c.scope, start, "1.21", "predeclared clear")
	}
	callsBefore := c.nonconstantCalls
	var values []genericArgument
	spread := false
	for i, span := range spans {
		if i == len(spans)-1 && tokenTextIs(file, span.EndTok-1, "...") {
			spread = true
			span.EndTok--
		}
		values = append(values, c.expression(span.StartTok, span.EndTok, before))
	}
	need := 1
	if name == "copy" || name == "delete" || name == "complex" {
		need = 2
	}
	if len(values) < need || name != "append" && (len(values) != need || spread) {
		e.fail(c.scope, start, "wrong number of builtin arguments")
		return genericArgument{}, true
	}
	first := values[0]
	v := e.types.get(e.coreType(first.typ))
	// The legacy runtime ABI also has close(int) -> int. A type parameter
	// constrained by ~int remains a Go close operand, so its entire type set
	// must consist of send-capable channels below.
	if name == "close" && first.typ == e.types.basic("int") {
		return genericArgument{typ: first.typ}, true
	}
	switch name {
	case "len", "cap":
		if e.types.get(first.typ).kind == genericParameter && !e.allowsBuiltin(first.typ, name) {
			e.fail(c.scope, start, "builtin is not valid for every type in the constraint")
		}
		length := ^uint64(0)
		if name == "len" && first.constant != nil && first.constant.text != nil {
			length = uint64(len(*first.constant.text))
		}
		array := v
		if array.kind == genericPointer {
			array = e.types.get(e.types.underlying(array.elem))
		}
		if array.kind == genericArray && e.types.get(first.typ).kind != genericParameter && c.nonconstantCalls == callsBefore {
			length = array.length
		}
		if length != ^uint64(0) {
			return genericArgument{untyped: genericUntypedInt, constant: genericConstantLiteral(genericUnsignedDecimal(length))}, true
		}
		return genericArgument{typ: e.types.basic("int")}, true
	case "append":
		valid := v.kind == genericSlice
		if spread {
			valid = valid && len(values) == 2
			if len(values) == 2 {
				other := e.types.get(e.coreType(values[1].typ))
				valid = valid && (other.kind == genericSlice && other.elem == v.elem || v.elem == e.types.basic("byte") && (values[1].untyped == genericUntypedString || other.kind == genericBasic && other.name == "string"))
			}
		} else {
			for _, value := range values[1:] {
				if !e.argumentAssignable(value, v.elem) {
					valid = false
				}
				c.lowerExpectedConstant(value, v.elem)
			}
		}
		if !valid {
			e.fail(c.scope, start, "append arguments do not match slice constraint")
		}
		return genericArgument{typ: first.typ}, true
	case "copy":
		other := e.types.get(e.coreType(values[1].typ))
		if v.kind != genericSlice || !(other.kind == genericSlice && v.elem == other.elem || v.elem == e.types.basic("byte") && (values[1].untyped == genericUntypedString || other.kind == genericBasic && other.name == "string")) {
			e.fail(c.scope, start, "copy requires compatible slice element types")
		}
		return genericArgument{typ: e.types.basic("int")}, true
	case "clear", "delete", "close":
		terms := []genericTerm{{typ: first.typ}}
		if e.types.get(first.typ).kind == genericParameter {
			constraint, ok := e.parameterConstraint(first.typ)
			terms = constraint.terms
			if !ok || constraint.all || len(terms) == 0 {
				e.fail(c.scope, start, "builtin requires a restricted constraint")
			}
		}
		key := 0
		for _, term := range terms {
			kind := e.types.get(e.types.underlying(term.typ))
			valid := false
			if name == "clear" {
				valid = kind.kind == genericSlice || kind.kind == genericMap
			}
			if name == "close" {
				valid = kind.kind == genericChan && kind.direction != ChanReceiveOnly
			}
			if name == "delete" {
				valid = kind.kind == genericMap && (key == 0 || key == kind.key) && e.argumentAssignable(values[1], kind.key)
				key = kind.key
			}
			if !valid {
				e.fail(c.scope, start, "builtin is not valid for every type in the constraint")
			}
		}
		if name == "delete" {
			c.lowerExpectedConstant(values[1], key)
		}
	case "real", "imag", "complex":
		result := genericArgument{}
		typed := 0
		for _, value := range values {
			if value.untyped == 0 {
				if typed != 0 && typed != value.typ {
					e.fail(c.scope, start, "complex operands must have matching types")
				}
				typed = value.typ
			}
			kind := e.types.get(e.types.underlying(value.typ))
			valid := value.untyped >= genericUntypedInt && value.untyped <= genericUntypedComplex
			if name == "complex" {
				valid = valid && value.constant != nil && len(value.constant.imaginary.numerator.words) == 0 || kind.kind == genericBasic && (kind.name == "float32" || kind.name == "float64")
			} else {
				valid = valid || kind.kind == genericBasic && (kind.name == "complex64" || kind.name == "complex128")
			}
			if !valid {
				e.fail(c.scope, start, "invalid complex builtin operand")
			}
		}
		kind := e.types.get(e.types.underlying(typed))
		if name == "complex" {
			if typed != 0 {
				for i, value := range values {
					if !e.argumentAssignable(value, typed) {
						e.fail(c.scope, start, "complex operand is not representable in its type")
					}
					c.lowerExpectedConstant(value, typed)
					values[i].constant = e.roundConstant(value.constant, typed)
				}
				first = values[0]
			}
			if typed == 0 {
				result.untyped = genericUntypedComplex
			} else if kind.name == "float32" {
				result.typ = e.types.basic("complex64")
			} else {
				result.typ = e.types.basic("complex128")
			}
			if first.constant != nil && values[1].constant != nil {
				result.constant = &genericConstant{real: first.constant.real, imaginary: values[1].constant.real}
			}
		} else {
			if typed == 0 {
				result.untyped = genericUntypedFloat
			} else if kind.name == "complex64" {
				result.typ = e.types.basic("float32")
			} else {
				result.typ = e.types.basic("float64")
			}
			if first.constant != nil {
				part := first.constant.real
				if name == "imag" {
					part = first.constant.imaginary
				}
				result.constant = &genericConstant{real: part, imaginary: genericRationalInteger(wideSmall(0))}
			}
		}
		return result, true

	}
	return genericArgument{}, true
}
