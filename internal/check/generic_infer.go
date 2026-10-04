package check

const (
	genericTyped = iota
	genericUntypedBool
	genericUntypedInt
	genericUntypedRune
	genericUntypedFloat
	genericUntypedComplex
	genericUntypedString
	genericUntypedNil
)

type genericArgument struct {
	typ        int
	untyped    int
	function   int // generic declaration index + 1 for an uninstantiated function value
	start, end int
	results    []int
	constant   *genericConstant
	shifted    []*genericConstant // untyped operands awaiting the context of a nonconstant shift
	commaOK    bool               // map index, receive, or assertion in a two-value assignment
}

type genericInference struct {
	types            *genericTypes
	parameters       []int
	arguments        []int
	fixed            int
	legacyInterfaces bool
}

func (s *genericInference) parameter(id int) int {
	for i := 0; i < len(s.parameters); i++ {
		if s.parameters[i] == id {
			return i
		}
	}
	return -1
}

func (s *genericInference) unify(pattern int, actual int, loose bool) bool {
	if pattern == 0 || actual == 0 {
		return false
	}
	if p := s.parameter(pattern); p >= 0 && s.parameter(s.arguments[p]) >= 0 {
		pattern = s.resolved(pattern)
	}
	actual = s.resolved(actual)
	if pattern == actual {
		return true
	}
	if p := s.parameter(pattern); p >= 0 {
		if s.arguments[p] == 0 {
			if s.occurs(pattern, actual) {
				return false
			}
			s.arguments[p] = actual
			return true
		}
		if s.arguments[p] == actual {
			return true
		}
		if other := s.parameter(actual); other >= 0 && s.arguments[other] == 0 {
			s.arguments[other] = s.arguments[p]
			return true
		}
		if !loose {
			return false
		}
		old, next := s.types.get(s.arguments[p]), s.types.get(actual)
		// Prefer a defined type when otherwise identical unnamed and defined
		// operands constrain the same parameter. Explicit arguments are fixed.
		if p >= s.fixed && old.kind != genericNamed && old.kind != genericBasic && next.kind == genericNamed && s.types.underlying(actual) == s.arguments[p] {
			s.arguments[p] = actual
			return true
		}
		return s.types.assignable(actual, s.arguments[p])
	}
	if p := s.parameter(actual); p >= 0 && s.arguments[p] == 0 {
		if s.occurs(actual, pattern) {
			return false
		}
		s.arguments[p] = pattern
		return true
	}
	if pattern == actual {
		return true
	}
	p, a := s.types.get(pattern), s.types.get(actual)
	if p.kind == genericNamed && a.kind == genericNamed && p.origin == a.origin && len(p.args) == len(a.args) {
		for i := 0; i < len(p.args); i++ {
			if !s.unify(p.args[i], a.args[i], false) {
				return false
			}
		}
		return true
	}
	if loose {
		if p.kind == genericNamed && a.kind == genericNamed {
			return false
		}
		p, a = s.types.get(s.types.underlying(pattern)), s.types.get(s.types.underlying(actual))
		if a.kind == genericParameter && a.underlying != 0 {
			a = s.types.get(a.underlying)
		}
	}
	if p.kind != a.kind {
		return loose && s.types.assignable(actual, pattern)
	}
	switch p.kind {
	case genericBasic:
		return p.name == a.name
	case genericPointer, genericSlice, genericArray, genericChan:
		if p.kind == genericArray && p.length != a.length {
			return false
		}
		if p.kind == genericChan && p.direction != a.direction && (!loose || a.direction != ChanBoth) {
			return false
		}
		return s.unify(p.elem, a.elem, false)
	case genericMap:
		return s.unify(p.key, a.key, false) && s.unify(p.elem, a.elem, false)
	case genericFunc:
		if p.variadic != a.variadic || len(p.params) != len(a.params) || len(p.results) != len(a.results) {
			return false
		}
		for i := 0; i < len(p.params); i++ {
			if !s.unify(p.params[i], a.params[i], false) {
				return false
			}
		}
		for i := 0; i < len(p.results); i++ {
			if !s.unify(p.results[i], a.results[i], false) {
				return false
			}
		}
		return true
	case genericStruct:
		if len(p.fields) != len(a.fields) {
			return false
		}
		for i := 0; i < len(p.fields); i++ {
			x, y := p.fields[i], a.fields[i]
			if x.name != y.name || x.pkg != y.pkg || x.tag != y.tag || x.embedded != y.embedded || !s.unify(x.typ, y.typ, false) {
				return false
			}
		}
		return true
	case genericInterface:
		// Interface methods also provide inference equations, including when
		// the argument is a named concrete type implementing the interface.
		return (!s.legacyInterfaces || len(p.methods) == len(a.methods)) && s.unifyMethods(p.methods, a.methods)
	}
	return false
}

// Follow existing equations as well as the proposed type. A cycle such as
// T=[]U, U=*T has no finite solution and must fail before substitution grows it.
func (s *genericInference) occurs(parameter int, typ int) bool {
	pending := []int{typ}
	for next := 0; next < len(pending); next++ {
		id := pending[next]
		if s.types.containsAnyParameter(id, []int{parameter}, nil) {
			return true
		}
		for i, bound := range s.arguments {
			if bound == 0 || !s.types.containsAnyParameter(id, s.parameters[i:i+1], nil) {
				continue
			}
			seen := false
			for _, old := range pending {
				if old == bound {
					seen = true
					break
				}
			}
			if !seen {
				pending = append(pending, bound)
			}
		}
	}
	return false
}

func (s *genericInference) resolved(id int) int {
	for count := 0; count < len(s.parameters); count++ {
		p := s.parameter(id)
		if p < 0 || s.arguments[p] == 0 || s.arguments[p] == id {
			return id
		}
		id = s.arguments[p]
	}
	return id
}

func (s *genericInference) unifyMethods(need []genericMethod, have []genericMethod) bool {
	for _, m := range need {
		found := false
		for _, h := range have {
			if genericSameMethodName(m, h) {
				if !s.unify(m.typ, h.typ, false) {
					return false
				}
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func (t *genericTypes) assignable(from int, to int) bool {
	if from == 0 || to == 0 {
		return false
	}
	if from == to {
		return true
	}
	f, d := t.get(from), t.get(to)
	fu, du := t.underlying(from), t.underlying(to)
	if fu == 0 || du == 0 {
		return false
	}
	if fu == du && (f.kind != genericNamed && f.kind != genericBasic || d.kind != genericNamed && d.kind != genericBasic) {
		return true
	}
	f, d = t.get(fu), t.get(du)
	if d.kind == genericInterface {
		return genericHasMethods(t.methodSet(from), d.methods)
	}
	if f.kind == genericChan && d.kind == genericChan && f.elem == d.elem && f.direction == ChanBoth && (t.get(from).kind != genericNamed || t.get(to).kind != genericNamed) {
		return true
	}
	return false
}

func (t *genericTypes) infer(parameters []int, constraints []genericConstraint, signature int, explicit []int, actual []genericArgument, spread bool, expected int) ([]int, bool) {
	arguments, ok := t.inferArguments(parameters, constraints, signature, explicit, actual, spread, expected)
	if !ok {
		return nil, false
	}
	for i, id := range arguments {
		constraint := t.substituteConstraint(constraints[i], parameters, arguments)
		if !t.satisfies(id, constraint) {
			return nil, false
		}
	}
	return arguments, true
}

func (t *genericTypes) inferArguments(parameters []int, constraints []genericConstraint, signature int, explicit []int, actual []genericArgument, spread bool, expected int) ([]int, bool) {
	return t.inferArgumentsVersion(parameters, constraints, signature, explicit, actual, spread, expected, false)
}

func (t *genericTypes) inferArgumentsVersion(parameters []int, constraints []genericConstraint, signature int, explicit []int, actual []genericArgument, spread bool, expected int, legacyInterfaces bool) ([]int, bool) {
	if len(explicit) > len(parameters) || len(constraints) != len(parameters) {
		return nil, false
	}
	s := genericInference{types: t, parameters: parameters, arguments: make([]int, len(parameters)), fixed: len(explicit), legacyInterfaces: legacyInterfaces}
	copy(s.arguments, explicit)
	fn := t.get(signature)
	if fn.kind != genericFunc {
		return nil, false
	}
	if expected != 0 {
		if !s.unify(signature, t.underlying(expected), false) {
			return nil, false
		}
	} else {
		if !fn.variadic && (spread || len(actual) != len(fn.params)) {
			return nil, false
		}
		if fn.variadic && (len(actual) < len(fn.params)-1 || spread && len(actual) != len(fn.params)) {
			return nil, false
		}
		// Typed arguments constrain inference before defaulting constants.
		for i, a := range actual {
			if a.untyped != genericTyped || a.function != 0 {
				continue
			}
			p := genericArgumentParameter(t, fn, i, spread)
			pattern := t.get(t.underlying(p))
			if pattern.kind == genericInterface {
				if legacyInterfaces {
					argument := t.get(t.underlying(a.typ))
					if argument.kind != genericInterface {
						// Before Go 1.21, a concrete method set did not infer
						// an interface parameter's type arguments. Other
						// arguments may still determine them for assignment.
						continue
					}
					if len(pattern.methods) != len(argument.methods) {
						return nil, false
					}
				}
				if !s.unifyMethods(pattern.methods, t.methodSet(a.typ)) {
					return nil, false
				}
			} else if !s.unify(p, a.typ, true) {
				return nil, false
			}
		}
	}
	if !s.inferConstraints(constraints) {
		return nil, false
	}
	defaults := make([]int, len(parameters))
	if expected == 0 {
		for i, a := range actual {
			if a.untyped == genericTyped || a.untyped == genericUntypedNil {
				continue
			}
			p := s.parameter(s.resolved(genericArgumentParameter(t, fn, i, spread)))
			if p < 0 || s.arguments[p] != 0 {
				continue
			}
			kind, ok := genericMergeUntyped(defaults[p], a.untyped)
			if !ok {
				return nil, false
			}
			defaults[p] = kind
		}
	}
	for i, kind := range defaults {
		if kind != 0 {
			s.arguments[i] = t.defaultType(kind)
		}
	}
	if !s.inferConstraints(constraints) {
		return nil, false
	}
	for i, id := range s.arguments {
		if id == 0 {
			s.arguments[i] = parameters[i]
		}
	}
	for round := 0; round < len(parameters); round++ {
		before := append([]int(nil), s.arguments...)
		for i, id := range before {
			if id != 0 {
				s.arguments[i] = t.substitute(id, parameters, before)
			}
		}
		if genericIDsEqual(before, s.arguments) {
			break
		}
	}
	for _, id := range s.arguments {
		if id == 0 || t.containsAnyParameter(id, parameters, nil) {
			return nil, false
		}
	}
	return s.arguments, true
}

func (t *genericTypes) containsAnyParameter(id int, parameters []int, seen []int) bool {
	for _, p := range parameters {
		if id == p {
			return true
		}
	}
	for _, prior := range seen {
		if prior == id {
			return false
		}
	}
	seen = append(seen, id)
	v := t.get(id)
	children := append([]int{v.elem, v.key}, v.args...)
	children = append(children, v.params...)
	children = append(children, v.results...)
	for _, field := range v.fields {
		children = append(children, field.typ)
	}
	for _, method := range v.methods {
		children = append(children, method.typ)
	}
	for _, term := range v.terms {
		children = append(children, term.typ)
	}
	for _, child := range children {
		if child != 0 && t.containsAnyParameter(child, parameters, seen) {
			return true
		}
	}
	return false
}

func genericArgumentParameter(t *genericTypes, fn *genericType, index int, spread bool) int {
	if fn.variadic && index >= len(fn.params)-1 {
		last := fn.params[len(fn.params)-1]
		if spread {
			return last
		}
		return t.get(last).elem
	}
	if index < len(fn.params) {
		return fn.params[index]
	}
	return 0
}

func genericMergeUntyped(a int, b int) (int, bool) {
	if a == 0 || a == b {
		return b, true
	}
	if a >= genericUntypedInt && a <= genericUntypedComplex && b >= genericUntypedInt && b <= genericUntypedComplex {
		if a > b {
			return a, true
		}
		return b, true
	}
	return 0, false
}

func (t *genericTypes) defaultType(kind int) int {
	switch kind {
	case genericUntypedBool:
		return t.basic("bool")
	case genericUntypedInt:
		return t.basic("int")
	case genericUntypedRune:
		return t.basic("rune")
	case genericUntypedFloat:
		return t.basic("float64")
	case genericUntypedComplex:
		return t.basic("complex128")
	case genericUntypedString:
		return t.basic("string")
	}
	return 0
}

func (s *genericInference) inferConstraints(constraints []genericConstraint) bool {
	for round := 0; round <= len(s.parameters); round++ {
		before := append([]int(nil), s.arguments...)
		for i, c := range constraints {
			actual := s.arguments[i]
			if actual == 0 {
				continue
			}
			if len(c.terms) == 1 {
				term := c.terms[0]
				if !s.unify(term.typ, actual, term.tilde) {
					return false
				}
			} else if core := s.types.coreType(c); core != 0 {
				if !s.unify(core, actual, true) {
					return false
				}
			}
			if !s.unifyMethods(c.methods, s.types.methodSet(actual)) {
				return false
			}
		}
		if genericIDsEqual(before, s.arguments) {
			return true
		}
	}
	return false
}
