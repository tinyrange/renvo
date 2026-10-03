package check

import "renvo.dev/internal/syntax"

type genericExpressionContext struct {
	specializer                *genericSpecializer
	scope                      genericTypeScope
	arguments                  []int
	bindings                   []scopedTypeBinding
	ranges                     []genericRangeBinding
	tuples                     []genericTupleBinding
	function                   syntax.FuncDecl
	receiverName, receiverType int
	depth                      int
	nonconstantCalls           int
	specialize                 bool
	changes                    []genericReplacement
}

type genericRangeBinding struct {
	name, start, end, expressionStart, expressionEnd, index int
}

type genericTupleBinding struct{ name, start, end, index, count int }

func (c *genericExpressionContext) expressionValues(spans []ExprSpan, before int) []genericArgument {
	var values []genericArgument
	for _, span := range spans {
		value := c.expression(span.StartTok, span.EndTok, before)
		if len(spans) == 1 && len(value.results) > 0 {
			for _, typ := range value.results {
				values = append(values, genericArgument{typ: typ})
			}
		} else {
			values = append(values, value)
		}
	}
	return values
}

func (c *genericExpressionContext) typeSpan(start int, end int) int {
	e := c.specializer.environment
	file := &e.graph.Packages[c.scope.pkg].Files[c.scope.file].File
	// A type-only scan can include the parentheses of a one-argument call,
	// as in make(map[K]V). Replace the type itself and retain its delimiters.
	start, end = stripOuterParens(file, start, end)
	scope := c.scope
	scope.bindings = c.bindings
	id := e.parseType(scope, start, end)
	if len(c.arguments) != 0 {
		original := id
		id = e.types.substitute(id, c.scope.parameters, c.arguments)
		// Map lowering emits package-level helpers. Their signatures cannot
		// refer to the specialization's local parameter aliases.
		if c.specialize && id != original && e.types.get(id).kind == genericMap {
			c.replace(start, end, c.scopedTypeText(id, start))
		}
	}
	return id
}

func (c *genericExpressionContext) binding(token int, before int) int {
	e := c.specializer.environment
	file := &e.graph.Packages[c.scope.pkg].Files[c.scope.file].File
	chosen := -1
	for i, b := range c.bindings {
		if b.name >= 0 && b.visible <= before && before < b.end && coreTokensEqual(file, b.name, token) && (chosen < 0 || b.visible > c.bindings[chosen].visible) {
			chosen = i
		}
	}
	if chosen >= 0 {
		if local := e.localType(c.scope, tokenString(file, token), before); local >= 0 && e.decls[local].token > c.bindings[chosen].name {
			return -1
		}
	}
	return chosen
}

func (c *genericExpressionContext) declaration(start int, end int) (int, int) {
	e := c.specializer.environment
	file := &e.graph.Packages[c.scope.pkg].Files[c.scope.file].File
	if start >= end || file.Tokens[start].KindLine&255 != syntax.TokenIdent || c.binding(start, start) >= 0 {
		return -1, start
	}
	name := tokenString(file, start)
	pkg, next := c.scope.pkg, start+1
	if local := e.localType(c.scope, name, start); local >= 0 && (next >= end || !tokCharIs(file, next, '.')) {
		return local, next
	}
	for _, p := range c.scope.parameters {
		if e.types.get(p).name == name {
			return -1, next
		}
	}
	if next < end && tokCharIs(file, next, '.') {
		pkg = e.imported(c.scope, name)
		if pkg < 0 {
			return -1, start
		}
		name = tokenString(file, next+1)
		if !syntax.IdentifierExported([]byte(name), 0) {
			e.fail(c.scope, next+1, "cannot access unexported imported declaration")
			return -1, start
		}
		next += 2
	}
	if pkg == c.scope.pkg && next == start+1 {
		return e.lookupInScope(c.scope, name), next
	}
	return e.lookup(pkg, name), next
}

func (c *genericExpressionContext) expression(start int, end int, before int) genericArgument {
	callsBefore := c.nonconstantCalls
	c.depth++
	value := c.expressionInner(start, end, before)
	// Looking up a variable may inspect its initializer for type inference.
	// Those calls are outside this expression's syntax and do not prevent a
	// surrounding array len/cap from being constant.
	file := &c.specializer.environment.graph.Packages[c.scope.pkg].Files[c.scope.file].File
	strippedStart, strippedEnd := stripOuterParens(file, start, end)
	if strippedEnd == strippedStart+1 {
		c.nonconstantCalls = callsBefore
	}
	if value.typ != 0 && value.constant != nil && !c.specializer.environment.constantFits(value, value.typ) {
		c.specializer.environment.fail(c.scope, start, "constant overflows its type")
	}
	value.constant = c.specializer.environment.roundConstant(value.constant, value.typ)
	if value.constant != nil {
		value.start, value.end = start, end
	}
	if c.specialize && len(c.arguments) != 0 && value.constant != nil && c.depth == 1 {
		c.lowerIntegerConstant(start, end, value)
	}
	c.depth--
	return value
}

func (c *genericExpressionContext) expressionInner(start int, end int, before int) genericArgument {
	e := c.specializer.environment
	file := &e.graph.Packages[c.scope.pkg].Files[c.scope.file].File
	start, end = stripOuterParens(file, start, end)
	if start < 0 || start >= end || c.depth > len(file.Tokens) {
		return genericArgument{}
	}
	// A literal's result type may contain pointer stars or nested function
	// signatures. They are type syntax, not binary operators in this expression.
	if file.Tokens[start].KindLine&255 == syntax.TokenFunc && tokCharIs(file, end-1, '}') {
		literal := genericFunctionLiteral(file, start, end)
		if literal.BodyStart >= 0 && literal.BodyEnd == end {
			return genericArgument{typ: c.typeSpan(start, literal.BodyStart)}
		}
	}
	for precedence := 1; precedence <= 5; precedence++ {
		operator := genericBinaryOperator(file, start, end, precedence)
		if operator < 0 {
			continue
		}
		left := c.expression(start, operator, before)
		right := c.expression(operator+1, end, before)
		op := tokenString(file, operator)
		result, ok := e.binaryOperation(left, right, op)
		if !ok {
			message := "operation is not valid for every type in the constraint"
			if len(c.scope.parameters) == 0 {
				message = "invalid operation for operand types"
			}
			e.fail(c.scope, operator, message)
		}
		if op != "<<" && op != ">>" {
			if left.untyped != 0 && right.typ != 0 {
				c.lowerExpectedConstant(left, right.typ)
			}
			if right.untyped != 0 && left.typ != 0 {
				c.lowerExpectedConstant(right, left.typ)
			}
		}
		if c.specialize && len(c.arguments) != 0 && (op == "<<" || op == ">>") {
			// A valid shift gives integral untyped operands an integer context.
			// Emit that context before the backend classifies numeric literals.
			if left.untyped != 0 && left.constant != nil {
				left.untyped = genericUntypedInt
				c.lowerIntegerConstant(start, operator, left)
			}
			if right.untyped != 0 && right.constant != nil {
				right.untyped = genericUntypedInt
				c.lowerIntegerConstant(operator+1, end, right)
			}
		}
		return result
	}
	if end == start+1 {
		token := file.Tokens[start]
		switch token.KindLine & 255 {
		case syntax.TokenString:
			text, _ := syntax.StringLiteralValue(file.Src, token)
			return genericArgument{untyped: genericUntypedString, constant: &genericConstant{text: &text}}
		case syntax.TokenChar:
			value, ok := syntax.RuneLiteralValue(file.Src, token)
			if ok {
				return genericArgument{untyped: genericUntypedRune, constant: &genericConstant{real: genericRationalInteger(wideSmall(value)), imaginary: genericRationalInteger(wideSmall(0))}}
			}
			return genericArgument{untyped: genericUntypedRune}
		case syntax.TokenNumber:
			text := tokenString(file, start)
			kind := genericUntypedInt
			hex := len(text) > 2 && text[0] == '0' && (text[1] == 'x' || text[1] == 'X')
			for i := 0; i < len(text); i++ {
				if text[i] == '.' || text[i] == 'p' || text[i] == 'P' || !hex && (text[i] == 'e' || text[i] == 'E') {
					kind = genericUntypedFloat
				}
			}
			if len(text) > 0 && text[len(text)-1] == 'i' {
				kind = genericUntypedComplex
			}
			constant := genericConstantLiteral(text)
			if constant == nil {
				e.fail(c.scope, start, "numeric constant exceeds supported precision")
			}
			return genericArgument{untyped: kind, constant: constant}
		}
		if binding := c.binding(start, before); binding >= 0 {
			b := c.bindings[binding]
			if c.receiverType != 0 && b.name == c.receiverName {
				return genericArgument{typ: c.receiverType}
			}
			for _, tuple := range c.tuples {
				if tuple.name == b.name {
					value := c.expression(tuple.start, tuple.end, b.name)
					if tuple.index < len(value.results) {
						return genericArgument{typ: value.results[tuple.index]}
					}
					if value.commaOK && tuple.index == 1 {
						return genericArgument{typ: e.types.basic("bool")}
					}
					if value.commaOK && tuple.index == 0 {
						return genericArgument{typ: value.typ}
					}
				}
			}
			for _, r := range c.ranges {
				if r.name != b.name || before < r.start || before >= r.end {
					continue
				}
				value := c.expression(r.expressionStart, r.expressionEnd, r.expressionStart)
				types, ok := e.rangeTypes(value)
				if ok && r.index < len(types) {
					return genericArgument{typ: types[r.index]}
				}

			}
			if b.typeStart >= 0 && b.typeEnd > b.typeStart {
				if tokenTextIs(file, b.typeStart, "...") {
					return genericArgument{typ: e.types.intern(genericType{kind: genericSlice, elem: c.typeSpan(b.typeStart+1, b.typeEnd)})}
				}
				value := genericArgument{typ: c.typeSpan(b.typeStart, b.typeEnd)}
				if b.constant && b.valueStart >= 0 {
					value.constant = c.expression(b.valueStart, b.valueEnd, b.name).constant
				}
				return value
			}
			if b.valueStart >= 0 {
				value := c.expression(b.valueStart, b.valueEnd, b.name)
				if !b.constant && value.untyped != 0 {
					value.typ = e.types.defaultType(value.untyped)
					value.untyped = 0
				}
				if !b.constant {
					value.constant = nil
					value.shifted = nil
				}
				return value
			}
		}
		name := tokenString(file, start)
		for _, parameter := range c.scope.parameters {
			if e.types.get(parameter).name == name {
				e.fail(c.scope, start, "type parameter used as a value")
				return genericArgument{}
			}
		}
		if e.localType(c.scope, name, start) >= 0 {
			e.fail(c.scope, start, "type used as a value")
			return genericArgument{}
		}
		if d := e.lookupInScope(c.scope, name); d >= 0 && e.decls[d].kind == SymbolType {
			e.fail(c.scope, start, "type used as a value")
			return genericArgument{}
		}
		if d := e.lookupInScope(c.scope, name); d >= 0 && e.decls[d].kind == SymbolFunc {
			if len(e.decls[d].parameters) > 0 {
				return genericArgument{typ: e.resolveDeclaration(d), function: d + 1, start: start, end: end}
			}
			return genericArgument{typ: e.resolveDeclaration(d)}
		}
		if value := c.globalValue(c.scope.pkg, name); value.typ != 0 || value.untyped != 0 {
			return value
		}
		if pkg := e.dotImportedValuePackage(c.scope, name); pkg >= 0 {
			return c.globalValue(pkg, name)
		}
		if name == "iota" {
			for _, binding := range c.bindings {
				if binding.constant && binding.name == before {
					return genericArgument{untyped: genericUntypedInt, constant: genericConstantLiteral(genericDecimal(binding.iotaValue))}
				}
			}
		}
		if name == "true" || name == "false" {
			return genericArgument{untyped: genericUntypedBool, constant: genericBooleanConstant(name == "true")}
		}
		if name == "nil" {
			return genericArgument{untyped: genericUntypedNil}
		}
		return genericArgument{}
	}
	if tokCharIs(file, start, '&') {
		value := c.expression(start+1, end, before)
		operand, operandEnd := stripOuterParens(file, start+1, end)
		literal := tokCharIs(file, operandEnd-1, '}') && file.Tokens[operand].KindLine&255 != syntax.TokenFunc
		if !c.addressable(start+1, end, before) && !literal {
			e.fail(c.scope, start, "address operand is not addressable")
		}
		if value.typ != 0 {
			return genericArgument{typ: e.types.intern(genericType{kind: genericPointer, elem: value.typ})}
		}
	}
	if tokCharIs(file, start, '*') {
		value := c.expression(start+1, end, before)
		pointer := e.types.get(e.coreType(value.typ))
		if pointer.kind != genericPointer {
			e.fail(c.scope, start, "dereference requires a pointer constraint")
		}
		return genericArgument{typ: pointer.elem}
	}
	if tokenTextIs(file, start, "<-") {
		c.nonconstantCalls++
		value := c.expression(start+1, end, before)
		channel := e.types.get(e.coreType(value.typ))
		if channel.kind != genericChan || channel.direction == ChanSendOnly {
			e.fail(c.scope, start, "receive requires a receive-capable channel constraint")
		}
		return genericArgument{typ: channel.elem, commaOK: true}
	}
	if tokCharIs(file, start, '+') || tokCharIs(file, start, '-') || tokCharIs(file, start, '!') || tokCharIs(file, start, '^') {
		value := c.expression(start+1, end, before)
		if value.constant != nil {
			copy := *value.constant
			if tokCharIs(file, start, '!') && copy.boolean != nil {
				copy.boolean = genericBooleanConstant(!*copy.boolean).boolean
			}
			if tokCharIs(file, start, '-') {
				copy.real = genericRationalNegate(copy.real)
				copy.imaginary = genericRationalNegate(copy.imaginary)
			}
			if tokCharIs(file, start, '^') {
				copy.real = genericRationalAdd(genericRationalNegate(copy.real), genericRationalInteger(wideSmall(-1)))
				name := e.types.get(e.types.underlying(value.typ)).name
				if len(name) > 0 && name[0] == 'u' && genericInteger(name) {
					bits := genericIntegerBits(name, e.wordBits, e.pointerBits)
					copy.real = genericRationalAdd(copy.real, genericRationalInteger(wideShift(wideSmall(1), bits, true)))
				}
			}
			value.constant = &copy
		}
		op := tokenString(file, start)
		if op == "+" || op == "-" {
			op = "unary" + op
		}
		typ := value.typ
		if typ == 0 {
			typ = e.types.defaultType(value.untyped)
		}
		if !e.allowsOperation(typ, op) {
			e.fail(c.scope, start, "unary operation is not valid for every type in the constraint")
		}
		return value
	}
	if tokCharIs(file, end-1, ')') {
		open := genericMatchingOpen(file, start, end-1, '(', ')')
		if open > start && tokCharIs(file, open-1, '.') {
			base := c.expression(start, open-1, before)
			if e.types.get(base.typ).kind == genericParameter || e.types.get(e.coreType(base.typ)).kind != genericInterface {
				e.fail(c.scope, start, "type assertion requires an interface value")
			}
			return genericArgument{typ: c.typeSpan(open+1, end-1), commaOK: true}
		}
		if open > start {
			return c.call(start, open, end, before)
		}
	}
	if tokCharIs(file, end-1, '}') {
		open := genericMatchingOpen(file, start, end-1, '{', '}')
		if open > start {
			if file.Tokens[start].KindLine&255 == syntax.TokenFunc {
				return genericArgument{typ: c.typeSpan(start, open)}
			}
			if tokCharIs(file, start, '[') && tokenTextIs(file, start+1, "...") && tokCharIs(file, start+2, ']') {
				typ := e.types.intern(genericType{kind: genericArray, elem: c.typeSpan(start+3, open), length: ^uint64(0)})
				return c.composite(typ, open, end, before)
			}
			return c.composite(c.typeSpan(start, open), open, end, before)
		}
	}
	if tokCharIs(file, end-1, ']') {
		if d, next := c.declaration(start, end); d >= 0 && e.decls[d].kind == SymbolFunc && tokCharIs(file, next, '[') && findTypeMatching(file, next, '[', ']') == end {
			if len(splitExprList(file, next+1, end-1)) == len(e.decls[d].parameters) {
				return c.genericCall(d, start, next, end, nil, false, false)
			}
			return genericArgument{typ: e.resolveDeclaration(d), function: d + 1, start: start, end: end}
		}
		open := genericMatchingOpen(file, start, end-1, '[', ']')
		if open > start {
			base := c.expression(start, open, before)
			if colon := findTypeTopLevelChar(file, open+1, end-1, ':'); colon >= 0 {
				return c.sliceExpression(base, start, open, colon, end, before)
			}
			// Indexing an untyped string yields a byte value, just as
			// indexing a typed string does; the result is not a constant.
			if base.untyped == genericUntypedString {
				base.typ = e.types.basic("string")
			}
			key, element, ok := e.indexType(base.typ)
			if ok {
				if e.types.get(e.coreType(base.typ)).kind == genericMap {
					value := c.expression(open+1, end-1, before)
					if !e.argumentAssignable(value, key) {
						e.fail(c.scope, open+1, "index does not match map key constraint")
					}
					c.lowerExpectedConstant(value, key)
				} else {
					index := c.validateIndex(open+1, end-1, before)
					limit := e.arrayIndexLimit(base.typ)
					if index >= 0 && (uint64(index) >= limit || base.constant != nil && base.constant.text != nil && uint64(index) >= uint64(len(*base.constant.text))) {
						e.fail(c.scope, open+1, "index exceeds length")
					}
				}
				return genericArgument{typ: element, commaOK: e.types.get(e.coreType(base.typ)).kind == genericMap}
			}
			if e.types.get(base.typ).kind == genericParameter {
				e.fail(c.scope, start, "type parameter cannot be indexed")
			}
		}
	}
	if end >= start+3 && tokCharIs(file, end-2, '.') {
		name := tokenString(file, end-1)
		if end == start+3 {
			if pkg := e.imported(c.scope, tokenString(file, start)); pkg >= 0 {
				if d := e.lookup(pkg, name); d >= 0 && e.decls[d].kind == SymbolFunc {
					value := genericArgument{typ: e.resolveDeclaration(d)}
					if len(e.decls[d].parameters) > 0 {
						value.function, value.start, value.end = d+1, start, end
					}
					return value
				}
				return c.globalValue(pkg, name)
			}
		}
		baseType := c.expressionType(start, end-2)
		base := genericArgument{typ: baseType}
		if baseType == 0 {
			base = c.expression(start, end-2, before)
		}
		e.attachMethods(base.typ)
		methodType := base.typ
		if baseType == 0 && e.types.get(base.typ).kind == genericNamed && e.types.get(e.coreType(base.typ)).kind != genericInterface && c.addressable(start, end-2, before) {
			methodType = e.types.intern(genericType{kind: genericPointer, elem: base.typ})
		}
		for _, method := range e.types.methodSet(methodType) {
			if method.name == name {
				if method.pkg != "" && method.pkg != e.graph.Packages[c.scope.pkg].Ref.ImportPath {
					e.fail(c.scope, end-1, "cannot access unexported method")
					return genericArgument{}
				}
				c.lowerGenericMethodSelection(start, end, baseType, method.typ)
				if baseType != 0 {
					fn := *e.types.get(method.typ)
					fn.params = append([]int{baseType}, fn.params...)
					return genericArgument{typ: e.types.intern(fn)}
				}
				return genericArgument{typ: method.typ}
			}
		}
		if e.types.get(base.typ).kind != genericParameter {
			for _, f := range e.types.promotedMembers(base.typ, true) {
				if f.name == name {
					if f.pkg != "" && f.pkg != e.graph.Packages[c.scope.pkg].Ref.ImportPath {
						e.fail(c.scope, end-1, "cannot access unexported field")
						return genericArgument{}
					}
					return genericArgument{typ: f.typ}
				}
			}
		}
		if e.types.get(base.typ).kind == genericParameter {
			constraint, ok := e.parameterConstraint(base.typ)
			if ok {
				for _, method := range constraint.methods {
					if method.name == name {
						return genericArgument{typ: method.typ}
					}
				}
			}
			e.fail(c.scope, end-1, "type parameter has no such method")
		}
	}
	return genericArgument{}
}

func (c *genericExpressionContext) globalValue(pkg int, name string) genericArgument {
	e := c.specializer.environment
	for f := 0; f < len(e.graph.Packages[pkg].Files); f++ {
		file := &e.graph.Packages[pkg].Files[f].File
		for _, d := range file.Decls {
			if tokenString(file, d.NameTok) != name || d.Kind == syntax.TokenType {
				continue
			}
			info := buildDeclInfo(file, f, PackageInfo{}, nil, d)
			nested := genericExpressionContext{specializer: c.specializer, scope: genericTypeScope{pkg: pkg, file: f}, depth: c.depth}
			if d.Kind == syntax.TokenConst {
				var ordinal int
				var ok bool
				info, ordinal, ok = constantDeclarationInfo(file, info)
				if !ok {
					return genericArgument{}
				}
				nested.bindings = []scopedTypeBinding{{name: d.NameTok, constant: true, iotaValue: ordinal}}
			}
			if info.TypeStart >= 0 && info.TypeEnd > info.TypeStart {
				value := genericArgument{typ: nested.typeSpan(info.TypeStart, info.TypeEnd)}
				if d.Kind == syntax.TokenConst && info.ValueIndex < len(info.Values) {
					span := info.Values[info.ValueIndex]
					value.constant = nested.expression(span.StartTok, span.EndTok, d.NameTok).constant
				}
				return value
			}
			if info.ValueIndex < len(info.Values) {
				span := info.Values[info.ValueIndex]
				value := nested.expression(span.StartTok, span.EndTok, d.NameTok)
				if d.Kind == syntax.TokenVar && value.untyped != 0 {
					value.typ = e.types.defaultType(value.untyped)
					value.untyped = 0
				}
				if d.Kind == syntax.TokenVar {
					value.constant = nil
					value.shifted = nil
				}
				return value
			}
		}
	}
	return genericArgument{}
}

func (c *genericExpressionContext) call(start int, open int, end int, before int) genericArgument {
	e := c.specializer.environment
	file := &e.graph.Packages[c.scope.pkg].Files[c.scope.file].File
	spans := splitExprList(file, open+1, end-1)
	calleeStart, calleeEnd := stripOuterParens(file, start, open)
	if calleeStart > start {
		d, _ := c.declaration(calleeStart, calleeEnd)
		kind := file.Tokens[calleeStart].KindLine & 255
		functionType := kind == syntax.TokenFunc && genericFunctionLiteral(file, calleeStart, calleeEnd).BodyStart < 0
		containerType := kind == syntax.TokenMap || kind == syntax.TokenChan || kind == syntax.TokenStruct || kind == syntax.TokenInterface || tokenTextIs(file, calleeStart, "<-") && calleeStart+1 < calleeEnd && file.Tokens[calleeStart+1].KindLine&255 == syntax.TokenChan
		if functionType || containerType || tokCharIs(file, calleeStart, '*') || tokCharIs(file, calleeStart, '[') || d >= 0 && e.decls[d].kind == SymbolType {
			return c.conversion(start, spans, c.typeSpan(calleeStart, calleeEnd), before)
		}
	}
	if open == start+3 && tokCharIs(file, start+1, '.') && c.binding(start, before) < 0 {
		pkg := e.imported(c.scope, tokenString(file, start))
		name := tokenString(file, start+2)
		if pkg >= 0 && e.graph.Packages[pkg].Ref.ImportPath == "unsafe" && (name == "Sizeof" || name == "Alignof" || name == "Offsetof") {
			return c.unsafeLayout(start, name, spans, before)
		}
	}
	d, next := c.declaration(start, open)
	if d >= 0 && next < open && (!tokCharIs(file, next, '[') || findTypeMatching(file, next, '[', ']') != open) {
		d = -1
	}
	if d >= 0 && e.decls[d].kind == SymbolType {
		return c.conversion(start, spans, c.typeSpan(start, open), before)
	}
	if open == start+1 && d < 0 && c.binding(start, before) < 0 {
		name := tokenString(file, start)
		for _, p := range c.scope.parameters {
			if e.types.get(p).name == name {
				return c.conversion(start, spans, c.typeSpan(start, open), before)
			}
		}
		if !e.packageValueName(c.scope.pkg, name) && (genericBasicName(name) || name == "any" || name == "error") {
			return c.conversion(start, spans, c.typeSpan(start, open), before)
		}
		if value, ok := c.builtinIfVisible(start, name, spans, before); ok {
			if value.constant == nil {
				c.nonconstantCalls++
			}
			return value
		}
	}

	if tokCharIs(file, start, '[') || file.Tokens[start].KindLine&255 == syntax.TokenMap || file.Tokens[start].KindLine&255 == syntax.TokenChan || file.Tokens[start].KindLine&255 == syntax.TokenStruct || file.Tokens[start].KindLine&255 == syntax.TokenInterface || file.Tokens[start].KindLine&255 == syntax.TokenFunc && genericFunctionLiteral(file, start, open).BodyStart < 0 {
		return c.conversion(start, spans, c.typeSpan(start, open), before)
	}
	c.nonconstantCalls++
	var actual []genericArgument
	spread := false
	for i, span := range spans {
		if i == len(spans)-1 && tokenTextIs(file, span.EndTok-1, "...") {
			span.EndTok--
			spread = true
		}
		value := c.expression(span.StartTok, span.EndTok, before)
		if len(spans) == 1 && len(value.results) > 0 && !spread {
			for _, typ := range value.results {
				actual = append(actual, genericArgument{typ: typ})
			}
		} else {
			actual = append(actual, value)
		}
	}
	if d >= 0 {
		if len(e.decls[d].parameters) != 0 {
			return c.genericCall(d, start, next, open, actual, spread, true)
		}
		fn := e.types.get(e.resolveDeclaration(d))
		c.functionArguments(fn, actual, spread, start)
		if len(fn.results) == 1 {
			return genericArgument{typ: fn.results[0]}
		}
		return genericArgument{results: fn.results}
	}
	callee := c.expression(start, open, before)
	fn := e.types.get(e.coreType(callee.typ))
	if fn.kind == genericFunc {
		c.functionArguments(fn, actual, spread, start)
	}
	if fn.kind == genericFunc && len(fn.results) == 1 {
		return genericArgument{typ: fn.results[0]}
	}
	if fn.kind == genericFunc {
		return genericArgument{results: fn.results}
	}
	return genericArgument{}
}

func (c *genericExpressionContext) genericCall(declaration int, start int, next int, end int, actual []genericArgument, spread bool, called bool) genericArgument {
	e := c.specializer.environment
	if !e.requireVersion(c.scope, start, "1.18", "function instantiation") {
		return genericArgument{}
	}
	file := &e.graph.Packages[c.scope.pkg].Files[c.scope.file].File
	e.resolveDeclaration(declaration)
	d := &e.decls[declaration]
	var explicit []int
	if next < end {
		if !tokCharIs(file, next, '[') || findTypeMatching(file, next, '[', ']') != end {
			return genericArgument{}
		}
		for i := next + 1; i < end-1; {
			last := nextTopLevelComma(file, i, end-1)
			explicit = append(explicit, c.typeSpan(i, last))
			i = last + 1
		}
	}
	arguments := explicit
	for _, argument := range actual {
		e.attachMethods(argument.typ)
	}
	ok := len(explicit) == len(d.parameters)
	if called && !ok {
		arguments, ok = c.inferCall(d, explicit, actual, spread)
	}
	if !ok {
		e.fail(c.scope, start, "cannot infer generic type arguments")
		return genericArgument{}
	}
	for i, arg := range arguments {
		constraint := e.types.substituteConstraint(d.constraints[i], d.parameters, arguments)
		if !e.argumentSatisfies(c.scope, arg, constraint) {
			e.fail(c.scope, start, "type argument does not satisfy constraint")
			return genericArgument{}
		}
	}
	if len(c.arguments) == 0 {
		e.recordInstantiation(c.scope, start, d.parameters, arguments)
	}
	fn := e.types.substitute(d.typ, d.parameters, arguments)
	if called {
		signature := e.types.get(fn)
		if !signature.variadic && (spread || len(actual) != len(signature.params)) || signature.variadic && (len(actual) < len(signature.params)-1 || spread && len(actual) != len(signature.params)) {
			e.fail(c.scope, start, "wrong number of generic call arguments")
			return genericArgument{}
		}
		for i, arg := range actual {
			parameter := genericArgumentParameter(&e.types, signature, i, spread)
			if arg.function != 0 {
				arg = c.functionValue(arg.function-1, arg.start, arg.end, parameter)
			}
			if !e.argumentAssignable(arg, parameter) {
				e.fail(c.scope, start, "argument is not assignable to instantiated parameter")
				return genericArgument{}
			}
			c.lowerExpectedConstant(arg, parameter)
		}
	}
	if c.specialize {
		instance := c.specializer.instantiate(declaration, arguments)
		if instance < 0 {
			e.fail(c.scope, start, "invalid generic instantiation")
			return genericArgument{}
		}
		name := c.specializer.qualify(c.scope.pkg, d.pkg, c.specializer.instances[instance].name)
		if d.pkg != c.scope.pkg && tokCharIs(file, start+1, '.') {
			name = tokenString(file, start) + "." + c.specializer.instances[instance].name
		}
		c.replace(start, end, name)
	}
	if called {
		signature := e.types.get(fn)
		if len(signature.results) == 1 {
			return genericArgument{typ: signature.results[0]}
		}
		return genericArgument{results: signature.results}
	}
	return genericArgument{typ: fn}
}

func genericMatchingOpen(file *syntax.File, start int, end int, open byte, close byte) int {
	depth := 1
	for i := end - 1; i >= start; i-- {
		if tokCharIs(file, i, close) {
			depth++
		}
		if tokCharIs(file, i, open) {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func genericBinaryOperator(file *syntax.File, start int, end int, precedence int) int {
	for i := end - 1; i > start; i-- {
		if tokCharIs(file, i, ')') || tokCharIs(file, i, ']') || tokCharIs(file, i, '}') {
			close := byte(file.Tokens[i].KindLine >> syntax.TokenOperatorCharShift & syntax.TokenOperatorCharMask)
			open := byte('(')
			if close == ']' {
				open = '['
			}
			if close == '}' {
				open = '{'
			}
			match := genericMatchingOpen(file, start, i, open, close)
			if match < 0 {
				return -1
			}
			i = match
			continue
		}
		op := tokenString(file, i)
		p := 0
		switch op {
		case "||":
			p = 1
		case "&&":
			p = 2
		case "==", "!=", "<", "<=", ">", ">=":
			p = 3
		case "+", "-", "|", "^":
			p = 4
		case "*", "/", "%", "<<", ">>", "&", "&^":
			p = 5
		}
		if p != precedence {
			continue
		}
		previous := file.Tokens[i-1].KindLine & 255
		if previous == syntax.TokenOperator && !tokCharIs(file, i-1, ')') && !tokCharIs(file, i-1, ']') && !tokCharIs(file, i-1, '}') {
			continue
		}
		return i
	}
	return -1
}
