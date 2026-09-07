package check

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

func invalidArrayLiteralBounds(pkg load.Package, info PackageInfo, fileIndex int, literals []CompositeExpr, scope CoreScope, fn syntax.FuncDecl, bindings []scopedTypeBinding) int {
	file := pkg.Files[fileIndex].File
	for _, literal := range literals {
		context := constantIndexContext{pkg: &pkg, info: &info, fileIndex: fileIndex, strict: true, scope: scope, before: literal.TypeStart}
		// Enclosing bindings do not describe a nested function's declarations.
		if fn.BodyEnd > 0 && !numericBuiltinInNestedFunction(file, fn, literal.TypeStart) {
			context.bindings = bindings
		}
		length, array := arrayLiteralLength(context, literal.TypeStart, literal.TypeEnd, 0)
		if !array {
			continue
		}
		next := wideSmall(0)
		for _, element := range literal.Elems {
			index := next
			colon := findTypeTopLevelChar(file, element.StartTok, element.EndTok, ':')
			if colon >= 0 {
				index = arrayLiteralConstant(context, element.StartTok, colon, scope)
			}
			if index.ok && (index.negative || length.ok && !length.negative && wideMagnitudeCompare(index, length) >= 0) {
				return element.StartTok
			}
			next = wideAdd(index, wideSmall(1))
		}
	}
	return -1
}

func arrayLiteralLength(context constantIndexContext, start, end, depth int) (wideConstant, bool) {
	if depth > len(context.info.Types)+1 || start < 0 || start >= end {
		return wideConstant{}, false
	}
	file := context.pkg.Files[context.fileIndex].File
	if classifyType(file, start, end) == TypeArray {
		lengthStart, lengthEnd, _, _ := parseArrayTypeShape(file, start, end)
		return wideConstantExpr(context, lengthStart, lengthEnd, 0), true
	}
	if end-start != 1 || lookupScopeTokenNameCore(context.scope, &file, start) >= 0 {
		return wideConstant{}, false
	}
	index := LookupType(*context.info, tokenString(&file, start))
	if index < 0 {
		return wideConstant{}, false
	}
	typ := context.info.Types[index]
	context.fileIndex = typ.File
	context.bindings = nil
	context.scope = CoreScope{}
	return arrayLiteralLength(context, typ.TypeStart, typ.TypeEnd, depth+1)
}

func arrayLiteralConstant(context constantIndexContext, start, end int, scope CoreScope) wideConstant {
	context.scope = scope
	return wideConstantExpr(context, start, end, 0)
}
