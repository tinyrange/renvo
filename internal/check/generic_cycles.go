package check

type genericInstantiationEdge struct {
	from    int
	to      int
	growing bool
	scope   genericTypeScope
	token   int
}

func (e *genericEnvironment) checkSizedTypes() {
	states := make([]int, len(e.types.items)+1)
	for declIndex := range e.decls {
		decl := &e.decls[declIndex]
		if decl.kind == SymbolType && len(decl.parameters) > 0 && !e.sizedType(decl.typ, states) {
			e.fail(genericTypeScope{pkg: decl.pkg, file: decl.file}, decl.token, "invalid recursive type")
		}
	}
}

func (e *genericEnvironment) sizedType(id int, states []int) bool {
	if id <= 0 || id >= len(states) || states[id] == 2 {
		return true
	}
	if states[id] == 1 {
		return false
	}
	states[id] = 1
	v := e.types.get(id)
	if v.kind == genericNamed && !e.sizedType(v.underlying, states) {
		return false
	}
	if v.kind == genericArray && !e.sizedType(v.elem, states) {
		return false
	}
	if v.kind == genericStruct {
		for _, field := range v.fields {
			if !e.sizedType(field.typ, states) {
				return false
			}
		}
	}
	states[id] = 2
	return true
}

// A generic call connects each source parameter to the destination parameters
// whose arguments contain it. A constructor on an edge means that traversing
// a cycle grows a type indefinitely. Fixed arguments have no parameter edge;
// F[T] calling F[int] is finite and must remain valid.
func (e *genericEnvironment) recordInstantiation(scope genericTypeScope, token int, parameters []int, arguments []int) {
	if len(parameters) != len(arguments) {
		return
	}
	for i, argument := range arguments {
		e.recordTypeEdges(scope, token, parameters[i], argument, false, 0)
	}
}

func (e *genericEnvironment) recordTypeEdges(scope genericTypeScope, token int, to int, argument int, growing bool, depth int) {
	if depth > len(e.types.items) {
		return
	}
	v := e.types.get(argument)
	if v.kind == genericParameter {
		for _, p := range scope.parameters {
			if argument != p {
				continue
			}
			for _, old := range e.edges {
				if old.from == p && old.to == to && old.growing == growing {
					return
				}
			}
			e.edges = append(e.edges, genericInstantiationEdge{from: p, to: to, growing: growing, scope: scope, token: token})
		}
		return
	}
	if v.kind == genericNamed {
		for _, arg := range v.args {
			e.recordTypeEdges(scope, token, to, arg, true, depth+1)
		}
		return
	}
	if v.elem != 0 {
		e.recordTypeEdges(scope, token, to, v.elem, true, depth+1)
	}
	if v.key != 0 {
		e.recordTypeEdges(scope, token, to, v.key, true, depth+1)
	}
	for _, arg := range v.params {
		e.recordTypeEdges(scope, token, to, arg, true, depth+1)
	}
	for _, arg := range v.results {
		e.recordTypeEdges(scope, token, to, arg, true, depth+1)
	}
	for _, f := range v.fields {
		e.recordTypeEdges(scope, token, to, f.typ, true, depth+1)
	}
	for _, m := range v.methods {
		e.recordTypeEdges(scope, token, to, m.typ, true, depth+1)
	}
}

func (e *genericEnvironment) checkInstantiationCycles() {
	// Only constructor edges can make a recursive instantiation grow. Build
	// outgoing links once, and reuse visit marks across those reachability walks.
	// Scanning every edge and clearing every type for each walk is quadratic
	// even when the parameter graph consists of many unrelated declarations.
	growing := false
	for _, edge := range e.edges {
		if edge.growing {
			growing = true
			break
		}
	}
	if !growing {
		return
	}
	heads := make([]int, len(e.types.items)+1)
	next := make([]int, len(e.edges))
	visited := make([]int, len(heads))
	for i, edge := range e.edges {
		next[i] = heads[edge.from]
		heads[edge.from] = i + 1
	}
	for i, edge := range e.edges {
		if !edge.growing {
			continue
		}
		mark := i + 1
		pending := []int{edge.to}
		for len(pending) > 0 {
			id := pending[len(pending)-1]
			pending = pending[:len(pending)-1]
			if id == edge.from {
				e.fail(edge.scope, edge.token, "recursive instantiation grows type arguments")
				return
			}
			if visited[id] == mark {
				continue
			}
			visited[id] = mark
			for link := heads[id]; link != 0; link = next[link-1] {
				pending = append(pending, e.edges[link-1].to)
			}
		}
	}
}
