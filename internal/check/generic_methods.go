package check

import "renvo.dev/internal/syntax"

func (e *genericEnvironment) collectMethods() {
	for p := 0; p < len(e.graph.Packages); p++ {
		for f := 0; f < len(e.graph.Packages[p].Files); f++ {
			file := &e.graph.Packages[p].Files[f].File
			for _, fn := range file.Funcs {
				if fn.ReceiverStart < 0 {
					continue
				}
				base := e.lookup(p, receiverTypeName(file, &fn))
				if base < 0 || e.decls[base].kind != SymbolType {
					continue
				}
				if e.decls[base].alias && len(e.decls[base].parameters) > 0 {
					e.fail(genericTypeScope{pkg: p, file: f}, fn.ReceiverStart, "generic alias cannot be a receiver type")
					continue
				}
				if e.decls[base].alias {
					id := e.resolveDeclaration(base)
					if e.types.get(id).kind == genericPointer {
						id = e.types.get(id).elem
					}
					if len(e.types.get(id).args) > 0 {
						e.fail(genericTypeScope{pkg: p, file: f}, fn.ReceiverStart, "alias to instantiated generic type cannot be a receiver")
						continue
					}
					for i := range e.decls {
						if e.decls[i].kind == SymbolType && !e.decls[i].alias && e.decls[i].pkg == p && e.decls[i].typ == id {
							base = i
							break
						}
					}
				}
				d := genericDeclaration{pkg: p, file: f, token: fn.NameTok, kind: SymbolMethod, name: e.decls[base].name + "." + tokenString(file, fn.NameTok), function: fn, receiver: base + 1}
				if e.lookup(p, d.name) >= 0 {
					e.fail(genericTypeScope{pkg: p, file: f}, fn.NameTok, "duplicate method")
				}
				open := findTypeTopLevelChar(file, fn.ReceiverStart, fn.ReceiverEnd, '[')
				if len(e.decls[base].parameters) > 0 {
					if open < 0 {
						e.fail(genericTypeScope{pkg: p, file: f}, fn.ReceiverStart, "generic receiver requires type parameters")
						continue
					}
					close := findTypeMatching(file, open, '[', ']')
					for i := open + 1; i < close-1; {
						end := nextTopLevelComma(file, i, close-1)
						if end != i+1 || file.Tokens[i].KindLine&255 != syntax.TokenIdent {
							e.fail(genericTypeScope{pkg: p, file: f}, i, "receiver type parameter must be an identifier")
							break
						}
						name := tokenString(file, i)
						for _, old := range d.parameters {
							if name != "_" && e.types.get(old).name == name {
								e.fail(genericTypeScope{pkg: p, file: f}, i, "duplicate receiver type parameter")
							}
						}
						origin := e.graph.Packages[p].Ref.ImportPath + "." + d.name + "/" + genericDecimal(len(d.parameters))
						d.parameters = append(d.parameters, e.types.intern(genericType{kind: genericParameter, name: name, origin: origin}))
						i = end + 1
					}
					if len(d.parameters) != len(e.decls[base].parameters) {
						e.fail(genericTypeScope{pkg: p, file: f}, fn.ReceiverStart, "wrong number of receiver type parameters")
					}
				}
				e.decls = append(e.decls, d)
				if len(d.parameters) > 0 {
					e.recordInstantiation(genericTypeScope{pkg: p, file: f, parameters: e.decls[base].parameters}, fn.ReceiverStart, d.parameters, e.decls[base].parameters)
				}
			}
		}
	}
}

func (e *genericEnvironment) receiverType(d *genericDeclaration, arguments []int) int {
	base := &e.decls[d.receiver-1]
	typ := e.types.substitute(e.resolveDeclaration(d.receiver-1), base.parameters, d.parameters)
	if len(arguments) > 0 {
		typ = e.types.substitute(typ, d.parameters, arguments)
	}
	file := &e.graph.Packages[d.pkg].Files[d.file].File
	fields := parseFieldList(file, d.function.ReceiverStart, d.function.ReceiverEnd)
	if len(fields) == 1 && tokCharIs(file, fields[0].TypeStart, '*') {
		typ = e.types.intern(genericType{kind: genericPointer, elem: typ})
	}
	return typ
}

func (c *genericExpressionContext) addressable(start int, end int, before int) bool {
	e := c.specializer.environment
	file := &e.graph.Packages[c.scope.pkg].Files[c.scope.file].File
	start, end = stripOuterParens(file, start, end)
	if start >= end {
		return false
	}
	packageIndex, name := c.scope.pkg, start
	if end == start+3 && tokCharIs(file, start+1, '.') && c.binding(start, before) < 0 {
		packageIndex = e.imported(c.scope, tokenString(file, start))
		name = start + 2
	}
	if packageIndex >= 0 && end == name+1 {
		if index := c.binding(start, before); index >= 0 {
			return c.bindings[index].writable
		}
		if name == start && !e.packageValueName(packageIndex, tokenString(file, name)) {
			if imported := e.dotImportedValuePackage(c.scope, tokenString(file, name)); imported >= 0 {
				packageIndex = imported
			}
		}
		for _, source := range e.graph.Packages[packageIndex].Files {
			for _, decl := range source.File.Decls {
				if decl.Kind == syntax.TokenVar && tokenString(&source.File, decl.NameTok) == tokenString(file, name) {
					return true
				}
			}
		}
	}
	if tokCharIs(file, start, '*') {
		return true
	}
	if end >= start+3 && tokCharIs(file, end-2, '.') {
		base := c.expression(start, end-2, before)
		return e.types.get(e.coreType(base.typ)).kind == genericPointer || c.addressable(start, end-2, before)
	}
	if tokCharIs(file, end-1, ']') {
		open := genericMatchingOpen(file, start, end-1, '[', ']')
		if open > start {
			value := c.expression(start, open, before)
			v := e.types.get(e.coreType(value.typ))
			return v.kind == genericSlice || v.kind == genericPointer && e.types.get(e.coreType(v.elem)).kind == genericArray || v.kind == genericArray && c.addressable(start, open, before)
		}
	}
	return false
}

func (c *genericExpressionContext) expressionType(start int, end int) int {
	e := c.specializer.environment
	file := &e.graph.Packages[c.scope.pkg].Files[c.scope.file].File
	start, end = stripOuterParens(file, start, end)
	if start >= end {
		return 0
	}
	kind := file.Tokens[start].KindLine & 255
	if (kind == syntax.TokenStruct || kind == syntax.TokenInterface) && tokCharIs(file, start+1, '{') && findTypeMatching(file, start+1, '{', '}') == end {
		return c.typeSpan(start, end)
	}
	if tokCharIs(file, start, '*') {
		if element := c.expressionType(start+1, end); element != 0 {
			return e.types.intern(genericType{kind: genericPointer, elem: element})
		}
		return 0
	}
	if c.binding(start, start) >= 0 {
		return 0
	}
	for _, p := range c.scope.parameters {
		if end == start+1 && tokenString(file, start) == e.types.get(p).name {
			return c.typeSpan(start, end)
		}
	}
	if d, next := c.declaration(start, end); d >= 0 && e.decls[d].kind == SymbolType && (next == end || tokCharIs(file, next, '[') && findTypeMatching(file, next, '[', ']') == end) {
		return c.typeSpan(start, end)
	}
	return 0
}

func (e *genericEnvironment) attachMethods(id int) {
	e.attachEmbeddedMethods(id, nil)
}

func (e *genericEnvironment) attachEmbeddedMethods(id int, seen []int) {
	if e.types.get(id).kind == genericPointer {
		id = e.types.get(id).elem
	}
	for _, previous := range seen {
		if id == previous {
			return
		}
	}
	seen = append(seen, id)
	e.attachOwnMethods(id)
	u := e.types.get(e.types.underlying(id))
	if u.kind == genericStruct {
		for _, field := range u.fields {
			if field.embedded {
				e.attachEmbeddedMethods(field.typ, seen)
			}
		}
	}
}

func (e *genericEnvironment) attachOwnMethods(id int) {
	v := e.types.get(id)
	if v.kind == genericPointer {
		id = v.elem
		v = e.types.get(id)
	}
	if v.kind != genericNamed {
		return
	}
	base := -1
	for i := range e.decls {
		d := &e.decls[i]
		if d.kind == SymbolType && !d.alias && e.types.get(d.typ).origin == v.origin {
			base = i
			break
		}
	}
	if base < 0 {
		return
	}
	var methods []genericMethod
	for i := range e.decls {
		d := &e.decls[i]
		if d.kind != SymbolMethod || d.receiver != base+1 {
			continue
		}
		if d.state == 1 {
			continue
		}
		signature := e.resolveDeclaration(i)
		file := &e.graph.Packages[d.pkg].Files[d.file].File
		name := tokenString(file, d.token)
		pkg := ""
		if !syntax.IdentifierExported([]byte(name), 0) {
			pkg = e.graph.Packages[d.pkg].Ref.ImportPath
		}
		receiver := parseFieldList(file, d.function.ReceiverStart, d.function.ReceiverEnd)
		pointer := len(receiver) == 1 && tokCharIs(file, receiver[0].TypeStart, '*')
		if len(d.parameters) != 0 {
			signature = e.types.substitute(signature, d.parameters, v.args)
		}
		methods = append(methods, genericMethod{name: name, pkg: pkg, typ: signature, pointer: pointer})
	}
	e.types.items[id-1].methods = methods
}

func (e *genericEnvironment) functionDeclaration(pkg int, file int, token int) int {
	for i := range e.decls {
		d := &e.decls[i]
		if d.kind != SymbolType && d.pkg == pkg && d.file == file && d.token == token {
			return i
		}
	}
	return -1
}
