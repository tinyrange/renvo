package check

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

// copy and append's expanded form both require a source slice with identical
// element type, or a string source with a byte-slice destination. sourceEnd
// excludes append's ellipsis; untyped nil stays unresolved and valid there.
func invalidSliceTransferElements(pkg load.Package, info PackageInfo, fileIndex int, scope CoreScope, bindings []scopedTypeBinding, before int, dst, src ExprSpan, sourceEnd int) int {
	destination := copySliceElement(pkg, info, fileIndex, scope, bindings, dst.StartTok, dst.EndTok, before, 0)
	source := copySliceElement(pkg, info, fileIndex, scope, bindings, src.StartTok, sourceEnd, before, 0)
	value := numericBuiltinExprValue(pkg, info, fileIndex, scope, bindings, src.StartTok, sourceEnd, before, 0)
	if destination != "" && (source != "" && destination != source || value.kind == "string" && destination != "uint8") {
		return src.StartTok
	}
	return -1
}

// A known element identity is stronger than an underlying scalar kind: copy
// permits different named slice types, but requires identical element types.
func copySliceElement(pkg load.Package, info PackageInfo, fileIndex int, scope CoreScope, bindings []scopedTypeBinding, start, end, before, depth int) string {
	if depth > 32 || start < 0 || start >= end {
		return ""
	}
	file := pkg.Files[fileIndex].File
	start, end = stripOuterParens(file, start, end)
	if tokCharIs(&file, end-1, '}') {
		for open := end - 2; open >= start; open-- {
			if tokCharIs(&file, open, '{') && findTypeMatching(file, open, '{', '}') == end {
				return copySliceTypeElement(pkg, info, fileIndex, scope, start, open, 0)
			}
		}
	}
	if start+1 < end && tokCharIs(&file, start+1, '(') && findTypeMatching(file, start+1, '(', ')') == end && lookupScopeTokenNameCore(scope, &file, start) < 0 {
		name := tokenString(&file, start)
		if name == "make" && LookupPackageSymbol(info, name) < 0 {
			return copySliceTypeElement(pkg, info, fileIndex, scope, start+2, nextTopLevelComma(file, start+2, end-1), 0)
		}
		if LookupType(info, name) >= 0 {
			return copySliceTypeElement(pkg, info, fileIndex, scope, start, start+1, 0)
		}
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
			return copySliceTypeElement(pkg, info, fileIndex, scope, binding.typeStart, binding.typeEnd, 0)
		}
		return copySliceElement(pkg, info, fileIndex, scope, bindings, binding.valueStart, binding.valueEnd, binding.name, depth+1)
	}
	for _, decl := range info.Decls {
		if decl.Kind != SymbolVar || decl.Name != tokenString(&file, start) {
			continue
		}
		if decl.TypeEnd > decl.TypeStart {
			return copySliceTypeElement(pkg, info, decl.File, CoreScope{}, decl.TypeStart, decl.TypeEnd, 0)
		}
		values := splitExprList(pkg.Files[decl.File].File, decl.ValueStart, decl.ValueEnd)
		if decl.ValueIndex >= 0 && decl.ValueIndex < len(values) {
			value := values[decl.ValueIndex]
			return copySliceElement(pkg, info, decl.File, CoreScope{}, nil, value.StartTok, value.EndTok, decl.Token, depth+1)
		}
	}
	return ""
}

func copySliceTypeElement(pkg load.Package, info PackageInfo, fileIndex int, scope CoreScope, start, end, depth int) string {
	if start < 0 || start >= end || depth > len(info.Types)+2 {
		return ""
	}
	file := pkg.Files[fileIndex].File
	start, end = stripOuterParens(file, start, end)
	if end-start >= 3 && tokCharIs(&file, start, '[') && tokCharIs(&file, start+1, ']') {
		return copyElementIdentity(pkg, info, fileIndex, scope, start+2, end, 0)
	}
	if end-start != 1 || lookupScopeTokenNameCore(scope, &file, start) >= 0 {
		return ""
	}
	index := LookupType(info, tokenString(&file, start))
	if index < 0 {
		return ""
	}
	typ := info.Types[index]
	return copySliceTypeElement(pkg, info, typ.File, CoreScope{}, typ.TypeStart, typ.TypeEnd, depth+1)
}

func copyElementIdentity(pkg load.Package, info PackageInfo, fileIndex int, scope CoreScope, start, end, depth int) string {
	if start < 0 || start >= end || depth > len(info.Types)+32 {
		return ""
	}
	file := pkg.Files[fileIndex].File
	start, end = stripOuterParens(file, start, end)
	if tokCharIs(&file, start, '*') {
		base := copyElementIdentity(pkg, info, fileIndex, scope, start+1, end, depth+1)
		if base != "" {
			return "*" + base
		}
		return ""
	}
	if end-start >= 3 && tokCharIs(&file, start, '[') && tokCharIs(&file, start+1, ']') {
		base := copyElementIdentity(pkg, info, fileIndex, scope, start+2, end, depth+1)
		if base != "" {
			return "[]" + base
		}
		return ""
	}
	if end-start != 1 || lookupScopeTokenNameCore(scope, &file, start) >= 0 {
		return ""
	}
	name := tokenString(&file, start)
	if index := LookupType(info, name); index >= 0 {
		typ := info.Types[index]
		if !typ.Alias {
			return "named:" + name
		}
		return copyElementIdentity(pkg, info, typ.File, CoreScope{}, typ.TypeStart, typ.TypeEnd, depth+1)
	}
	if LookupPackageSymbol(info, name) >= 0 {
		return ""
	}
	if name == "byte" {
		return "uint8"
	}
	if name == "rune" {
		return "int32"
	}
	return conversionUnderlyingType(pkg, info, fileIndex, scope, start, end, 0)
}
