package check

import "renvo.dev/internal/syntax"

func genericFunctionLiteral(file *syntax.File, start int, end int) syntax.FuncDecl {
	fn := syntax.FuncDecl{NameTok: -1, StartTok: start, ReceiverStart: -1, ReceiverEnd: -1, BodyStart: -1}
	previous := start - 1
	for previous >= 0 && tokCharIs(file, previous, '*') {
		previous--
	}
	if previous >= 0 && (tokCharIs(file, previous, ']') || file.Tokens[previous].KindLine&255 == syntax.TokenChan) {
		return fn
	}
	if !tokCharIs(file, start+1, '(') {
		return fn
	}
	fn.ParamsStart = start + 1
	fn.ParamsEnd = findTypeMatching(file, start+1, '(', ')')
	if fn.ParamsEnd <= fn.ParamsStart {
		return fn
	}
	fn.ResultStart = fn.ParamsEnd
	for i := fn.ParamsEnd; i < end; i++ {
		if tokCharIs(file, i, ';') || tokCharIs(file, i, ',') || tokCharIs(file, i, ')') || tokCharIs(file, i, ']') || tokCharIs(file, i, '}') || tokCharIs(file, i, '=') || syntax.TokenLine(file.Tokens[i]) > syntax.TokenLine(file.Tokens[i-1]) {
			return fn
		}
		if tokCharIs(file, i, '(') {
			close := findTypeMatching(file, i, '(', ')')
			if close <= i || close > end {
				return fn
			}
			i = close - 1
			continue
		}
		if tokCharIs(file, i, '[') {
			close := findTypeMatching(file, i, '[', ']')
			if close <= i || close > end {
				return fn
			}
			i = close - 1
			continue
		}
		if tokCharIs(file, i, '{') {
			if file.Tokens[i-1].KindLine&255 == syntax.TokenStruct || file.Tokens[i-1].KindLine&255 == syntax.TokenInterface {
				close := findTypeMatching(file, i, '{', '}')
				if close <= i || close > end {
					return fn
				}
				i = close - 1
				continue
			}
			fn.ResultEnd, fn.BodyStart = i, i
			fn.BodyEnd = findTypeMatching(file, i, '{', '}')
			fn.EndTok = fn.BodyEnd
			return fn
		}
	}
	return fn
}

func (c *genericExpressionContext) scanClosure(fn syntax.FuncDecl) {
	e := c.specializer.environment
	file := &e.graph.Packages[c.scope.pkg].Files[c.scope.file].File
	inner := newGenericExpressionContext(c.specializer, c.scope, fn, c.arguments)
	inner.specialize = c.specialize
	inner.bindings = append(append([]scopedTypeBinding(nil), c.bindings...), inner.bindings...)
	inner.ranges = append(inner.ranges, c.ranges...)
	inner.tuples = append(inner.tuples, c.tuples...)
	inner.receiverName, inner.receiverType = c.receiverName, c.receiverType
	if c.specialize {
		signature := buildFuncSignature(file, &fn)
		id := e.types.substitute(e.signature(c.scope, signature), c.scope.parameters, c.arguments)
		var shadowed []string
		for _, field := range signature.Params {
			shadowed = append(shadowed, field.Name)
		}
		for _, field := range signature.Results {
			shadowed = append(shadowed, field.Name)
		}
		for _, binding := range c.bindings {
			if binding.name >= 0 && binding.visible <= fn.StartTok && fn.StartTok < binding.end {
				shadowed = append(shadowed, tokenString(file, binding.name))
			}
		}
		aliases := ""
		hiddenTypes := append(c.typeNamesHiddenAt(fn.StartTok), shadowed...)
		for index := range e.decls {
			d := &e.decls[index]
			if d.owner == 0 || d.pkg != c.scope.pkg || d.file != c.scope.file || d.token >= fn.StartTok || fn.StartTok >= d.scopeEnd || e.localType(c.scope, d.name, fn.StartTok) != index {
				continue
			}
			hidden := false
			for _, name := range shadowed {
				if name == d.name {
					hidden = true
				}
			}
			if hidden {
				continue
			}
			id := e.types.substitute(e.resolveDeclaration(index), c.scope.parameters, c.arguments)
			aliases += "type " + d.name + " = " + c.specializer.scopedTypeText(id, c.scope.pkg, hiddenTypes) + "\n"
			shadowed = append(shadowed, d.name)
			hiddenTypes = append(hiddenTypes, d.name)
		}
		aliases += c.specializer.parameterAliases(c.scope.parameters, c.arguments, c.scope.pkg, shadowed, hiddenTypes)
		// Closure lowering may lift the body into a package function. Give
		// its signature concrete types and keep its lexical type aliases in
		// an outer block so original local declarations can still shadow them.
		inner.replace(fn.StartTok, fn.BodyStart+1, "func"+c.specializer.scopedSignatureText(e.types.get(id), c.scope.pkg, signature.Params, signature.Results, c.typeNamesHiddenAt(fn.StartTok))+" {\n"+aliases+" { ")
		inner.replace(fn.BodyEnd-1, fn.BodyEnd, "} }")
		inner.validateBody(e.types.get(id))
	} else {
		inner.validateBody(e.types.get(e.signature(c.scope, buildFuncSignature(file, &fn))))
	}
	c.changes = append(c.changes, inner.changes...)
}
