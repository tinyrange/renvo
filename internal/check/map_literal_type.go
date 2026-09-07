package check

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

func invalidMapLiteralTypes(pkg load.Package, info PackageInfo, file syntax.File, literals []CompositeExpr, scope CoreScope) int {
	for _, literal := range literals {
		key, value := mapLiteralPrimitiveTypes(pkg, info, file, literal.TypeStart, literal.TypeEnd, scope, 0)
		if key == "" && value == "" {
			continue
		}
		for _, element := range literal.Elems {
			colon := findTypeTopLevelChar(file, element.StartTok, element.EndTok, ':')
			if colon < 0 {
				continue
			}
			if mapLiteralPrimitiveMismatch(file, element.StartTok, colon, key) {
				return element.StartTok
			}
			if mapLiteralPrimitiveMismatch(file, colon+1, element.EndTok, value) {
				return colon + 1
			}
		}
	}
	return -1
}

func mapLiteralPrimitiveTypes(pkg load.Package, info PackageInfo, file syntax.File, start int, end int, scope CoreScope, depth int) (string, string) {
	if start < 0 || start >= end || depth > len(info.Types)+1 {
		return "", ""
	}
	if file.Tokens[start].KindLine&255 == syntax.TokenMap {
		ks, ke, vs, ve := parseMapTypeShape(file, start, end)
		return mapLiteralPrimitiveType(pkg, info, file, ks, ke, scope, depth+1), mapLiteralPrimitiveType(pkg, info, file, vs, ve, scope, depth+1)
	}
	if end-start != 1 || lookupScopeTokenNameCore(scope, &file, start) >= 0 {
		return "", ""
	}
	index := LookupType(info, tokenString(&file, start))
	if index < 0 {
		return "", ""
	}
	typ := info.Types[index]
	return mapLiteralPrimitiveTypes(pkg, info, pkg.Files[typ.File].File, typ.TypeStart, typ.TypeEnd, CoreScope{}, depth+1)
}

func mapLiteralPrimitiveType(pkg load.Package, info PackageInfo, file syntax.File, start int, end int, scope CoreScope, depth int) string {
	if start < 0 || end-start != 1 || depth > len(info.Types)+2 || lookupScopeTokenNameCore(scope, &file, start) >= 0 {
		return ""
	}
	name := tokenString(&file, start)
	index := LookupType(info, name)
	if index >= 0 {
		typ := info.Types[index]
		return mapLiteralPrimitiveType(pkg, info, pkg.Files[typ.File].File, typ.TypeStart, typ.TypeEnd, CoreScope{}, depth+1)
	}
	if LookupPackageSymbol(info, name) >= 0 {
		return ""
	}
	return name
}

func mapLiteralPrimitiveMismatch(file syntax.File, start int, end int, want string) bool {
	start, end = stripOuterParens(file, start, end)
	if start < 0 || end-start != 1 {
		return false
	}
	kind := file.Tokens[start].KindLine & 255
	if kind == syntax.TokenString {
		return primitiveTypeMismatch(want, "string")
	}
	if kind == syntax.TokenNumber || kind == syntax.TokenChar {
		return primitiveTypeMismatch(want, "int")
	}
	// true/false and other identifiers require lexical name resolution.
	return false
}
