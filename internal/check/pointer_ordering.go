package check

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

func invalidResolvedOperatorOperands(pkg *load.Package, info *PackageInfo, fileIndex int, fn syntax.FuncDecl, body *syntax.Body, signature *FuncSignature, scope CoreScope, cachedBindings *[]scopedTypeBinding) int {
	file := &pkg.Files[fileIndex].File
	bindings := *cachedBindings
	ready := bindings != nil
	for op := fn.BodyStart + 1; op < fn.BodyEnd-1; op++ {
		if file.Tokens[op].KindLine&255 == syntax.TokenFunc {
			// A literal has its own parameters and locals. Do not borrow the
			// enclosing declaration's type for a same-named closure parameter.
			op = pointerOrderingNestedFunctionEnd(*file, op, fn.BodyEnd-1)
			continue
		}
		// Most tokens cannot be ordering or shift operators. Reject them
		// before checking their full spelling.
		kind := file.Tokens[op].KindLine
		if kind&255 != syntax.TokenOperator {
			continue
		}
		first := file.Src[file.Tokens[op].Start]
		if first != '<' && first != '>' {
			continue
		}
		shift := tokenTextIs(file, op, "<<") || tokenTextIs(file, op, ">>")
		if !shift && !tokenTextIs(file, op, "<") && !tokenTextIs(file, op, "<=") && !tokenTextIs(file, op, ">") && !tokenTextIs(file, op, ">=") {
			continue
		}
		if !ready {
			bindings = collectScopedTypeBindings(*file, fn, *body, signature)
			*cachedBindings = bindings
			ready = true
		}
		left := operandBoundary(file, op-1, fn.BodyStart+1, -1, false) + 1
		right := operandBoundary(file, op+1, fn.BodyEnd-1, 1, false)
		if shift {
			left, right = shiftOperandBounds(file, left, op, right)
			if invalidKnownShiftOperand(pkg, info, fileIndex, scope, bindings, left, op, op) || invalidKnownShiftOperand(pkg, info, fileIndex, scope, bindings, op+1, right, op) {
				return op
			}
			continue
		}
		if definiteOrderingExprKind(pkg, info, fileIndex, scope, bindings, left, op, op, 0) != 0 || definiteOrderingExprKind(pkg, info, fileIndex, scope, bindings, op+1, right, op, 0) != 0 {
			return op
		}
	}
	return -1
}

// Low bits retain pointer depth; bit 256 records a definite struct base.
// Resolving both properties together avoids visiting each operand twice.
func definiteOrderingExprKind(pkg *load.Package, info *PackageInfo, fileIndex int, scope CoreScope, bindings []scopedTypeBinding, start, end, before, depth int) int {
	if depth > 32 || start < 0 || start >= end {
		return 0
	}
	file := &pkg.Files[fileIndex].File
	start, end = trimExprSpan(*file, start, end)
	start, end = stripOuterParens(file, start, end)
	if start < 0 || start >= end {
		return 0
	}
	if tokCharIs(file, start, '&') {
		return 1 + definiteOrderingExprKind(pkg, info, fileIndex, scope, bindings, start+1, end, before, depth+1)
	}
	if tokCharIs(file, start, '*') {
		n := definiteOrderingExprKind(pkg, info, fileIndex, scope, bindings, start+1, end, before, depth+1)
		if n&255 > 0 {
			return n - 1
		}
		return 0
	}
	if start+1 < end && tokCharIs(file, end-1, ')') {
		if tokenTextIs(file, start, "new") && tokCharIs(file, start+1, '(') && findTypeMatching(file, start+1, '(', ')') == end && lookupScopeTokenNameCore(scope, file, start) < 0 && lookupPackageSymbol(info.Symbols, "new") < 0 {
			return 1 + definiteOrderingTypeKind(pkg, info, fileIndex, scope, start+2, end-1, 0)
		}
		if tokCharIs(file, start, '(') {
			close := findTypeMatching(file, start, '(', ')')
			if close > start && close < end && tokCharIs(file, close, '(') && findTypeMatching(file, close, '(', ')') == end {
				return definiteOrderingTypeKind(pkg, info, fileIndex, scope, start+1, close-1, 0)
			}
		}
		if file.Tokens[start].KindLine&255 == syntax.TokenIdent && tokCharIs(file, start+1, '(') && findTypeMatching(file, start+1, '(', ')') == end && lookupScopeTokenNameCore(scope, file, start) < 0 {
			if typ := lookupType(info.Types, tokenString(file, start)); typ >= 0 {
				return definiteOrderingTypeKind(pkg, info, fileIndex, scope, start, start+1, 0)
			}
			calleeFile, callee, ok := findDefinitePackageFunc(pkg, info, file, start)
			if ok {
				signature := buildFuncSignature(pkg.Files[calleeFile].File, callee)
				if len(signature.Results) == 1 {
					result := signature.Results[0]
					return definiteOrderingTypeKind(pkg, info, calleeFile, CoreScope{}, result.TypeStart, result.TypeEnd, 0)
				}
			}
		}
	}
	if tokCharIs(file, end-1, '}') {
		for open := end - 2; open >= start; open-- {
			if tokCharIs(file, open, '{') && findTypeMatching(file, open, '{', '}') == end {
				return definiteOrderingTypeKind(pkg, info, fileIndex, scope, start, open, depth+1)
			}
		}
	}
	if end-start != 1 || file.Tokens[start].KindLine&255 != syntax.TokenIdent {
		return 0
	}
	chosen := -1
	for i, binding := range bindings {
		if binding.visible <= before && before < binding.end && coreTokensEqual(file, binding.name, start) && (chosen < 0 || binding.visible > bindings[chosen].visible) {
			chosen = i
		}
	}
	if chosen >= 0 {
		binding := bindings[chosen]
		if binding.typeEnd > binding.typeStart {
			return definiteOrderingTypeKind(pkg, info, fileIndex, scope, binding.typeStart, binding.typeEnd, 0)
		}
		return definiteOrderingExprKind(pkg, info, fileIndex, scope, bindings, binding.valueStart, binding.valueEnd, binding.name, depth+1)
	}
	name := tokenString(file, start)
	if name == "nil" && lookupPackageSymbol(info.Symbols, name) < 0 && lookupScopeTokenNameCore(scope, file, start) < 0 {
		return 1
	}
	for _, decl := range info.Decls {
		if decl.Name != name || decl.Kind != SymbolVar {
			continue
		}
		if decl.TypeEnd > decl.TypeStart {
			return definiteOrderingTypeKind(pkg, info, decl.File, CoreScope{}, decl.TypeStart, decl.TypeEnd, 0)
		}
		values := splitExprList(pkg.Files[decl.File].File, decl.ValueStart, decl.ValueEnd)
		if decl.ValueIndex >= 0 && decl.ValueIndex < len(values) {
			value := values[decl.ValueIndex]
			return definiteOrderingExprKind(pkg, info, decl.File, CoreScope{}, nil, value.StartTok, value.EndTok, decl.Token, depth+1)
		}
	}
	return 0
}

func definiteOrderingTypeKind(pkg *load.Package, info *PackageInfo, fileIndex int, scope CoreScope, start, end, depth int) int {
	if depth > len(info.Types)+32 {
		return 0
	}
	file := &pkg.Files[fileIndex].File
	start, end = stripOuterParens(file, start, end)
	if start < 0 || start >= end {
		return 0
	}
	if file.Tokens[start].KindLine&255 == syntax.TokenStruct && tokCharIs(file, start+1, '{') && findTypeMatching(file, start+1, '{', '}') == end {
		return 256
	}
	if tokCharIs(file, start, '*') {
		return 1 + definiteOrderingTypeKind(pkg, info, fileIndex, scope, start+1, end, depth+1)
	}
	if end-start != 1 || lookupScopeTokenNameCore(scope, file, start) >= 0 {
		return 0
	}
	index := lookupType(info.Types, tokenString(file, start))
	if index < 0 {
		return 0
	}
	typ := info.Types[index]
	return definiteOrderingTypeKind(pkg, info, typ.File, CoreScope{}, typ.TypeStart, typ.TypeEnd, depth+1)
}
