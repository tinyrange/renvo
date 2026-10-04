package check

// Compiler annotations retain semantic names after specialization. They are
// ordinary comments, so source lowering and frontend caching preserve them
// without extending the backend's concrete type contract.
func (e *genericEnvironment) typeDisplay(id int, qualified bool) string {
	v := e.types.get(id)
	switch v.kind {
	case genericBasic, genericParameter:
		return v.name
	case genericNamed:
		name := v.name
		if qualified {
			for _, decl := range e.decls {
				if decl.kind == SymbolType && e.types.get(decl.typ).origin == v.origin {
					pkg := &e.graph.Packages[decl.pkg]
					path := pkg.Ref.ImportPath
					if pkg.Name == "main" {
						path = "main"
					}
					name = path + "." + name
					break
				}
			}
		}
		if len(v.args) > 0 {
			name += "["
			for i, arg := range v.args {
				if i > 0 {
					name += ","
				}
				name += e.typeDisplay(arg, true)
			}
			name += "]"
		}
		return name
	case genericPointer:
		return "*" + e.typeDisplay(v.elem, true)
	case genericSlice:
		return "[]" + e.typeDisplay(v.elem, true)
	case genericArray:
		return "[" + genericUnsignedDecimal(v.length) + "]" + e.typeDisplay(v.elem, true)
	case genericMap:
		return "map[" + e.typeDisplay(v.key, true) + "]" + e.typeDisplay(v.elem, true)
	case genericChan:
		prefix := "chan "
		if v.direction == ChanSendOnly {
			prefix = "chan<- "
		} else if v.direction == ChanReceiveOnly {
			prefix = "<-chan "
		}
		return prefix + e.typeDisplay(v.elem, true)
	case genericFunc:
		return "func" + e.signatureDisplay(v)
	case genericStruct:
		if len(v.fields) == 0 {
			return "struct {}"
		}
		text := "struct {"
		for i, field := range v.fields {
			if i > 0 {
				text += ";"
			}
			text += " "
			fieldName := e.memberDisplay(field.name, field.pkg)
			if field.embedded {
				base := e.types.get(field.typ)
				if base.kind == genericPointer {
					base = e.types.get(base.elem)
				}
				if base.kind != genericNamed || base.name != field.name {
					text += fieldName + " = "
				}
			} else {
				text += fieldName + " "
			}
			text += e.typeDisplay(field.typ, true)
			if field.tag != "" {
				text += " " + genericQuoted(field.tag)
			}
		}
		return text + " }"
	case genericInterface:
		if len(v.methods) == 0 {
			return "interface {}"
		}
		// Runtime names order methods by name, independently of the identity
		// table's package-first ordering for private method sets.
		methods := append([]genericMethod(nil), v.methods...)
		for i := 1; i < len(methods); i++ {
			method, j := methods[i], i
			for j > 0 && (checkStringAfter(methods[j-1].name, method.name) || methods[j-1].name == method.name && checkStringAfter(methods[j-1].pkg, method.pkg)) {
				methods[j] = methods[j-1]
				j--
			}
			methods[j] = method
		}
		text := "interface {"
		for i, method := range methods {
			if i > 0 {
				text += ";"
			}
			text += " " + e.memberDisplay(method.name, method.pkg) + e.signatureDisplay(e.types.get(method.typ))
		}
		return text + " }"
	}
	return ""
}

func (e *genericEnvironment) memberDisplay(name string, path string) string {
	if path != "" {
		for _, pkg := range e.graph.Packages {
			if pkg.Ref.ImportPath == path {
				return pkg.Name + "." + name
			}
		}
	}
	return name
}

func (e *genericEnvironment) signatureDisplay(v *genericType) string {
	text := "("
	for i, param := range v.params {
		if i > 0 {
			text += ", "
		}
		if v.variadic && i == len(v.params)-1 {
			text += "..."
			param = e.types.get(param).elem
		}
		text += e.typeDisplay(param, true)
	}
	text += ")"
	if len(v.results) == 1 {
		text += " " + e.typeDisplay(v.results[0], true)
	} else if len(v.results) > 1 {
		text += " ("
		for i, result := range v.results {
			if i > 0 {
				text += ", "
			}
			text += e.typeDisplay(result, true)
		}
		text += ")"
	}
	return text
}
