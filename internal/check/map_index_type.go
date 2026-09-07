package check

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

// Keep visibility intervals alongside declaration spans: an inner slice binding
// must not borrow the key type of an outer map with the same name.
type mapIndexBinding struct {
	name, visible, end   int
	typeStart, typeEnd   int
	valueStart, valueEnd int
}

func invalidMapIndexType(pkg load.Package, info PackageInfo, file syntax.File, fn syntax.FuncDecl, body syntax.Body) int {
	indexes := buildFuncIndexExprs(&file, &body)
	if len(indexes) == 0 {
		return -1
	}
	scope, ok, _ := buildFuncScopeCore(file, fn)
	if !ok {
		return -1
	}
	var bindings []mapIndexBinding
	signature := buildFuncSignature(file, fn)
	for group := 0; group < 3; group++ {
		fields := signature.Params
		if group == 1 {
			fields = signature.Results
		}
		if group == 2 {
			fields = signature.Receiver
		}
		for _, field := range fields {
			bindings = append(bindings, mapIndexBinding{field.NameTok, fn.BodyStart, fn.BodyEnd, field.TypeStart, field.TypeEnd, -1, -1})
		}
	}
	for _, stmt := range body.Stmts {
		start, end := stmt.StartTok, stmt.EndTok
		scopeEnd := localRuleScopeEnd(body, start)
		if stmt.Kind == syntax.StmtDecl {
			kind := file.Tokens[start].KindLine & 255
			start++
			if tokCharIs(&file, start, '(') {
				for pos := start + 1; pos < end-1; {
					pos = skipLocalSeparators(file, pos, end-1)
					if pos >= end-1 || tokCharIs(&file, pos, ')') {
						break
					}
					finish := statementSpecEnd(file, pos, end-1)
					bindings = appendMapIndexBindings(bindings, file, pos, finish, scopeEnd, kind == syntax.TokenVar, false)
					if finish <= pos {
						break
					}
					pos = finish
				}
			} else {
				bindings = appendMapIndexBindings(bindings, file, start, end, scopeEnd, kind == syntax.TokenVar, false)
			}
			continue
		}
		if stmt.Kind == syntax.StmtIf || stmt.Kind == syntax.StmtFor || stmt.Kind == syntax.StmtSwitch {
			start++
			end = stmt.BodyStart
			scopeEnd = stmt.EndTok
			if semi := findTypeTopLevelChar(file, start, end, ';'); semi >= 0 {
				end = semi
			}
		} else if stmt.Kind == syntax.StmtCase {
			start++
		} else if stmt.Kind != syntax.StmtAssign {
			continue
		}
		op := findTopLevelAssignOp(file, start, end)
		if op >= 0 && tokenTextIs(&file, op, ":=") {
			bindings = appendMapIndexBindings(bindings, file, start, end, scopeEnd, true, true)
		}
	}
	for _, index := range indexes {
		key := mapIndexExprKey(pkg, info, file, scope, bindings, index.BaseStart, index.BaseEnd, index.OpenTok, 0)
		if mapLiteralPrimitiveMismatch(file, index.IndexStart, index.IndexEnd, key) {
			return index.IndexStart
		}
	}
	return -1
}

func appendMapIndexBindings(bindings []mapIndexBinding, file syntax.File, start, end, scopeEnd int, variable, short bool) []mapIndexBinding {
	start, end = trimDeclSpan(file, start, end)
	names, namesEnd := localDeclNameTokens(file, start, end)
	op := findTopLevelAssignOp(file, start, end)
	typeStart, typeEnd := namesEnd, end
	var values []ExprSpan
	if op >= 0 {
		typeEnd = op
		values = splitExprList(file, op+1, end)
	}
	for i, name := range names {
		binding := mapIndexBinding{name, end, scopeEnd, -1, -1, -1, -1}
		if variable {
			binding.typeStart, binding.typeEnd = typeStart, typeEnd
			if len(values) == len(names) {
				binding.valueStart, binding.valueEnd = values[i].StartTok, values[i].EndTok
			}
		}
		// A same-block short declaration reuses the existing variable.
		reused := false
		for _, old := range bindings {
			if short && old.end == scopeEnd && old.visible <= start && coreTokensEqual(&file, old.name, name) {
				reused = true
			}
		}
		if !reused {
			bindings = append(bindings, binding)
		}
	}
	return bindings
}

func mapIndexExprKey(pkg load.Package, info PackageInfo, file syntax.File, scope CoreScope, bindings []mapIndexBinding, start, end, before, depth int) string {
	if depth > 32 {
		return ""
	}
	start, end = stripOuterParens(file, start, end)
	if start < 0 || start >= end {
		return ""
	}
	if tokCharIs(&file, end-1, '}') {
		open := findTypeTopLevelChar(file, start, end, '{')
		if open >= 0 && findTypeMatching(file, open, '{', '}') == end {
			key, _ := mapLiteralPrimitiveTypes(pkg, info, file, start, open, scope, 0)
			return key
		}
	}
	if tokenTextIs(&file, start, "make") && start+1 < end && tokCharIs(&file, start+1, '(') && findTypeMatching(file, start+1, '(', ')') == end && lookupScopeTokenNameCore(scope, &file, start) < 0 && LookupPackageSymbol(info, "make") < 0 {
		key, _ := mapLiteralPrimitiveTypes(pkg, info, file, start+2, nextTopLevelComma(file, start+2, end-1), scope, 0)
		return key
	}
	if end-start != 1 || file.Tokens[start].KindLine&255 != syntax.TokenIdent {
		return ""
	}
	chosen := -1
	for i, binding := range bindings {
		if binding.visible <= before && before < binding.end && coreTokensEqual(&file, binding.name, start) && (chosen < 0 || binding.visible > bindings[chosen].visible) {
			chosen = i
		}
	}
	if chosen >= 0 {
		binding := bindings[chosen]
		if binding.typeEnd > binding.typeStart {
			key, _ := mapLiteralPrimitiveTypes(pkg, info, file, binding.typeStart, binding.typeEnd, scope, 0)
			return key
		}
		return mapIndexExprKey(pkg, info, file, scope, bindings, binding.valueStart, binding.valueEnd, binding.name, depth+1)
	}
	name := tokenString(&file, start)
	for _, decl := range info.Decls {
		if decl.Name != name || decl.Kind != SymbolVar {
			continue
		}
		declFile := pkg.Files[decl.File].File
		if decl.TypeEnd > decl.TypeStart {
			key, _ := mapLiteralPrimitiveTypes(pkg, info, declFile, decl.TypeStart, decl.TypeEnd, CoreScope{}, 0)
			return key
		}
		values := splitExprList(declFile, decl.ValueStart, decl.ValueEnd)
		if decl.ValueIndex >= 0 && decl.ValueIndex < len(values) {
			value := values[decl.ValueIndex]
			return mapIndexExprKey(pkg, info, declFile, CoreScope{}, nil, value.StartTok, value.EndTok, decl.Token, depth+1)
		}
	}
	return ""
}
