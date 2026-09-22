package check

import "renvo.dev/internal/load"

func conversionUnderlyingType(pkg *load.Package, info *PackageInfo, fileIndex int, scope CoreScope, start, end, depth int) string {
	if depth > len(info.Types)+2 || start < 0 || start >= end {
		return ""
	}
	file := &pkg.Files[fileIndex].File
	if end-start >= 3 && tokCharIs(file, start, '[') && tokCharIs(file, start+1, ']') {
		element := conversionUnderlyingType(pkg, info, fileIndex, scope, start+2, end, depth+1)
		if element != "" {
			return "slice:" + element
		}
		return ""
	}
	if end-start != 1 || lookupScopeTokenNameCore(scope, file, start) >= 0 {
		return ""
	}
	name := tokenString(file, start)
	if index := lookupType(info.Types, name); index >= 0 {
		typ := info.Types[index]
		return conversionUnderlyingType(pkg, info, typ.File, CoreScope{}, typ.TypeStart, typ.TypeEnd, depth+1)
	}
	if lookupPackageSymbol(info.Symbols, name) >= 0 {
		return ""
	}
	if name == "string" || name == "bool" || name == "int" || name == "int8" || name == "int16" || name == "int32" || name == "int64" || name == "uint" || name == "uint8" || name == "uint16" || name == "uint32" || name == "uint64" || name == "uintptr" || name == "byte" || name == "rune" || name == "float32" || name == "float64" || name == "complex64" || name == "complex128" {
		return name
	}
	return ""
}
