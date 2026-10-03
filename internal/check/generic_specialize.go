package check

import "renvo.dev/internal/syntax"

type genericInstance struct {
	declaration int
	arguments   []int
	typ         int
	name        string
}

type genericImport struct {
	into  int
	from  int
	alias string
}

type genericBridge struct {
	pkg          int
	name, target string
	embeddedName string
}

// Specialization operates on checked identities, never spelling-based type
// substitution. An instance is reserved before traversing its body so ordinary
// recursion reuses it. The emitted declarations use only concrete Go types.
type genericSpecializer struct {
	environment *genericEnvironment
	instances   []genericInstance
	imports     []genericImport
	bridges     []genericBridge
	builtins    []string
}

func (s *genericSpecializer) nameUsed(pkg int, name string) bool {
	for i := range s.environment.graph.Packages[pkg].Files {
		file := &s.environment.graph.Packages[pkg].Files[i]
		if genericFileUsesName(&file.File, name) {
			return true
		}
	}
	return false
}

func (s *genericSpecializer) instantiate(declaration int, arguments []int) int {
	e := s.environment
	d := &e.decls[declaration]
	if len(arguments) != len(d.parameters) {
		return -1
	}
	for i, instance := range s.instances {
		if instance.declaration == declaration && genericIDsEqual(instance.arguments, arguments) {
			return i
		}
	}
	e.resolveDeclaration(declaration)
	d = &e.decls[declaration]
	if len(d.constraints) != len(arguments) {
		return -1
	}
	for i, arg := range arguments {
		constraint := e.types.substituteConstraint(d.constraints[i], d.parameters, arguments)
		if !e.satisfies(arg, constraint) {
			return -1
		}
	}
	name := "RenvoGenericInstance_" + genericDecimal(len(s.instances))
	for s.nameUsed(d.pkg, name) {
		name += "_"
	}
	typ := e.types.substitute(d.typ, d.parameters, arguments)
	index := len(s.instances)
	s.instances = append(s.instances, genericInstance{declaration: declaration, arguments: append([]int(nil), arguments...), typ: typ, name: name})
	if d.kind == SymbolType && !d.alias {
		e.attachMethods(typ)
		for i := range e.decls {
			method := &e.decls[i]
			if method.kind == SymbolMethod && method.receiver == declaration+1 {
				s.instantiate(i, arguments)
			}
		}
	}
	return index
}

func (s *genericSpecializer) qualify(into int, from int, name string) string {
	if into == from {
		return name
	}
	for _, imp := range s.imports {
		if imp.into == into && imp.from == from {
			return imp.alias + "." + name
		}
	}
	alias := "__renvo_generic_package_" + genericDecimal(from)
	for s.nameUsed(into, alias) {
		alias += "_"
	}
	s.imports = append(s.imports, genericImport{into: into, from: from, alias: alias})
	return alias + "." + name
}

func (s *genericSpecializer) bridge(pkg int, target string) string {
	for _, bridge := range s.bridges {
		if bridge.pkg == pkg && bridge.target == target {
			return bridge.name
		}
	}
	name := "RenvoGenericType_" + genericDecimal(len(s.bridges))
	for s.nameUsed(pkg, name) {
		name += "_"
	}
	s.bridges = append(s.bridges, genericBridge{pkg: pkg, name: name, target: target})
	return name
}

func (s *genericSpecializer) typeText(id int, into int) string {
	e := s.environment
	v := e.types.get(id)
	privatePackage := ""
	for _, field := range v.fields {
		if field.pkg != "" {
			privatePackage = field.pkg
			break
		}
	}
	if v.kind == genericInterface {
		for _, method := range v.methods {
			if method.pkg != "" {
				privatePackage = method.pkg
				break
			}
		}
	}
	if (v.kind == genericStruct || v.kind == genericInterface) && privatePackage != "" && privatePackage != e.graph.Packages[into].Ref.ImportPath {
		for owner, pkg := range e.graph.Packages {
			if pkg.Ref.ImportPath == privatePackage {
				name := s.bridge(owner, s.typeText(id, owner))
				return s.qualify(into, owner, name)
			}
		}
	}
	switch v.kind {
	case genericBasic:
		if v.name == "unsafe.Pointer" {
			for pkg := range e.graph.Packages {
				if e.graph.Packages[pkg].Ref.ImportPath == "unsafe" {
					return s.qualify(into, pkg, "Pointer")
				}
			}
		}
		return s.builtinTypeText(v.name, into)
	case genericNamed:
		if v.origin == "builtin.error" {
			return s.builtinTypeText("error", into)
		}
		for i := range e.decls {
			d := &e.decls[i]
			if d.kind != SymbolType || d.alias || e.types.get(d.typ).origin != v.origin {
				continue
			}
			if len(d.parameters) == 0 && d.owner == 0 {
				if into != d.pkg && !syntax.IdentifierExported([]byte(d.name), 0) {
					name := s.bridge(d.pkg, d.name)
					return s.qualify(into, d.pkg, name)
				}
				return s.qualify(into, d.pkg, d.name)
			}
			instance := s.instantiate(i, v.args)
			if instance < 0 {
				return ""
			}
			return s.qualify(into, d.pkg, s.instances[instance].name)
		}
	case genericPointer:
		return "*" + s.typeText(v.elem, into)
	case genericSlice:
		return "[]" + s.typeText(v.elem, into)
	case genericArray:
		return "[" + genericUnsignedDecimal(v.length) + "]" + s.typeText(v.elem, into)
	case genericMap:
		return "map[" + s.typeText(v.key, into) + "]" + s.typeText(v.elem, into)
	case genericChan:
		prefix := "chan "
		if v.direction == ChanSendOnly {
			prefix = "chan<- "
		}
		if v.direction == ChanReceiveOnly {
			prefix = "<-chan "
		}
		return prefix + s.typeText(v.elem, into)
	case genericFunc:
		return "func" + s.signatureText(v, into, nil, nil)
	case genericStruct:
		out := "struct { "
		for _, f := range v.fields {
			if !f.embedded {
				out += f.name + " " + s.typeText(f.typ, into)
			} else {
				out += s.embeddedTypeText(f.typ, f.name, f.pkg, into)
			}
			if f.tag != "" {
				out += " " + genericQuoted(f.tag)
			}
			out += "; "
		}
		return out + "}"
	case genericInterface:
		if v.restricted || v.comparable {
			return ""
		}
		out := "interface { "
		for _, m := range v.methods {
			if m.pkg != "" && m.pkg != e.graph.Packages[into].Ref.ImportPath {
				part := e.types.intern(genericType{kind: genericInterface, methods: []genericMethod{m}})
				out += s.typeText(part, into) + "; "
				continue
			}
			out += m.name + s.signatureText(e.types.get(m.typ), into, nil, nil) + "; "
		}
		return out + "}"
	}
	return ""
}

func genericQuoted(value string) string {
	out := "\""
	digits := "0123456789abcdef"
	for i := 0; i < len(value); i++ {
		b := value[i]
		if b == '"' || b == '\\' {
			out += "\\" + string([]byte{b})
		} else if b < 32 || b >= 127 {
			out += "\\x" + string([]byte{digits[int(b)/16], digits[int(b)%16]})
		} else {
			out += string([]byte{b})
		}
	}
	return out + "\""
}

func (s *genericSpecializer) signatureText(signature *genericType, into int, parameters []Field, results []Field) string {
	return s.scopedSignatureText(signature, into, parameters, results, nil)
}

func (s *genericSpecializer) scopedSignatureText(signature *genericType, into int, parameters []Field, results []Field, hidden []string) string {
	out := "("
	for i, id := range signature.params {
		if i > 0 {
			out += ", "
		}
		if i < len(parameters) && parameters[i].Name != "" {
			out += parameters[i].Name + " "
		}
		if signature.variadic && i == len(signature.params)-1 {
			out += "..."
			id = s.environment.types.get(id).elem
		}
		out += s.scopedTypeText(id, into, hidden)
	}
	out += ")"
	if len(signature.results) != 0 {
		out += " ("
		for i, id := range signature.results {
			if i > 0 {
				out += ", "
			}
			if i < len(results) && results[i].Name != "" {
				out += results[i].Name + " "
			}
			out += s.scopedTypeText(id, into, hidden)
		}
		out += ")"
	}
	return out
}

func (s *genericSpecializer) declarationText(index int) string {
	e := s.environment
	instance := s.instances[index]
	d := &e.decls[instance.declaration]
	if d.kind == SymbolType {
		if d.alias {
			return "type " + instance.name + " = " + s.typeText(instance.typ, d.pkg) + "\n"
		}
		file := &e.graph.Packages[d.pkg].Files[d.file].File
		prefix := "//renvo:typename " + genericQuoted(e.typeDisplay(instance.typ, false)) + " " + genericQuoted(d.name) + "\n"
		if syntax.ReflectDirective(file, syntax.TopDecl{Kind: syntax.TokenType, StartTok: d.token}) {
			prefix += "//renvo:reflect\n"
		}
		return prefix + "type " + instance.name + " " + s.typeText(e.types.get(instance.typ).underlying, d.pkg) + "\n"
	}
	file := &e.graph.Packages[d.pkg].Files[d.file].File
	signature := buildFuncSignature(file, &d.function)
	name := instance.name
	if d.kind == SymbolMethod {
		receiver := signature.Receiver[0]
		typ := e.receiverType(d, instance.arguments)
		name = "(" + receiver.Name + " " + s.typeText(typ, d.pkg) + ") " + tokenString(file, d.token)
	}
	out := "func " + name + s.signatureText(e.types.get(instance.typ), d.pkg, signature.Params, signature.Results) + " {\n"
	out += s.parameterAliases(d.parameters, instance.arguments, d.pkg, nil, genericSignatureNames(signature))
	if d.function.BodyStart < 0 {
		return ""
	}
	start := int(file.Tokens[d.function.BodyStart].End)
	end := int(file.Tokens[d.function.BodyEnd-1].Start)
	context := newGenericExpressionContext(s, genericTypeScope{pkg: d.pkg, file: d.file, parameters: d.parameters}, d.function, instance.arguments)
	context.validateBody(e.types.get(instance.typ))
	changes := append([]genericReplacement(nil), context.changes...)
	changes = e.appendArrayLengthChanges(d.pkg, d.file, start, end, changes)
	for i := 0; i < len(changes); i++ {
		changes[i].start -= start
		changes[i].end -= start
	}
	out += applyGenericReplacements(file.Src[start:end], changes) + "\n}\n"
	return out
}

func genericTextUsesIdentifier(text string, name string) bool {
	for i := 0; i < len(text); {
		start := i
		for i < len(text) && (text[i] >= 'a' && text[i] <= 'z' || text[i] >= 'A' && text[i] <= 'Z' || text[i] >= '0' && text[i] <= '9' || text[i] == '_' || text[i] >= 128) {
			i++
		}
		if i > start && text[start:i] == name {
			return true
		}
		if i == start {
			i++
		}
	}
	return false
}

// Local aliases preserve type names without rewriting unrelated field names.
func (s *genericSpecializer) parameterAliases(parameters []int, arguments []int, pkg int, shadowed []string, hidden []string) string {
	e := s.environment
	// Alias-name skipping is separate from protecting its concrete RHS.
	hidden = append(append([]string(nil), hidden...), shadowed...)
	for _, parameter := range parameters {
		hidden = append(hidden, e.types.get(parameter).name)
	}
	out := ""
	for i, parameter := range parameters {
		name := e.types.get(parameter).name
		for _, shadow := range shadowed {
			if shadow == name {
				name = "_"
				break
			}
		}
		if name != "_" {
			text := s.scopedTypeText(arguments[i], pkg, hidden)
			out += "type " + name + " = " + text + "\n"
		}
	}
	return out
}
