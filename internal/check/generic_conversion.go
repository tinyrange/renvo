package check

func (e *genericEnvironment) convertible(value genericArgument, target int) bool {
	if value.typ == target && target != 0 {
		return true
	}
	// Pointer/uintptr conversions involving parameters require one underlying
	// type in each type set. Checking each union term independently would accept
	// a mixed unsafe.Pointer | uintptr set by combining two conversion rules.
	if e.types.get(value.typ).kind == genericParameter || e.types.get(target).kind == genericParameter {
		crosses := e.conversionHasUnderlying(value.typ, "unsafe.Pointer") && e.conversionHasUnderlying(target, "uintptr") ||
			e.conversionHasUnderlying(value.typ, "uintptr") && e.conversionHasUnderlying(target, "unsafe.Pointer")
		if crosses && (e.coreType(value.typ) == 0 || e.coreType(target) == 0) {
			return false
		}
	}
	if e.types.get(target).kind == genericParameter {
		constraint, ok := e.parameterConstraint(target)
		if !ok || constraint.all || len(constraint.terms) == 0 {
			return false
		}
		for _, term := range constraint.terms {
			if !e.convertible(value, term.typ) {
				return false
			}
		}
		return true
	}
	if e.types.get(value.typ).kind == genericParameter {
		if e.types.get(e.types.underlying(target)).kind == genericInterface {
			return e.satisfies(value.typ, e.types.interfaceConstraint(target))
		}
		constraint, ok := e.parameterConstraint(value.typ)
		if !ok || constraint.all || len(constraint.terms) == 0 {
			return false
		}
		for _, term := range constraint.terms {
			if !e.convertible(genericArgument{typ: term.typ}, target) {
				return false
			}
		}
		return true
	}
	to := e.types.get(e.types.underlying(target))
	if to.kind == genericInterface {
		return e.argumentAssignable(value, target)
	}
	if value.constant != nil && to.kind == genericBasic && genericNumeric(to.name) && !e.constantFits(value, target) {
		return false
	}
	if value.untyped != 0 {
		if value.untyped == genericUntypedString && to.kind == genericSlice {
			element := e.types.get(to.elem)
			return element.kind == genericBasic && (element.name == "uint8" || element.name == "int32")
		}
		if to.kind == genericBasic && to.name == "string" && (value.untyped == genericUntypedInt || value.untyped == genericUntypedRune) {
			return len(value.shifted) == 0
		}
		return e.argumentAssignable(value, target)
	}
	if e.types.assignable(value.typ, target) {
		return true
	}
	from := e.types.get(e.types.underlying(value.typ))
	if from.kind == genericBasic && from.name == "unsafe.Pointer" {
		return to.kind == genericPointer || to.kind == genericBasic && (to.name == "uintptr" || to.name == "unsafe.Pointer")
	}
	if to.kind == genericBasic && to.name == "unsafe.Pointer" {
		return from.kind == genericPointer || from.kind == genericBasic && from.name == "uintptr"
	}
	if from.kind == genericInvalid || to.kind == genericInvalid {
		return false
	}
	if genericConversionTypeEqual(&e.types, e.types.underlying(value.typ), e.types.underlying(target)) {
		return true
	}
	if from.kind == genericBasic && to.kind == genericBasic {
		if genericReal(from.name) && genericReal(to.name) {
			return true
		}
		if (from.name == "complex64" || from.name == "complex128") && (to.name == "complex64" || to.name == "complex128") {
			return true
		}
		return genericInteger(from.name) && to.name == "string"
	}
	if from.kind == genericSlice && to.kind == genericBasic && to.name == "string" {
		element := e.types.get(from.elem)
		return element.kind == genericBasic && (element.name == "uint8" || element.name == "int32")
	}
	if to.kind == genericSlice && from.kind == genericBasic && from.name == "string" {
		element := e.types.get(to.elem)
		return element.kind == genericBasic && (element.name == "uint8" || element.name == "int32")
	}
	if from.kind == genericSlice {
		array := to
		if array.kind == genericPointer {
			array = e.types.get(e.types.underlying(array.elem))
		}
		if array.kind == genericArray && array.elem == from.elem {
			return true
		}
	}
	return e.types.get(value.typ).kind != genericNamed && e.types.get(target).kind != genericNamed && from.kind == genericPointer && to.kind == genericPointer && genericConversionTypeEqual(&e.types, e.types.underlying(from.elem), e.types.underlying(to.elem))
}

func (e *genericEnvironment) conversionHasUnderlying(id int, name string) bool {
	if e.types.get(id).kind == genericParameter {
		constraint, ok := e.parameterConstraint(id)
		if !ok {
			return false
		}
		for _, term := range constraint.terms {
			if e.types.get(e.types.underlying(term.typ)).name == name {
				return true
			}
		}
		return false
	}
	return e.types.get(e.types.underlying(id)).name == name
}

func (c *genericExpressionContext) conversion(start int, spans []ExprSpan, target int, before int) genericArgument {
	e := c.specializer.environment
	if !e.valueType(target) {
		e.fail(c.scope, start, "constraint interface cannot be used as a value type")
		return genericArgument{typ: target}
	}
	if len(spans) != 1 {
		e.fail(c.scope, start, "conversion requires one argument")
		return genericArgument{typ: target}
	}
	value := c.expression(spans[0].StartTok, spans[0].EndTok, before)
	if !e.convertible(value, target) {
		e.fail(c.scope, start, "conversion is not valid for every type in the constraint")
	}
	c.lowerExpectedConstant(value, target)
	constant := value.constant
	v := e.types.get(e.types.underlying(target))
	if v.kind == genericBasic && v.name == "string" && constant != nil && constant.text == nil {
		integer, _ := wideDivide(constant.real.numerator, constant.real.denominator)
		r, ok := wideInt(integer)
		if !ok || r < 0 || r > 0x10ffff || r >= 0xd800 && r <= 0xdfff {
			r = 0xfffd
		}
		text := string(rune(r))
		constant = &genericConstant{text: &text}
	}
	if e.types.get(target).kind == genericParameter || v.kind != genericBasic {
		constant = nil
	}
	return genericArgument{typ: target, constant: constant}
}
