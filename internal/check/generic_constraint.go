package check

type genericTerm struct {
	typ   int
	tilde bool
}

// A constraint is an intersection of method requirements, strict
// comparability, and a union of type terms. all distinguishes an unrestricted
// type set from an empty one; both have zero terms.
type genericConstraint struct {
	all        bool
	comparable bool
	terms      []genericTerm
	methods    []genericMethod
}

func (t *genericTypes) interfaceType(c genericConstraint) int {
	terms := append([]genericTerm(nil), c.terms...)
	for i := 1; i < len(terms); i++ {
		v := terms[i]
		j := i
		for j > 0 && (terms[j-1].typ > v.typ || terms[j-1].typ == v.typ && terms[j-1].tilde && !v.tilde) {
			terms[j] = terms[j-1]
			j--
		}
		terms[j] = v
	}
	methods, _ := genericMergeMethods(nil, c.methods)
	return t.intern(genericType{kind: genericInterface, methods: methods, terms: terms, restricted: !c.all, comparable: c.comparable})
}

func (t *genericTypes) interfaceConstraint(id int) genericConstraint {
	v := t.get(t.underlying(id))
	return genericConstraint{all: !v.restricted, comparable: v.comparable, terms: v.terms, methods: v.methods}
}

// Union of interface type sets may overlap. The disjointness restriction is
// checked separately on the non-interface terms written directly in source.
func (t *genericTypes) unionSets(a genericConstraint, b genericConstraint) genericConstraint {
	if a.all || b.all {
		return genericConstraint{all: true}
	}
	result := genericConstraint{terms: append([]genericTerm(nil), a.terms...)}
	for _, term := range b.terms {
		covered := false
		for _, existing := range result.terms {
			if z, ok := t.termIntersection(term, existing); ok && z == term {
				covered = true
				break
			}
		}
		if covered {
			continue
		}
		kept := result.terms[:0]
		for _, existing := range result.terms {
			if z, ok := t.termIntersection(term, existing); !ok || z != existing {
				kept = append(kept, existing)
			}
		}
		result.terms = append(kept, term)
	}
	return result
}

func (t *genericTypes) termContains(term genericTerm, id int) bool {
	if term.tilde {
		return term.typ == t.underlying(id)
	}
	return term.typ == id
}

func (t *genericTypes) termIntersection(a genericTerm, b genericTerm) (genericTerm, bool) {
	if a == b {
		return a, true
	}
	if a.tilde && t.termContains(a, b.typ) {
		return b, true
	}
	if b.tilde && t.termContains(b, a.typ) {
		return a, true
	}
	return genericTerm{}, false
}

// union checks the disjointness rule on source union terms before normalizing
// intersections. In particular, int | ~int and ~int | MyInt are invalid.
func (t *genericTypes) union(terms []genericTerm) (genericConstraint, bool) {
	for i := 0; i < len(terms); i++ {
		v := t.get(terms[i].typ)
		if v.kind == genericInvalid || v.kind == genericParameter || v.kind == genericInterface {
			return genericConstraint{}, false
		}
		if terms[i].tilde && terms[i].typ != t.underlying(terms[i].typ) {
			return genericConstraint{}, false
		}
		for j := 0; j < i; j++ {
			if _, overlap := t.termIntersection(terms[i], terms[j]); overlap {
				return genericConstraint{}, false
			}
		}
	}
	return genericConstraint{terms: append([]genericTerm(nil), terms...)}, true
}

func genericSameMethodName(a genericMethod, b genericMethod) bool {
	return a.name == b.name && a.pkg == b.pkg
}

func genericMethodBefore(a genericMethod, b genericMethod) bool {
	if a.pkg != b.pkg {
		return checkStringAfter(b.pkg, a.pkg)
	}
	return checkStringAfter(b.name, a.name)
}

func genericMergeMethods(a []genericMethod, b []genericMethod) ([]genericMethod, bool) {
	result := append([]genericMethod(nil), a...)
	for _, m := range b {
		found := false
		for _, existing := range result {
			if genericSameMethodName(m, existing) {
				if m.typ != existing.typ {
					return nil, false
				}
				found = true
				break
			}
		}
		if !found {
			result = append(result, m)
		}
	}
	for i := 1; i < len(result); i++ {
		v := result[i]
		j := i
		for j > 0 && genericMethodBefore(v, result[j-1]) {
			result[j] = result[j-1]
			j--
		}
		result[j] = v
	}
	return result, true
}

func (t *genericTypes) intersect(a genericConstraint, b genericConstraint) (genericConstraint, bool) {
	methods, ok := genericMergeMethods(a.methods, b.methods)
	if !ok {
		return genericConstraint{}, false
	}
	r := genericConstraint{all: a.all && b.all, comparable: a.comparable || b.comparable, methods: methods}
	if a.all {
		r.terms = append(r.terms, b.terms...)
	} else if b.all {
		r.terms = append(r.terms, a.terms...)
	} else {
		for _, x := range a.terms {
			for _, y := range b.terms {
				if z, overlaps := t.termIntersection(x, y); overlaps {
					r.terms = append(r.terms, z)
				}
			}
		}
	}
	filtered := make([]genericTerm, 0, len(r.terms))
	for _, term := range r.terms {
		if r.comparable && !t.comparable(term.typ, true) {
			continue
		}
		duplicate := false
		for _, existing := range filtered {
			if term == existing {
				duplicate = true
				break
			}
		}
		if !duplicate {
			filtered = append(filtered, term)
		}
	}
	r.terms = filtered
	return r, true
}

func (t *genericTypes) methodSet(id int) []genericMethod {
	return t.promotedMethods(id)
}

func (t *genericTypes) directMethods(id int, pointer bool) []genericMethod {
	v := t.get(id)
	if v.kind == genericPointer {
		pointer = true
		v = t.get(v.elem)
	}
	if v.kind == genericNamed {
		u := t.get(t.underlying(id))
		if u.kind == genericInterface {
			return u.methods
		}
	}
	if len(v.methods) == 0 {
		return nil
	}
	var out []genericMethod
	for _, method := range v.methods {
		if method.pointer && !pointer {
			method.typ = 0
		}
		method.pointer = false
		out = append(out, method)
	}
	return out
}

func genericHasMethods(have []genericMethod, need []genericMethod) bool {
	for _, m := range need {
		found := false
		for _, h := range have {
			if genericSameMethodName(h, m) && h.typ == m.typ {
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

// satisfies applies the Go 1.20 comparable exception to concrete type
// arguments: an ordinary interface, or a struct containing one, may satisfy
// interface{ comparable; E } where E is a basic interface. Type parameters
// instead use constraintSubset and must guarantee strict comparability.
func (t *genericTypes) satisfies(id int, c genericConstraint) bool {
	if t.get(id).kind == genericInvalid || t.get(id).kind == genericParameter {
		return false
	}
	if !genericHasMethods(t.methodSet(id), c.methods) {
		return false
	}
	if c.comparable && !t.comparable(id, !c.all) {
		return false
	}
	if c.all {
		return true
	}
	for _, term := range c.terms {
		if t.termContains(term, id) {
			return true
		}
	}
	return false
}

func (t *genericTypes) constraintStrictlyComparable(c genericConstraint) bool {
	if c.comparable {
		return true
	}
	if c.all {
		return false
	}
	for _, term := range c.terms {
		if !t.comparable(term.typ, true) {
			return false
		}
	}
	return true
}

func (t *genericTypes) constraintSubset(a genericConstraint, b genericConstraint) bool {
	if !a.all && len(a.terms) == 0 {
		return true
	}
	if b.comparable && !t.constraintStrictlyComparable(a) {
		return false
	}
	if !genericHasMethods(a.methods, b.methods) {
		if a.all {
			return false
		}
		for _, term := range a.terms {
			if term.tilde || !genericHasMethods(t.methodSet(term.typ), b.methods) {
				return false
			}
		}
	}
	if b.all {
		return true
	}
	if a.all {
		return false
	}
	for _, x := range a.terms {
		covered := false
		for _, y := range b.terms {
			if z, ok := t.termIntersection(x, y); ok && z == x {
				covered = true
				break
			}
		}
		if !covered {
			return false
		}
	}
	return true
}

func (t *genericTypes) substituteConstraint(c genericConstraint, parameters []int, arguments []int) genericConstraint {
	r := genericConstraint{all: c.all, comparable: c.comparable}
	for _, term := range c.terms {
		r.terms = append(r.terms, genericTerm{typ: t.substitute(term.typ, parameters, arguments), tilde: term.tilde})
	}
	for _, m := range c.methods {
		m.typ = t.substitute(m.typ, parameters, arguments)
		r.methods = append(r.methods, m)
	}
	return r
}
