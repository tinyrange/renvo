package check

import "renvo.dev/internal/syntax"

func (e *genericEnvironment) collectLocalTypes() {
	functions := len(e.decls)
	for owner := 0; owner < functions; owner++ {
		parent := &e.decls[owner]
		if parent.kind == SymbolType || parent.function.BodyStart < 0 {
			continue
		}
		e.collectLocalTypesInFunction(owner, parent.function)
	}
}

func (e *genericEnvironment) collectLocalTypesInFunction(owner int, fn syntax.FuncDecl) {
	parent := &e.decls[owner]
	file := &e.graph.Packages[parent.pkg].Files[parent.file].File
	body := syntax.ParseFuncBodyStatements(*file, fn)
	ends := localRuleScopeEnds(&body)
	for i, stmt := range body.Stmts {
		if stmt.Kind != syntax.StmtDecl || file.Tokens[stmt.StartTok].KindLine&255 != syntax.TokenType {
			continue
		}
		start, end := stmt.StartTok+1, stmt.EndTok
		if tokCharIs(file, start, '(') {
			end = findTypeMatching(file, start, '(', ')')
			for pos := start + 1; pos < end-1; {
				pos = skipLocalSeparators(file, pos, end-1)
				if pos >= end-1 || tokCharIs(file, pos, ')') {
					break
				}
				finish := genericLocalSpecEnd(file, pos, end-1)
				e.addLocalType(owner, pos, finish, ends[i])
				if finish <= pos {
					break
				}
				pos = finish
			}
		} else {
			e.addLocalType(owner, start, end, ends[i])
		}
	}
	for i := fn.BodyStart + 1; i < fn.BodyEnd-1; i++ {
		if file.Tokens[i].KindLine&255 != syntax.TokenFunc {
			continue
		}
		inner := genericFunctionLiteral(file, i, fn.BodyEnd-1)
		if inner.BodyStart >= 0 && inner.BodyEnd > inner.BodyStart {
			e.collectLocalTypesInFunction(owner, inner)
			i = inner.BodyEnd - 1
		}
	}
}

func genericLocalSpecEnd(file *syntax.File, start int, end int) int {
	for i := start; i < end; i++ {
		if tokCharIs(file, i, ';') || i > start && (file.Tokens[i-1].KindLine>>syntax.TokenOperatorLineShift&syntax.TokenLineLimit) < (file.Tokens[i].KindLine>>syntax.TokenOperatorLineShift&syntax.TokenLineLimit) {
			return i
		}
		for _, pair := range []string{"()", "[]", "{}"} {
			if tokCharIs(file, i, pair[0]) {
				close := findTypeMatching(file, i, pair[0], pair[1])
				if close > i {
					i = close - 1
				}
				break
			}
		}
	}
	return end
}

func (e *genericEnvironment) addLocalType(owner int, start int, end int, scopeEnd int) {
	parent := &e.decls[owner]
	file := &e.graph.Packages[parent.pkg].Files[parent.file].File
	start, end = trimDeclSpan(file, start, end)
	if start >= end || file.Tokens[start].KindLine&255 != syntax.TokenIdent {
		return
	}
	d := genericDeclaration{pkg: parent.pkg, file: parent.file, token: start, kind: SymbolType, name: tokenString(file, start), start: start + 1, end: end, owner: owner + 1, scopeEnd: scopeEnd, parameters: parent.parameters}
	d.alias = tokCharIs(file, d.start, '=')
	if d.alias {
		d.start++
	} else {
		origin := e.graph.Packages[d.pkg].Ref.ImportPath + "/local/" + genericDecimal(d.file) + "/" + genericDecimal(d.token)
		d.typ = e.types.intern(genericType{kind: genericNamed, name: d.name, origin: origin, args: d.parameters})
	}
	e.decls = append(e.decls, d)
}

func (e *genericEnvironment) localType(scope genericTypeScope, name string, token int) int {
	chosen := -1
	for i := range e.decls {
		d := &e.decls[i]
		if d.owner != 0 && d.pkg == scope.pkg && d.file == scope.file && d.name == name && d.token <= token && token < d.scopeEnd && (chosen < 0 || d.token > e.decls[chosen].token) {
			chosen = i
		}
	}
	return chosen
}
