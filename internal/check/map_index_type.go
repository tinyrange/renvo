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

type mapIndexShape struct {
	key                        string
	file, valueStart, valueEnd int
	scope                      CoreScope
	known                      bool
}

func invalidMapIndexType(pkg load.Package, info PackageInfo, fileIndex int, fn syntax.FuncDecl, body syntax.Body) (int, int) {
	file := pkg.Files[fileIndex].File
	indexes := buildFuncIndexExprs(&file, &body)
	if len(indexes) == 0 {
		return CheckOK, -1
	}
	scope, ok, _ := buildFuncScopeCore(file, fn)
	if !ok {
		return CheckOK, -1
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
		shape := mapIndexExprShape(pkg, info, fileIndex, scope, bindings, index.BaseStart, index.BaseEnd, index.OpenTok, 0)
		if mapLiteralPrimitiveMismatch(file, index.IndexStart, index.IndexEnd, shape.key) {
			return CheckErrType, index.IndexStart
		}
		if tok := invalidMapElementFieldWrite(pkg, info, file, body, index, shape); tok >= 0 {
			return CheckErrAssignTarget, tok
		}
	}
	return CheckOK, -1
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

func mapIndexExprShape(pkg load.Package, info PackageInfo, fileIndex int, scope CoreScope, bindings []mapIndexBinding, start, end, before, depth int) mapIndexShape {
	if depth > 32 {
		return mapIndexShape{}
	}
	file := pkg.Files[fileIndex].File
	start, end = stripOuterParens(file, start, end)
	if start < 0 || start >= end {
		return mapIndexShape{}
	}
	if tokCharIs(&file, end-1, '}') {
		open, braces := -1, 0
		for tok := end - 1; tok >= start; tok-- {
			if tokCharIs(&file, tok, '}') {
				braces++
			}
			if tokCharIs(&file, tok, '{') {
				braces--
				if braces == 0 {
					open = tok
					break
				}
			}
		}
		if open >= 0 && findTypeMatching(file, open, '{', '}') == end {
			return mapIndexTypeShape(pkg, info, fileIndex, start, open, scope, 0)
		}
	}
	if tokenTextIs(&file, start, "make") && start+1 < end && tokCharIs(&file, start+1, '(') && findTypeMatching(file, start+1, '(', ')') == end && lookupScopeTokenNameCore(scope, &file, start) < 0 && LookupPackageSymbol(info, "make") < 0 {
		return mapIndexTypeShape(pkg, info, fileIndex, start+2, nextTopLevelComma(file, start+2, end-1), scope, 0)
	}
	if end-start != 1 || file.Tokens[start].KindLine&255 != syntax.TokenIdent {
		return mapIndexShape{}
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
			return mapIndexTypeShape(pkg, info, fileIndex, binding.typeStart, binding.typeEnd, scope, 0)
		}
		return mapIndexExprShape(pkg, info, fileIndex, scope, bindings, binding.valueStart, binding.valueEnd, binding.name, depth+1)
	}
	name := tokenString(&file, start)
	for _, decl := range info.Decls {
		if decl.Name != name || decl.Kind != SymbolVar {
			continue
		}
		declFile := pkg.Files[decl.File].File
		if decl.TypeEnd > decl.TypeStart {
			return mapIndexTypeShape(pkg, info, decl.File, decl.TypeStart, decl.TypeEnd, CoreScope{}, 0)
		}
		values := splitExprList(declFile, decl.ValueStart, decl.ValueEnd)
		if decl.ValueIndex >= 0 && decl.ValueIndex < len(values) {
			value := values[decl.ValueIndex]
			return mapIndexExprShape(pkg, info, decl.File, CoreScope{}, nil, value.StartTok, value.EndTok, decl.Token, depth+1)
		}
	}
	return mapIndexShape{}
}

func mapIndexTypeShape(pkg load.Package, info PackageInfo, fileIndex, start, end int, scope CoreScope, depth int) mapIndexShape {
	if start < 0 || start >= end || depth > len(info.Types)+1 {
		return mapIndexShape{}
	}
	file := pkg.Files[fileIndex].File
	if file.Tokens[start].KindLine&255 == syntax.TokenMap {
		ks, ke, vs, ve := parseMapTypeShape(file, start, end)
		return mapIndexShape{mapLiteralPrimitiveType(pkg, info, file, ks, ke, scope, 0), fileIndex, vs, ve, scope, true}
	}
	if end-start != 1 || lookupScopeTokenNameCore(scope, &file, start) >= 0 {
		return mapIndexShape{}
	}
	index := LookupType(info, tokenString(&file, start))
	if index < 0 {
		return mapIndexShape{}
	}
	typ := info.Types[index]
	return mapIndexTypeShape(pkg, info, typ.File, typ.TypeStart, typ.TypeEnd, CoreScope{}, depth+1)
}
