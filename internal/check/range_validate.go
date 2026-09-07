package check

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

func invalidRangeOperand(pkg load.Package, info PackageInfo, fileIndex int, fn syntax.FuncDecl, scope CoreScope) int {
	file := pkg.Files[fileIndex].File
	// Avoid parsing and collecting bindings for functions without range loops.
	found := false
	for tok := fn.BodyStart + 1; tok < fn.BodyEnd; tok++ {
		if file.Tokens[tok].KindLine&255 == syntax.TokenRange {
			found = true
			break
		}
	}
	if !found {
		return -1
	}
	body := syntax.ParseFuncBodyStatements(file, fn)
	bindings := collectScopedTypeBindings(file, fn, body)
	for _, stmt := range body.Stmts {
		if stmt.Kind != syntax.StmtFor || numericBuiltinInNestedFunction(file, fn, stmt.StartTok) {
			continue
		}
		for tok := stmt.StartTok + 1; tok < stmt.BodyStart; tok++ {
			if file.Tokens[tok].KindLine&255 != syntax.TokenRange {
				continue
			}
			start, end := tok+1, stmt.BodyStart
			value := numericBuiltinExprValue(pkg, info, fileIndex, scope, bindings, start, end, tok, 0)
			if value.kind == "bool" || value.kind == "other" || value.kind == "float" || value.kind == "complex" || rangeStructExpr(pkg, info, fileIndex, scope, bindings, start, end, tok, 0) {
				return start
			}
			break
		}
	}
	return -1
}

// Resolve only definite struct values. Unknown calls, selectors, and promoted
// types must not inherit the type of a name in a different lexical scope.
func rangeStructExpr(pkg load.Package, info PackageInfo, fileIndex int, scope CoreScope, bindings []scopedTypeBinding, start, end, before, depth int) bool {
	if depth > 32 || start < 0 || start >= end {
		return false
	}
	file := pkg.Files[fileIndex].File
	start, end = stripOuterParens(file, start, end)
	if tokCharIs(&file, end-1, '}') {
		for open := end - 2; open >= start; open-- {
			if tokCharIs(&file, open, '{') && findTypeMatching(file, open, '{', '}') == end {
				return rangeStructType(pkg, info, fileIndex, scope, start, open, 0)
			}
		}
	}
	if end-start != 1 || file.Tokens[start].KindLine&255 != syntax.TokenIdent {
		return false
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
			return rangeStructType(pkg, info, fileIndex, scope, binding.typeStart, binding.typeEnd, 0)
		}
		return rangeStructExpr(pkg, info, fileIndex, scope, bindings, binding.valueStart, binding.valueEnd, binding.name, depth+1)
	}
	for _, decl := range info.Decls {
		if decl.Name != tokenString(&file, start) || decl.Kind != SymbolVar {
			continue
		}
		if decl.TypeEnd > decl.TypeStart {
			return rangeStructType(pkg, info, decl.File, CoreScope{}, decl.TypeStart, decl.TypeEnd, 0)
		}
		values := splitExprList(pkg.Files[decl.File].File, decl.ValueStart, decl.ValueEnd)
		if decl.ValueIndex >= 0 && decl.ValueIndex < len(values) {
			value := values[decl.ValueIndex]
			return rangeStructExpr(pkg, info, decl.File, CoreScope{}, nil, value.StartTok, value.EndTok, decl.Token, depth+1)
		}
	}
	return false
}

func rangeStructType(pkg load.Package, info PackageInfo, fileIndex int, scope CoreScope, start, end, depth int) bool {
	if depth > len(info.Types)+1 || start < 0 || start >= end {
		return false
	}
	file := pkg.Files[fileIndex].File
	start, end = stripOuterParens(file, start, end)
	if file.Tokens[start].KindLine&255 == syntax.TokenStruct && tokCharIs(&file, start+1, '{') {
		return findTypeMatching(file, start+1, '{', '}') == end
	}
	if end-start != 1 || lookupScopeTokenNameCore(scope, &file, start) >= 0 {
		return false
	}
	index := LookupType(info, tokenString(&file, start))
	if index < 0 {
		return false
	}
	typ := info.Types[index]
	return rangeStructType(pkg, info, typ.File, CoreScope{}, typ.TypeStart, typ.TypeEnd, depth+1)
}
