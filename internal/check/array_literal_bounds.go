package check

import "renvo.dev/internal/load"

func invalidArrayLiteralBounds(pkg load.Package, info PackageInfo, fileIndex int, literals []CompositeExpr, scope CoreScope) int {
	file := pkg.Files[fileIndex].File
	context := constantIndexContext{pkg: &pkg, info: &info, fileIndex: fileIndex, strict: true}
	for _, literal := range literals {
		length, array := arrayLiteralLength(pkg, info, fileIndex, literal.TypeStart, literal.TypeEnd, scope, 0)
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

func arrayLiteralLength(pkg load.Package, info PackageInfo, fileIndex, start, end int, scope CoreScope, depth int) (wideConstant, bool) {
	if depth > len(info.Types)+1 || start < 0 || start >= end {
		return wideConstant{}, false
	}
	file := pkg.Files[fileIndex].File
	if classifyType(file, start, end) == TypeArray {
		lengthStart, lengthEnd, _, _ := parseArrayTypeShape(file, start, end)
		context := constantIndexContext{pkg: &pkg, info: &info, fileIndex: fileIndex, strict: true}
		return arrayLiteralConstant(context, lengthStart, lengthEnd, scope), true
	}
	if end-start != 1 || lookupScopeTokenNameCore(scope, &file, start) >= 0 {
		return wideConstant{}, false
	}
	index := LookupType(info, tokenString(&file, start))
	if index < 0 {
		return wideConstant{}, false
	}
	typ := info.Types[index]
	return arrayLiteralLength(pkg, info, typ.File, typ.TypeStart, typ.TypeEnd, CoreScope{}, depth+1)
}

func arrayLiteralConstant(context constantIndexContext, start, end int, scope CoreScope) wideConstant {
	context.scope = scope
	return wideConstantExpr(context, start, end, 0)
}
