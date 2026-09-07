package check

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

func invalidResolvedOperatorOperands(pkg load.Package, info PackageInfo, fileIndex int, fn syntax.FuncDecl, body syntax.Body) int {
	file := pkg.Files[fileIndex].File
	var bindings []scopedTypeBinding
	var scope CoreScope
	ready := false
	for op := fn.BodyStart + 1; op < fn.BodyEnd-1; op++ {
		if file.Tokens[op].KindLine&255 == syntax.TokenFunc {
			// A literal has its own parameters and locals. Do not borrow the
			// enclosing declaration's type for a same-named closure parameter.
			op = pointerOrderingNestedFunctionEnd(file, op, fn.BodyEnd-1)
			continue
		}
		shift := tokenTextIs(&file, op, "<<") || tokenTextIs(&file, op, ">>")
		if !shift && !tokenTextIs(&file, op, "<") && !tokenTextIs(&file, op, "<=") && !tokenTextIs(&file, op, ">") && !tokenTextIs(&file, op, ">=") {
			continue
		}
		if !ready {
			var ok bool
			scope, ok, _ = buildFuncScopeCore(file, fn)
			if !ok {
				return -1
			}
			bindings = collectScopedTypeBindings(file, fn, body)
			ready = true
		}
		left := literalOrderingBoundary(file, op-1, fn.BodyStart+1, -1) + 1
		right := literalOrderingBoundary(file, op+1, fn.BodyEnd-1, 1)
		if shift {
			left, right = shiftOperandBounds(file, left, op, right)
			if invalidKnownShiftOperand(pkg, info, fileIndex, scope, bindings, left, op, op) || invalidKnownShiftOperand(pkg, info, fileIndex, scope, bindings, op+1, right, op) {
				return op
			}
			continue
		}
		if definiteStructExpr(pkg, info, fileIndex, scope, bindings, left, op, op, 0) || definiteStructExpr(pkg, info, fileIndex, scope, bindings, op+1, right, op, 0) {
			return op
		}
		if definiteExprPointerDepth(pkg, info, fileIndex, scope, bindings, left, op, op, 0) > 0 || definiteExprPointerDepth(pkg, info, fileIndex, scope, bindings, op+1, right, op, 0) > 0 {
			return op
		}
	}
	return -1
}

func pointerOrderingNestedFunctionEnd(file syntax.File, start, end int) int {
	for tok := start + 1; tok < end; tok++ {
		if tokCharIs(&file, tok, ';') {
			return start
		}
		if tokCharIs(&file, tok, '(') || tokCharIs(&file, tok, '[') {
			open, close := byte('('), byte(')')
			if tokCharIs(&file, tok, '[') {
				open, close = '[', ']'
			}
			finish := findTypeMatching(file, tok, open, close)
			if finish <= tok {
				return start
			}
			tok = finish - 1
			continue
		}
		if tokCharIs(&file, tok, '{') {
			finish := findTypeMatching(file, tok, '{', '}')
			if finish <= tok {
				return start
			}
			if isCompositeTypeBodyOpen(file, tok) {
				tok = finish - 1
				continue
			}
			return finish - 1
		}
	}
	return start
}

// Zero means non-pointer or unresolved. Only positive pointer depth is
// rejection evidence; tracking depth preserves valid comparisons of *p.
func definiteExprPointerDepth(pkg load.Package, info PackageInfo, fileIndex int, scope CoreScope, bindings []scopedTypeBinding, start, end, before, depth int) int {
	if depth > 32 {
		return 0
	}
	file := pkg.Files[fileIndex].File
	start, end = trimExprSpan(file, start, end)
	start, end = stripOuterParens(file, start, end)
	if start < 0 || start >= end {
		return 0
	}
	if tokCharIs(&file, start, '&') {
		return 1 + definiteExprPointerDepth(pkg, info, fileIndex, scope, bindings, start+1, end, before, depth+1)
	}
	if tokCharIs(&file, start, '*') {
		n := definiteExprPointerDepth(pkg, info, fileIndex, scope, bindings, start+1, end, before, depth+1)
		if n > 0 {
			return n - 1
		}
		return 0
	}
	if start+1 < end && tokCharIs(&file, end-1, ')') {
		if tokenTextIs(&file, start, "new") && tokCharIs(&file, start+1, '(') && findTypeMatching(file, start+1, '(', ')') == end && lookupScopeTokenNameCore(scope, &file, start) < 0 && LookupPackageSymbol(info, "new") < 0 {
			return 1 + definiteTypePointerDepth(pkg, info, fileIndex, scope, start+2, end-1, 0)
		}
		if tokCharIs(&file, start, '(') {
			close := findTypeMatching(file, start, '(', ')')
			if close > start && close < end && tokCharIs(&file, close, '(') && findTypeMatching(file, close, '(', ')') == end {
				return definiteTypePointerDepth(pkg, info, fileIndex, scope, start+1, close-1, 0)
			}
		}
		if file.Tokens[start].KindLine&255 == syntax.TokenIdent && tokCharIs(&file, start+1, '(') && findTypeMatching(file, start+1, '(', ')') == end && lookupScopeTokenNameCore(scope, &file, start) < 0 {
			if typ := LookupType(info, tokenString(&file, start)); typ >= 0 {
				return definiteTypePointerDepth(pkg, info, fileIndex, scope, start, start+1, 0)
			}
			calleeFile, callee, ok := findDefinitePackageFunc(&pkg, &info, &file, start)
			if ok {
				signature := buildFuncSignature(pkg.Files[calleeFile].File, callee)
				if len(signature.Results) == 1 {
					result := signature.Results[0]
					return definiteTypePointerDepth(pkg, info, calleeFile, CoreScope{}, result.TypeStart, result.TypeEnd, 0)
				}
			}
		}
	}
	if end-start != 1 || file.Tokens[start].KindLine&255 != syntax.TokenIdent {
		return 0
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
			return definiteTypePointerDepth(pkg, info, fileIndex, scope, binding.typeStart, binding.typeEnd, 0)
		}
		return definiteExprPointerDepth(pkg, info, fileIndex, scope, bindings, binding.valueStart, binding.valueEnd, binding.name, depth+1)
	}
	name := tokenString(&file, start)
	if name == "nil" && LookupPackageSymbol(info, name) < 0 && lookupScopeTokenNameCore(scope, &file, start) < 0 {
		return 1
	}
	for _, decl := range info.Decls {
		if decl.Name != name || decl.Kind != SymbolVar {
			continue
		}
		if decl.TypeEnd > decl.TypeStart {
			return definiteTypePointerDepth(pkg, info, decl.File, CoreScope{}, decl.TypeStart, decl.TypeEnd, 0)
		}
		values := splitExprList(pkg.Files[decl.File].File, decl.ValueStart, decl.ValueEnd)
		if decl.ValueIndex >= 0 && decl.ValueIndex < len(values) {
			value := values[decl.ValueIndex]
			return definiteExprPointerDepth(pkg, info, decl.File, CoreScope{}, nil, value.StartTok, value.EndTok, decl.Token, depth+1)
		}
	}
	return 0
}

func definiteTypePointerDepth(pkg load.Package, info PackageInfo, fileIndex int, scope CoreScope, start, end, depth int) int {
	if depth > len(info.Types)+32 {
		return 0
	}
	file := pkg.Files[fileIndex].File
	start, end = stripOuterParens(file, start, end)
	if start < 0 || start >= end {
		return 0
	}
	if tokCharIs(&file, start, '*') {
		return 1 + definiteTypePointerDepth(pkg, info, fileIndex, scope, start+1, end, depth+1)
	}
	if end-start != 1 || lookupScopeTokenNameCore(scope, &file, start) >= 0 {
		return 0
	}
	index := LookupType(info, tokenString(&file, start))
	if index < 0 {
		return 0
	}
	typ := info.Types[index]
	return definiteTypePointerDepth(pkg, info, typ.File, CoreScope{}, typ.TypeStart, typ.TypeEnd, depth+1)
}
