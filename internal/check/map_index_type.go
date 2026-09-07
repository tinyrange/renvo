package check

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

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
	bindings := collectScopedTypeBindings(file, fn, body)
	for _, index := range indexes {
		if !numericBuiltinInNestedFunction(file, fn, index.OpenTok) {
			value := numericBuiltinExprValue(pkg, info, fileIndex, scope, bindings, index.BaseStart, index.BaseEnd, index.OpenTok, 0)
			if value.kind != "" && value.kind != "string" || definiteStructExpr(pkg, info, fileIndex, scope, bindings, index.BaseStart, index.BaseEnd, index.OpenTok, 0) || containerBuiltinExprKind(pkg, info, fileIndex, scope, bindings, index.BaseStart, index.BaseEnd, index.OpenTok, 0) == TypeChan {
				return CheckErrOperand, index.OpenTok
			}
		}
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

func mapIndexExprShape(pkg load.Package, info PackageInfo, fileIndex int, scope CoreScope, bindings []scopedTypeBinding, start, end, before, depth int) mapIndexShape {
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
