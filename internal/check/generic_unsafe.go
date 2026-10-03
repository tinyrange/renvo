package check

// Layout intrinsics do not evaluate their operands. Checking still defaults an
// untyped operand, checks its representability, and requires a single value.
func (c *genericExpressionContext) unsafeLayout(start int, name string, spans []ExprSpan, before int) genericArgument {
	e := c.specializer.environment
	file := &e.graph.Packages[c.scope.pkg].Files[c.scope.file].File
	result := genericArgument{typ: e.types.basic("uintptr")}
	if len(spans) != 1 || tokenTextIs(file, spans[0].EndTok-1, "...") {
		e.fail(c.scope, start, "unsafe layout operation requires one value argument")
		return result
	}
	operandStart, operandEnd := stripOuterParens(file, spans[0].StartTok, spans[0].EndTok)
	callsBefore := c.nonconstantCalls
	if name == "Offsetof" {
		if operandEnd < operandStart+3 || !tokCharIs(file, operandEnd-2, '.') {
			e.fail(c.scope, operandStart, "unsafe.Offsetof requires a field selector")
			return result
		}
		base := c.expression(operandStart, operandEnd-2, before)
		c.nonconstantCalls = callsBefore
		if !c.unsafeOperand(base, operandStart) {
			return result
		}
		path, indirect, method := e.unsafeFieldPath(base.typ, tokenString(file, operandEnd-1), c.scope.pkg)
		if method {
			e.fail(c.scope, operandEnd-1, "unsafe.Offsetof cannot use a method value")
			return result
		}
		if len(path) == 0 {
			e.fail(c.scope, operandEnd-1, "unsafe.Offsetof requires a single accessible field")
			return result
		}
		if indirect {
			e.fail(c.scope, operandEnd-1, "unsafe.Offsetof field is embedded through a pointer")
			return result
		}
		typ := base.typ
		if pointer := e.types.get(e.types.underlying(typ)); pointer.kind == genericPointer {
			typ = pointer.elem
		}
		if e.types.variableSize(typ, 0) {
			c.nonconstantCalls++
		} else if offset, ok := e.layoutOffset(typ, path); ok {
			result.constant = genericConstantLiteral(genericUnsignedDecimal(offset))
		}
		return result
	}
	value := c.expression(operandStart, operandEnd, before)
	c.nonconstantCalls = callsBefore
	if !c.unsafeOperand(value, operandStart) {
		return result
	}
	if value.untyped != 0 {
		value.typ = e.types.defaultType(value.untyped)
		c.lowerExpectedConstant(value, value.typ)
	}
	if !e.graph.Layout.Object && e.types.get(e.types.underlying(value.typ)).kind == genericFunc {
		// Give layout queries an explicit callable type before backend constant
		// collection. Direct function names otherwise still look like native
		// code pointers, and local callback names may not be collected yet. A
		// descriptor aggregate has the callable's layout and keeps the original
		// operand's syntactic variable uses without evaluating it. Store it in
		// the interface field so array-checking contexts need no instantiated
		// signature to spell this fixed layout.
		byteStart := int(file.Tokens[operandStart].Start)
		byteEnd := int(file.Tokens[operandEnd-1].End)
		var changes []genericReplacement
		for _, change := range c.changes {
			if change.start >= byteStart && change.end <= byteEnd {
				change.start -= byteStart
				change.end -= byteStart
				changes = append(changes, change)
			}
		}
		operand := applyGenericReplacements(file.Src[byteStart:byteEnd], changes)
		kept := c.changes[:0]
		for _, change := range c.changes {
			if change.start < byteStart || change.end > byteEnd {
				kept = append(kept, change)
			}
		}
		c.changes = kept
		c.replace(operandStart, operandEnd, "struct { kind int; data interface{}; _ [0][]struct{} }{data: "+operand+"}")
	}
	if e.types.variableSize(value.typ, 0) {
		c.nonconstantCalls++
	} else {
		layout := e.valueLayout(value.typ, e.graph.Layout.Object, 0)
		if name == "Sizeof" && layout.known {
			result.constant = genericConstantLiteral(genericUnsignedDecimal(genericLayoutQuerySize(layout)))
		} else if name == "Alignof" && layout.align > 0 {
			result.constant = genericConstantLiteral(genericDecimal(layout.align))
		}
	}
	return result
}

func (e *genericEnvironment) unsafeScalarSize(typ int) int {
	v := e.types.get(e.types.underlying(typ))
	if v.kind != genericBasic {
		return 0
	}
	switch v.name {
	case "bool", "int8", "uint8":
		return 1
	case "int16", "uint16":
		return 2
	case "int32", "uint32", "float32":
		return 4
	case "int64", "uint64", "float64", "complex64":
		return 8
	case "complex128", "string":
		return 16
	case "int", "uint", "uintptr":
		return e.wordBits / 8
	}
	return 0
}

func (c *genericExpressionContext) unsafeOperand(value genericArgument, token int) bool {
	e := c.specializer.environment
	if value.function != 0 || len(value.results) != 0 || value.untyped == genericUntypedNil || value.typ == 0 && value.untyped == 0 {
		e.fail(c.scope, token, "unsafe layout operand must be a single value with a type")
		return false
	}
	if value.untyped != 0 && !e.constantFits(value, e.types.defaultType(value.untyped)) {
		e.fail(c.scope, token, "unsafe layout operand overflows its default type")
		return false
	}
	return true
}

// A parameter is variable-sized even when all of its constraint terms have
// the same layout. Only arrays and structs propagate that property; a slice,
// pointer, map, channel, interface or function has a fixed representation.
func (t *genericTypes) variableSize(id int, depth int) bool {
	if depth > len(t.items) {
		return false
	}
	v := t.get(t.underlying(id))
	if v.kind == genericParameter {
		return true
	}
	if v.kind == genericArray {
		return t.variableSize(v.elem, depth+1)
	}
	if v.kind == genericStruct {
		for _, field := range v.fields {
			if t.variableSize(field.typ, depth+1) {
				return true
			}
		}
	}
	return false
}

type genericUnsafeFieldNode struct {
	typ       int
	path      []int
	ancestors []int
	indirect  bool
}

// Offsetof includes value embedding offsets, but must reject promotion through
// an embedded pointer. Its explicit base may itself be a pointer to a struct.
func (e *genericEnvironment) unsafeFieldPath(typ int, name string, pkg int) ([]int, bool, bool) {
	if e.types.get(typ).kind == genericParameter {
		return nil, false, false
	}
	if v := e.types.get(e.types.underlying(typ)); v.kind == genericPointer {
		typ = v.elem
	}
	e.attachMethods(typ)
	owner := e.graph.Packages[pkg].Ref.ImportPath
	pending := []genericUnsafeFieldNode{{typ: typ}}
	for len(pending) > 0 {
		var next []genericUnsafeFieldNode
		matches := 0
		var selected genericUnsafeFieldNode
		method := false
		for _, node := range pending {
			v := e.types.get(e.types.underlying(node.typ))
			for _, m := range e.types.directMethods(node.typ, true) {
				if m.name == name && (m.pkg == "" || m.pkg == owner) {
					matches++
					method = true
				}
			}
			if v.kind != genericStruct {
				continue
			}
			ancestors := append(append([]int(nil), node.ancestors...), node.typ)
			for i, field := range v.fields {
				path := append(append([]int(nil), node.path...), i)
				if field.name == name && (field.pkg == "" || field.pkg == owner) {
					matches++
					selected = node
					selected.path = path
				}
				if !field.embedded {
					continue
				}
				child, indirect := field.typ, node.indirect
				if pointer := e.types.get(e.types.underlying(child)); pointer.kind == genericPointer {
					child, indirect = pointer.elem, true
				}
				cycle := false
				for _, ancestor := range ancestors {
					if ancestor == child {
						cycle = true
					}
				}
				if !cycle {
					next = append(next, genericUnsafeFieldNode{typ: child, path: path, ancestors: ancestors, indirect: indirect})
				}
			}
		}
		if matches > 0 {
			if matches == 1 && !method {
				return selected.path, selected.indirect, false
			}
			return nil, false, matches == 1 && method
		}
		pending = next
	}
	return nil, false, false
}
