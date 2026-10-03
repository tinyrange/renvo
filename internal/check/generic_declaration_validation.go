package check

func (e *genericEnvironment) validatePackageNames() {
	for p := range e.graph.Packages {
		var names []string
		for f := range e.graph.Packages[p].Files {
			file := &e.graph.Packages[p].Files[f].File
			var tokens []int
			for _, decl := range file.Decls {
				tokens = append(tokens, decl.NameTok)
			}
			for _, fn := range file.Funcs {
				if fn.ReceiverStart < 0 && !tokenTextIs(file, fn.NameTok, "init") {
					tokens = append(tokens, fn.NameTok)
				}
			}
			for _, token := range tokens {
				name := tokenString(file, token)
				if name == "_" {
					continue
				}
				for _, old := range names {
					if old == name {
						e.fail(genericTypeScope{pkg: p, file: f}, token, "duplicate package declaration")
					}
				}
				names = append(names, name)
			}
		}
	}
}

func (e *genericEnvironment) validateParameterNames(d *genericDeclaration) {
	file := &e.graph.Packages[d.pkg].Files[d.file].File
	var names []string
	for _, p := range d.parameters {
		if name := e.types.get(p).name; name != "_" {
			names = append(names, name)
		}
	}
	signature := buildFuncSignature(file, &d.function)
	fields := append([]Field(nil), signature.Receiver...)
	fields = append(fields, signature.Params...)
	fields = append(fields, signature.Results...)
	for _, field := range fields {
		if field.Name == "" || field.Name == "_" {
			continue
		}
		for _, name := range names {
			if name == field.Name {
				e.fail(genericTypeScope{pkg: d.pkg, file: d.file}, field.NameTok, "duplicate parameter name")
			}
		}
		names = append(names, field.Name)
	}
}

// Reuse ordinary lexical resolution before a template can be discarded. Its
// type parameters are local type names; its value names obey the same rules
// as a concrete function, including unused locals and imported visibility.
func (e *genericEnvironment) validateGenericScope(d *genericDeclaration) {
	if e.headers == nil {
		for pkg := range e.graph.Packages {
			info, ok, _, file, token := checkPackageHeader(*e.graph, pkg)
			if !ok {
				e.fail(genericTypeScope{pkg: pkg, file: file}, token, "invalid generic package declaration")
				return
			}
			info.CoreSymbolHash = buildCoreSymbolHash(info.Symbols)
			e.headers = append(e.headers, info)
		}
	}
	if len(e.headers) != len(e.graph.Packages) {
		return
	}
	file := &e.graph.Packages[d.pkg].Files[d.file].File
	scope, ok, token := buildFuncScopeCore(file, &d.function)
	where := genericTypeScope{pkg: d.pkg, file: d.file}
	if !ok {
		e.fail(where, token, "invalid generic function scope")
		return
	}
	ordinary := scope.Names
	scope = CoreScope{}
	for _, parameter := range d.parameters {
		name := e.types.get(parameter).name
		for token := d.function.StartTok; token < d.function.BodyStart; token++ {
			if tokenTextIs(file, token, name) {
				addCoreScopeName(&scope, file, token, NameLocal, false, false, false)
				break
			}
		}
	}
	for _, name := range ordinary {
		addCoreScopeName(&scope, file, name.Token, name.Kind, false, false, false)
		scope.Names[len(scope.Names)-1].End = name.End
	}
	_, _, undefined := appendResolutionRefsCore(nil, nil, file, d.file, &e.headers[d.pkg], e.headers, &scope, d.function.BodyStart+1, d.function.BodyEnd-1, nil)
	if undefined >= 0 {
		e.fail(where, undefined, "undefined name in generic definition")
	}
	if unused := unusedCoreLocalToken(&scope); unused >= 0 {
		e.fail(where, unused, "unused local in generic definition")
	}
}
