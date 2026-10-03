package check

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

type genericDeclaration struct {
	pkg         int
	file        int
	token       int
	kind        int
	name        string
	start       int
	end         int
	alias       bool
	state       int
	typ         int
	parameters  []int
	constraints []genericConstraint
	list        syntax.TypeParamList
	function    syntax.FuncDecl
	receiver    int // receiver declaration index + 1 for a method
	owner       int // enclosing function declaration index + 1 for a local type
	scopeEnd    int
}

type genericEnvironment struct {
	graph            *load.Graph
	wordBits         int
	pointerBits      int
	types            genericTypes
	decls            []genericDeclaration
	errorText        string
	errorPkg         int
	errorFile        int
	errorToken       int
	edges            []genericInstantiationEdge
	typeUses         []genericTypeUse
	headers          []PackageInfo
	arrayEvaluations []genericArrayLengthEvaluation
	arrayLengths     []genericArrayLengthSource
	lexicalNames     []genericLexicalBindings
}

type genericTypeUse struct {
	scope                   genericTypeScope
	token, declaration, key int
	arguments               []int
}

func (e *genericEnvironment) validateTypeUses() {
	for i := 0; i < len(e.typeUses); i++ {
		use := e.typeUses[i]
		if use.key != 0 {
			if !e.satisfies(use.key, genericConstraint{all: true, comparable: true}) {
				e.fail(use.scope, use.token, "map key must be comparable")
			}
			continue
		}
		e.resolveDeclaration(use.declaration)
		d := &e.decls[use.declaration]
		for i, arg := range use.arguments {
			if i >= len(d.constraints) || !e.argumentSatisfies(use.scope, arg, e.types.substituteConstraint(d.constraints[i], d.parameters, use.arguments)) {
				e.fail(use.scope, use.token, "type argument does not satisfy constraint")
			}
		}
	}
}

type genericTypeScope struct {
	pkg        int
	file       int
	parameters []int
	bindings   []scopedTypeBinding
}

func newGenericEnvironment(graph *load.Graph) *genericEnvironment {
	e := &genericEnvironment{graph: graph, errorPkg: -1, errorFile: -1, errorToken: -1}
	e.wordBits, e.pointerBits = graph.Layout.WordBits, graph.Layout.PointerBits
	if e.wordBits == 0 {
		e.wordBits = 64
	}
	if e.pointerBits == 0 {
		e.pointerBits = e.wordBits
	}
	for p := 0; p < len(graph.Packages); p++ {
		pkg := &graph.Packages[p]
		for f := 0; f < len(pkg.Files); f++ {
			file := &pkg.Files[f].File
			for _, d := range file.Decls {
				if d.Kind != syntax.TokenType {
					continue
				}
				list := syntax.TypeParameters(file, d.NameTok)
				start := d.NameTok + 1
				if list.EndTok > start {
					start = list.EndTok
				}
				alias := tokCharIs(file, start, '=')
				if alias {
					start++
				}
				e.decls = append(e.decls, genericDeclaration{pkg: p, file: f, token: d.NameTok, kind: SymbolType, name: tokenString(file, d.NameTok), start: start, end: d.EndTok, alias: alias, list: list})
			}
			for _, fn := range file.Funcs {
				if fn.ReceiverStart >= 0 {
					continue
				}
				e.decls = append(e.decls, genericDeclaration{pkg: p, file: f, token: fn.NameTok, kind: SymbolFunc, name: tokenString(file, fn.NameTok), function: fn, list: syntax.TypeParameters(file, fn.NameTok)})
			}
		}
	}
	// Reserve declaration identities before resolving recursive definitions.
	for i := 0; i < len(e.decls); i++ {
		d := &e.decls[i]
		file := &graph.Packages[d.pkg].Files[d.file].File
		origin := graph.Packages[d.pkg].Ref.ImportPath + "." + d.name
		if d.name == "_" {
			origin += "/" + genericDecimal(d.file) + "/" + genericDecimal(d.token)
		}
		for _, p := range d.list.Parameters {
			name := tokenString(file, p.NameTok)
			for _, prior := range d.parameters {
				if name != "_" && e.types.get(prior).name == name {
					e.fail(genericTypeScope{pkg: d.pkg, file: d.file}, p.NameTok, "duplicate type parameter")
				}
			}
			id := e.types.intern(genericType{kind: genericParameter, name: name, origin: origin + "/" + genericDecimal(len(d.parameters))})
			d.parameters = append(d.parameters, id)
		}
		if d.kind == SymbolType && !d.alias {
			d.typ = e.types.intern(genericType{kind: genericNamed, name: d.name, origin: origin, args: d.parameters})
		}
	}
	e.collectMethods()
	e.collectLocalTypes()
	return e
}

func genericDecimal(n int) string {
	if n == 0 {
		return "0"
	}
	var digits [24]byte
	i := len(digits)
	for n > 0 {
		i--
		digits[i] = byte(n%10) + '0'
		n /= 10
	}
	return string(digits[i:])
}

func genericUnsignedDecimal(n uint64) string {
	if n <= uint64(^uint(0)>>1) {
		return genericDecimal(int(n))
	}
	var digits [24]byte
	i := len(digits)
	for n > 0 {
		i--
		digits[i] = byte(n%10) + '0'
		n /= 10
	}
	return string(digits[i:])
}

func (e *genericEnvironment) fail(scope genericTypeScope, token int, message string) {
	if e.errorText != "" {
		return
	}
	e.errorText, e.errorPkg, e.errorFile, e.errorToken = message, scope.pkg, scope.file, token
}

func (e *genericEnvironment) lookup(pkg int, name string) int {
	for i := 0; i < len(e.decls); i++ {
		if e.decls[i].pkg == pkg && e.decls[i].name == name && e.decls[i].owner == 0 {
			return i
		}
	}
	return -1
}

func (e *genericEnvironment) lookupInScope(scope genericTypeScope, name string) int {
	if own := e.lookup(scope.pkg, name); own >= 0 {
		return own
	}
	if !syntax.IdentifierExported([]byte(name), 0) {
		return -1
	}
	file := &e.graph.Packages[scope.pkg].Files[scope.file].File
	found := -1
	for _, imp := range file.Imports {
		if imp.NameTok < 0 || !tokenTextIs(file, imp.NameTok, ".") {
			continue
		}
		path, _ := syntax.StringLiteralValue(file.Src, file.Tokens[imp.PathTok])
		for p := range e.graph.Packages {
			if e.graph.Packages[p].Ref.ImportPath != path {
				continue
			}
			if d := e.lookup(p, name); d >= 0 {
				if found >= 0 {
					e.fail(scope, imp.NameTok, "ambiguous dot-imported name")
					return -1
				}
				found = d
			}
		}
	}
	return found
}

func (e *genericEnvironment) resolveDeclaration(index int) int {
	if index < 0 || index >= len(e.decls) {
		return 0
	}
	d := &e.decls[index]
	if d.state == 2 {
		return d.typ
	}
	if d.state == 1 {
		if d.alias {
			e.fail(genericTypeScope{pkg: d.pkg, file: d.file}, d.token, "recursive type alias")
		}
		return d.typ
	}
	e.decls[index].state = 1
	// unsafe.Pointer is a compiler-defined basic type. The adapter's source
	// declaration supplies lexical visibility, not its underlying identity.
	if e.types.get(d.typ).origin == "unsafe.Pointer" {
		e.decls[index].typ = e.types.basic("unsafe.Pointer")
		e.decls[index].state = 2
		return e.decls[index].typ
	}
	scope := genericTypeScope{pkg: d.pkg, file: d.file, parameters: d.parameters}
	if len(d.list.Parameters) != 0 {
		e.requireVersion(scope, d.token, "1.18", "type parameters")
		if d.kind == SymbolType && d.alias {
			e.requireVersion(scope, d.token, "1.24", "generic type aliases")
		}
	}
	if d.owner > 0 {
		e.resolveDeclaration(d.owner - 1)
		e.decls[index].constraints = append([]genericConstraint(nil), e.decls[d.owner-1].constraints...)
	}
	if d.kind == SymbolMethod {
		if len(d.parameters) != 0 {
			e.requireVersion(scope, d.token, "1.18", "generic receiver")
		}
		e.resolveDeclaration(d.receiver - 1)
		base := &e.decls[d.receiver-1]
		for _, constraint := range base.constraints {
			e.decls[index].constraints = append(e.decls[index].constraints, e.types.substituteConstraint(constraint, base.parameters, d.parameters))
		}
	}
	for _, p := range d.list.Parameters {
		c := e.parseConstraint(scope, p.ConstraintStart, p.ConstraintEnd)
		e.decls[index].constraints = append(e.decls[index].constraints, c)
	}
	for i, parameter := range d.parameters {
		if i < len(e.decls[index].constraints) {
			constraint := e.decls[index].constraints[i]
			e.types.items[parameter-1].underlying = e.types.coreType(constraint)
			e.types.items[parameter-1].methods = append([]genericMethod(nil), constraint.methods...)
		}
	}
	if d.kind == SymbolType {
		underlying := e.parseType(scope, d.start, d.end)
		if d.alias {
			e.decls[index].typ = underlying
		} else {
			if e.types.get(underlying).kind == genericParameter {
				e.fail(scope, d.start, "defined type cannot have a type parameter as its underlying type")
			}
			e.types.items[d.typ-1].underlying = underlying
		}
	} else {
		file := &e.graph.Packages[d.pkg].Files[d.file].File
		sig := buildFuncSignature(file, &d.function)
		e.decls[index].typ = e.signature(scope, sig)
	}
	e.decls[index].state = 2
	return e.decls[index].typ
}

func (e *genericEnvironment) signature(scope genericTypeScope, sig FuncSignature) int {
	v := genericType{kind: genericFunc}
	for _, f := range sig.Params {
		start := f.TypeStart
		if f.Variadic {
			start++
			v.variadic = true
		}
		id := e.parseType(scope, start, f.TypeEnd)
		if !e.valueType(id) {
			e.fail(scope, start, "constraint interface cannot be used as a value type")
		}
		if f.Variadic {
			id = e.types.intern(genericType{kind: genericSlice, elem: id})
		}
		v.params = append(v.params, id)
	}
	for _, f := range sig.Results {
		id := e.parseType(scope, f.TypeStart, f.TypeEnd)
		if !e.valueType(id) {
			e.fail(scope, f.TypeStart, "constraint interface cannot be used as a value type")
		}
		v.results = append(v.results, id)
	}
	return e.types.intern(v)
}

func (e *genericEnvironment) imported(scope genericTypeScope, name string) int {
	file := &e.graph.Packages[scope.pkg].Files[scope.file].File
	for _, imp := range file.Imports {
		path := tokenString(file, imp.PathTok)
		if len(path) < 2 {
			continue
		}
		path = path[1 : len(path)-1]
		for p := 0; p < len(e.graph.Packages); p++ {
			pkg := &e.graph.Packages[p]
			if pkg.Ref.ImportPath != path {
				continue
			}
			alias := pkg.Name
			if imp.NameTok >= 0 {
				alias = tokenString(file, imp.NameTok)
			}
			if alias == name {
				return p
			}
		}
	}
	return -1
}

func (e *genericEnvironment) parseType(scope genericTypeScope, start int, end int) int {
	file := &e.graph.Packages[scope.pkg].Files[scope.file].File
	start, end = stripOuterParens(file, start, end)
	if start < 0 || start >= end {
		e.fail(scope, start, "expected type")
		return 0
	}
	kind := file.Tokens[start].KindLine & 255
	if tokCharIs(file, start, '*') {
		return e.types.intern(genericType{kind: genericPointer, elem: e.parseType(scope, start+1, end)})
	}
	if tokCharIs(file, start, '[') {
		close := findTypeMatching(file, start, '[', ']')
		if close <= start || close >= end {
			e.fail(scope, start, "invalid array or slice type")
			return 0
		}
		elem := e.parseType(scope, close, end)
		if close == start+2 {
			return e.types.intern(genericType{kind: genericSlice, elem: elem})
		}
		length := int64(0)
		info := PackageInfo{}
		for f, source := range e.graph.Packages[scope.pkg].Files {
			for _, decl := range source.File.Decls {
				if decl.Kind == syntax.TokenConst {
					info.Decls = append(info.Decls, buildDeclInfo(&source.File, f, PackageInfo{}, nil, decl))
				}
			}
		}
		sortDecls(info.Decls)
		context := constantIndexContext{pkg: &e.graph.Packages[scope.pkg], info: &info, fileIndex: scope.file, before: start}
		var enclosing syntax.FuncDecl
		enclosing.BodyStart = -1
		for _, fn := range file.Funcs {
			if fn.BodyStart < start && start < fn.BodyEnd {
				enclosing = fn
				body := syntax.ParseFuncBodyStatements(*file, fn)
				sig := buildFuncSignature(file, &fn)
				context.bindings = collectScopedTypeBindings(file, &fn, &body, &sig)
				for token := fn.BodyStart + 1; token < start; token++ {
					if file.Tokens[token].KindLine&255 != syntax.TokenFunc {
						continue
					}
					inner := genericFunctionLiteral(file, token, fn.BodyEnd)
					if inner.BodyStart >= 0 && inner.BodyStart < start && start < inner.BodyEnd {
						enclosing = inner
						body = syntax.ParseFuncBodyStatements(*file, inner)
						sig = buildFuncSignature(file, &inner)
						context.bindings = append(context.bindings, collectScopedTypeBindings(file, &inner, &body, &sig)...)
					}
				}
				break
			}
		}
		value := wideConstantExpr(&context, start+1, close-1, 0)
		for token := start + 1; token < close-1; token++ {
			if file.Tokens[token].KindLine&255 == syntax.TokenIdent && e.localType(scope, tokenString(file, token), token) >= 0 {
				value = wideConstant{}
			}
		}
		for _, parameter := range scope.parameters {
			name := e.types.get(parameter).name
			for token := start + 1; token < close-1; token++ {
				if tokenTextIs(file, token, name) {
					value = wideConstant{}
				}
			}
		}
		if !value.ok {
			value = e.genericArrayLength(scope, start+1, close-1, enclosing, context.bindings)
		}
		if !value.ok || value.negative {
			e.fail(scope, start+1, "invalid array length")
			return 0
		}
		var fits bool
		length, fits = wideInt64(value)
		if !fits || !genericRationalFits(genericRationalInteger(value), "int", e.wordBits, e.pointerBits) {
			e.fail(scope, start+1, "array length is too large")
			return 0
		}
		e.recordArrayLength(scope, start+1, close-1, value, context.bindings)
		return e.types.intern(genericType{kind: genericArray, elem: elem, length: uint64(length)})
	}
	if kind == syntax.TokenMap {
		close := findTypeMatching(file, start+1, '[', ']')
		if close <= start+2 || close >= end {
			e.fail(scope, start, "invalid map type")
			return 0
		}
		key := e.parseType(scope, start+2, close-1)
		e.typeUses = append(e.typeUses, genericTypeUse{scope: scope, token: start + 2, key: key})
		return e.types.intern(genericType{kind: genericMap, key: key, elem: e.parseType(scope, close, end)})
	}
	if kind == syntax.TokenChan || tokenTextIs(file, start, "<-") {
		direction, next := ChanBoth, start+1
		if tokenTextIs(file, start, "<-") {
			direction, next = ChanReceiveOnly, start+2
		} else if tokenTextIs(file, next, "<-") {
			direction, next = ChanSendOnly, next+1
		}
		return e.types.intern(genericType{kind: genericChan, direction: direction, elem: e.parseType(scope, next, end)})
	}
	if kind == syntax.TokenFunc {
		close := findTypeMatching(file, start+1, '(', ')')
		if close <= start+1 {
			e.fail(scope, start, "invalid function type")
			return 0
		}
		return e.signature(scope, buildSignatureFromParts(file, -1, -1, start+1, close, close, end))
	}
	if kind == syntax.TokenStruct {
		fields := parseStructFields(file, start+2, end-1)
		v := genericType{kind: genericStruct}
		for _, f := range fields {
			finish := f.TypeEnd
			tag := ""
			if finish < end && file.Tokens[finish].KindLine&255 == syntax.TokenString {
				tag, _ = syntax.StringLiteralValue(file.Src, file.Tokens[finish])
			}
			id := e.parseType(scope, f.TypeStart, finish)
			name := f.Name
			if f.NameTok < 0 {
				base := e.types.get(id)
				if base.kind == genericPointer {
					base = e.types.get(base.elem)
				}
				if base.kind == genericParameter {
					e.fail(scope, f.TypeStart, "embedded field cannot be a type parameter")
				}
				nameToken := f.TypeStart
				if tokCharIs(file, nameToken, '*') {
					nameToken++
				}
				if tokCharIs(file, nameToken+1, '.') {
					nameToken += 2
				}
				name = tokenString(file, nameToken)
			}
			for _, previous := range v.fields {
				if name != "_" && previous.name == name {
					e.fail(scope, f.TypeStart, "duplicate struct field")
				}
			}
			if !e.valueType(id) {
				e.fail(scope, f.TypeStart, "constraint interface cannot be used as a field type")
			}
			pkg := ""
			if len(name) > 0 && !syntax.IdentifierExported([]byte(name), 0) {
				pkg = e.graph.Packages[scope.pkg].Ref.ImportPath
			}
			v.fields = append(v.fields, genericField{name: name, pkg: pkg, typ: id, tag: tag, embedded: f.NameTok < 0})
		}
		return e.types.intern(v)
	}
	if kind == syntax.TokenInterface {
		c := e.parseConstraint(scope, start, end)
		return e.types.interfaceType(c)
	}
	if kind != syntax.TokenIdent {
		e.fail(scope, start, "expected type name")
		return 0
	}
	name := tokenString(file, start)
	if e.typeNameIsValue(scope, start) {
		e.fail(scope, start, "name does not denote a type")
		return 0
	}
	if local := e.localType(scope, name, start); local >= 0 && (start+1 >= end || !tokCharIs(file, start+1, '.')) {
		if end != start+1 {
			e.fail(scope, start, "local type cannot have type arguments")
			return 0
		}
		return e.resolveDeclaration(local)
	}
	for _, id := range scope.parameters {
		if e.types.get(id).name == name && name != "_" {
			if end != start+1 {
				e.fail(scope, start, "type parameter cannot be instantiated")
				return 0
			}
			return id
		}
	}
	pkg, next := scope.pkg, start+1
	if next < end && tokCharIs(file, next, '.') {
		pkg = e.imported(scope, name)
		name = tokenString(file, next+1)
		next += 2
		if pkg < 0 || !syntax.IdentifierExported([]byte(name), 0) {
			e.fail(scope, start, "unresolved imported type")
			return 0
		}
	}
	decl := e.lookup(pkg, name)
	if pkg == scope.pkg && next == start+1 {
		decl = e.lookupInScope(scope, name)
	}
	if decl < 0 {
		if pkg == scope.pkg && next == start+1 && (e.packageValueName(pkg, name) || e.imported(scope, name) >= 0) {
			e.fail(scope, start, "name does not denote a type")
			return 0
		}
		if pkg == scope.pkg && next == end {
			if name == "comparable" {
				e.requireVersion(scope, start, "1.18", "predeclared comparable")
				return e.types.interfaceType(genericConstraint{all: true, comparable: true})
			}
			if name == "any" {
				e.requireVersion(scope, start, "1.18", "predeclared any")
				return e.types.intern(genericType{kind: genericInterface})
			}
			if name == "error" {
				return e.types.errorType()
			}
			if genericBasicName(name) {
				return e.types.basic(name)
			}
		}
		e.fail(scope, start, "undefined type "+name)
		return 0
	}
	if e.decls[decl].kind != SymbolType {
		e.fail(scope, start, "name does not denote a type")
		return 0
	}
	id := e.resolveDeclaration(decl)
	d := &e.decls[decl]
	if len(d.parameters) == 0 {
		if next != end {
			e.fail(scope, next, "cannot instantiate a non-generic type")
			return 0
		}
		return id
	}
	if !tokCharIs(file, next, '[') || findTypeMatching(file, next, '[', ']') != end {
		e.fail(scope, start, "generic type requires type arguments")
		return 0
	}
	e.requireVersion(scope, next, "1.18", "type instantiation")
	var args []int
	for i := next + 1; i < end-1; {
		last := nextTopLevelComma(file, i, end-1)
		args = append(args, e.parseType(scope, i, last))
		i = last + 1
	}
	if len(args) != len(d.parameters) {
		e.fail(scope, start, "wrong number of type arguments")
		return 0
	}
	e.recordInstantiation(scope, start, d.parameters, args)
	e.typeUses = append(e.typeUses, genericTypeUse{scope: scope, token: start, declaration: decl, arguments: append([]int(nil), args...)})
	return e.types.substitute(id, d.parameters, args)
}

func genericBasicName(name string) bool {
	return name == "bool" || name == "string" || name == "int" || name == "uint" || name == "uintptr" || name == "byte" || name == "rune" || name == "int8" || name == "int16" || name == "int32" || name == "int64" || name == "uint8" || name == "uint16" || name == "uint32" || name == "uint64" || name == "float32" || name == "float64" || name == "complex64" || name == "complex128"
}

func (e *genericEnvironment) parseConstraint(scope genericTypeScope, start int, end int) genericConstraint {
	file := &e.graph.Packages[scope.pkg].Files[scope.file].File
	start, end = stripOuterParens(file, start, end)
	if start >= end {
		e.fail(scope, start, "expected constraint")
		return genericConstraint{}
	}
	if file.Tokens[start].KindLine&255 == syntax.TokenInterface {
		if !tokCharIs(file, start+1, '{') || findTypeMatching(file, start+1, '{', '}') != end {
			e.fail(scope, start, "invalid constraint interface")
			return genericConstraint{}
		}
		methods, embeds := parseInterfaceElements(file, start+2, end-1)
		c := genericConstraint{all: true}
		for _, m := range methods {
			pkg := ""
			if !syntax.IdentifierExported([]byte(m.Name), 0) {
				pkg = e.graph.Packages[scope.pkg].Ref.ImportPath
			}
			method := genericMethod{name: m.Name, pkg: pkg, typ: e.signature(scope, m.Signature)}
			for _, old := range c.methods {
				if genericSameMethodName(old, method) {
					e.fail(scope, m.NameTok, "duplicate constraint method")
				}
			}
			c.methods = append(c.methods, method)
		}
		c.methods, _ = genericMergeMethods(nil, c.methods)
		for _, embedded := range embeds {
			other := e.parseConstraint(scope, embedded.TypeStart, embedded.TypeEnd)
			var ok bool
			c, ok = e.types.intersect(c, other)
			if !ok {
				e.fail(scope, embedded.TypeStart, "conflicting constraint methods")
			}
		}
		return c
	}
	var terms []genericTerm
	var result genericConstraint
	for i := start; i < end; {
		last := genericUnionEnd(file, i, end)
		if last < end {
			e.requireVersion(scope, i, "1.18", "interface type unions")
		}
		tilde := tokCharIs(file, i, '~')
		next := i
		if tilde {
			e.requireVersion(scope, i, "1.18", "interface type terms")
			next++
		}
		id := e.parseType(scope, next, last)
		u := e.types.get(e.types.underlying(id))
		if u.kind == genericInterface {
			if tilde {
				e.fail(scope, i, "invalid approximation of an interface")
			}
			c := e.types.interfaceConstraint(id)
			if i == start && last == end {
				return c
			}
			if c.comparable || len(c.methods) != 0 {
				e.fail(scope, i, "invalid interface in constraint union")
				return genericConstraint{}
			}
			result = e.types.unionSets(result, c)
		} else {
			e.requireVersion(scope, i, "1.18", "interface type terms")
			term := genericTerm{typ: id, tilde: tilde}
			terms = append(terms, term)
			result = e.types.unionSets(result, genericConstraint{terms: []genericTerm{term}})
		}
		i = last + 1
	}
	_, ok := e.types.union(terms)
	if !ok {
		e.fail(scope, start, "invalid or overlapping constraint terms")
	}
	return result
}

func genericUnionEnd(file *syntax.File, start int, end int) int {
	for i := start; i < end; i++ {
		if tokCharIs(file, i, '|') {
			return i
		}
		if tokCharIs(file, i, '[') || tokCharIs(file, i, '(') || tokCharIs(file, i, '{') {
			open := byte(file.Tokens[i].KindLine >> syntax.TokenOperatorCharShift & syntax.TokenOperatorCharMask)
			close := byte(')')
			if open == '[' {
				close = ']'
			}
			if open == '{' {
				close = '}'
			}
			next := findTypeMatching(file, i, open, close)
			if next <= i {
				return end
			}
			i = next - 1
		}
	}
	return end
}
