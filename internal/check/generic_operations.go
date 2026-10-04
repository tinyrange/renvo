package check

func (e *genericEnvironment) parameterConstraint(id int) (genericConstraint, bool) {
	for declIndex := range e.decls {
		d := &e.decls[declIndex]
		for i, p := range d.parameters {
			if p == id && i < len(d.constraints) {
				return d.constraints[i], true
			}
		}
	}
	return genericConstraint{}, false
}

func genericInteger(name string) bool {
	return name == "int" || name == "uint" || name == "uintptr" || name == "int8" || name == "int16" || name == "int32" || name == "int64" || name == "uint8" || name == "uint16" || name == "uint32" || name == "uint64"
}

func genericReal(name string) bool {
	return genericInteger(name) || name == "float32" || name == "float64"
}

func genericNumeric(name string) bool {
	return genericReal(name) || name == "complex64" || name == "complex128"
}

func (t *genericTypes) basicOperation(id int, op string) bool {
	v := t.get(t.underlying(id))
	if op == "nil" {
		return v.kind == genericPointer || v.kind == genericSlice || v.kind == genericMap || v.kind == genericChan || v.kind == genericFunc || v.kind == genericInterface || v.kind == genericBasic && v.name == "unsafe.Pointer"
	}
	if op == "==" || op == "!=" {
		return t.comparable(id, false)
	}
	if v.kind != genericBasic {
		return false
	}
	switch op {
	case "+":
		return genericNumeric(v.name) || v.name == "string"
	case "-", "*", "/", "unary+", "unary-":
		return genericNumeric(v.name)
	case "%", "&", "|", "^", "&^", "<<", ">>":
		return genericInteger(v.name)
	case "<", "<=", ">", ">=":
		return genericReal(v.name) || v.name == "string"
	case "&&", "||", "!":
		return v.name == "bool"
	}
	return false
}

func (e *genericEnvironment) allowsOperation(id int, op string) bool {
	if e.types.get(id).kind != genericParameter {
		return e.types.basicOperation(id, op)
	}
	c, ok := e.parameterConstraint(id)
	if !ok {
		return false
	}
	if op == "==" || op == "!=" {
		return e.types.constraintStrictlyComparable(c)
	}
	if c.all || len(c.terms) == 0 {
		return false
	}
	for _, term := range c.terms {
		if !e.types.basicOperation(term.typ, op) {
			return false
		}
	}
	return true
}

// coreType is shared by constraint inference and operations which require a
// common representation. Channel unions permit a common element type and a
// compatible, most restrictive direction.
func (t *genericTypes) coreType(c genericConstraint) int {
	if c.all || len(c.terms) == 0 {
		return 0
	}
	id := t.underlying(c.terms[0].typ)
	for _, term := range c.terms[1:] {
		next := t.underlying(term.typ)
		if next == id {
			continue
		}
		a, b := t.get(id), t.get(next)
		if a.kind != genericChan || b.kind != genericChan || a.elem != b.elem || a.direction != b.direction && a.direction != ChanBoth && b.direction != ChanBoth {
			return 0
		}
		if a.direction == ChanBoth {
			id = next
		}
	}
	return id
}

func (e *genericEnvironment) coreType(id int) int {
	if e.types.get(id).kind != genericParameter {
		return e.types.underlying(id)
	}
	c, ok := e.parameterConstraint(id)
	if !ok {
		return 0
	}
	return e.types.coreType(c)
}

func (e *genericEnvironment) indexType(id int) (int, int, bool) {
	v := e.types.get(id)
	if v.kind != genericParameter {
		return e.concreteIndexType(id)
	}
	c, ok := e.parameterConstraint(id)
	if !ok || c.all || len(c.terms) == 0 {
		return 0, 0, false
	}
	key, elem, ok := e.concreteIndexType(c.terms[0].typ)
	if !ok {
		return 0, 0, false
	}
	mapIndex := e.types.get(e.types.underlying(c.terms[0].typ)).kind == genericMap
	for _, term := range c.terms[1:] {
		k, v, valid := e.concreteIndexType(term.typ)
		if !valid || k != key || v != elem || mapIndex != (e.types.get(e.types.underlying(term.typ)).kind == genericMap) {
			return 0, 0, false
		}
	}
	return key, elem, true
}

func (e *genericEnvironment) concreteIndexType(id int) (int, int, bool) {
	v := e.types.get(e.types.underlying(id))
	if v.kind == genericMap {
		return v.key, v.elem, true
	}
	if v.kind == genericPointer {
		v = e.types.get(e.types.underlying(v.elem))
		if v.kind != genericArray {
			return 0, 0, false
		}
	}
	if v.kind == genericArray || v.kind == genericSlice {
		return e.types.basic("int"), v.elem, true
	}
	if v.kind == genericBasic && v.name == "string" {
		return e.types.basic("int"), e.types.basic("byte"), true
	}
	return 0, 0, false
}

func (e *genericEnvironment) satisfies(id int, constraint genericConstraint) bool {
	if e.types.get(id).kind == genericParameter {
		c, ok := e.parameterConstraint(id)
		return ok && e.types.constraintSubset(c, constraint)
	}
	v := e.types.get(e.types.underlying(id))
	if v.kind == genericInterface && (v.restricted || v.comparable) {
		return false
	}
	if len(constraint.methods) != 0 {
		e.attachMethods(id)
	}
	return e.types.satisfies(id, constraint)
}

func (e *genericEnvironment) allowsBuiltin(id int, name string) bool {
	c, ok := e.parameterConstraint(id)
	if !ok || c.all || len(c.terms) == 0 {
		return false
	}
	for _, term := range c.terms {
		v := e.types.get(e.types.underlying(term.typ))
		if v.kind == genericPointer {
			v = e.types.get(e.types.underlying(v.elem))
			if v.kind != genericArray {
				return false
			}
		}
		if name == "copy" {
			if v.kind != genericSlice {
				return false
			}
		} else if v.kind != genericArray && v.kind != genericSlice && v.kind != genericChan && (name != "len" || v.kind != genericMap && (v.kind != genericBasic || v.name != "string")) {
			return false
		}
	}
	return true
}

func (e *genericEnvironment) valueType(id int) bool {
	if !e.types.nonbasic {
		return true
	}
	v := e.types.get(e.types.underlying(id))
	if v.kind == genericBasic || v.kind == genericParameter || v.kind == genericInvalid {
		return true
	}
	if v.kind == genericInterface {
		return !v.restricted && !v.comparable
	}
	return e.valueTypeSeen(id, make([]bool, len(e.types.items)+1))
}

func (e *genericEnvironment) valueTypeSeen(id int, seen []bool) bool {
	if id <= 0 || id >= len(seen) || seen[id] {
		return true
	}
	seen[id] = true
	v := e.types.get(id)
	if v.kind == genericNamed {
		return e.valueTypeSeen(v.underlying, seen)
	}
	if v.kind == genericInterface {
		return !v.restricted && !v.comparable
	}
	if v.elem != 0 && !e.valueTypeSeen(v.elem, seen) {
		return false
	}
	if v.key != 0 && !e.valueTypeSeen(v.key, seen) {
		return false
	}
	for _, field := range v.fields {
		if !e.valueTypeSeen(field.typ, seen) {
			return false
		}
	}
	for _, typ := range v.params {
		if !e.valueTypeSeen(typ, seen) {
			return false
		}
	}
	for _, typ := range v.results {
		if !e.valueTypeSeen(typ, seen) {
			return false
		}
	}
	return true
}
