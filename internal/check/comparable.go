package check

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

func nonComparableTypeSpan(pkg load.Package, info PackageInfo, file syntax.File, start int, end int, depth int) bool {
	if depth > 32 || start < 0 || start >= end {
		return false
	}
	start, end = stripOuterParens(file, start, end)
	kind := classifyType(file, start, end)
	if kind == TypeSlice || kind == TypeMap || kind == TypeFunc {
		return true
	}
	if kind == TypeArray {
		close := findTypeMatching(file, start, '[', ']')
		return close > start && nonComparableTypeSpan(pkg, info, file, close, end, depth+1)
	}
	if kind == TypeStruct {
		open := findTypeTopLevelChar(file, start, end, '{')
		if open < 0 {
			return false
		}
		fields := parseStructFields(file, open+1, end-1)
		for i := 0; i < len(fields); i++ {
			if nonComparableTypeSpan(pkg, info, file, fields[i].TypeStart, fields[i].TypeEnd, depth+1) {
				return true
			}
		}
	}
	if end-start == 1 {
		index := LookupType(info, tokenString(&file, start))
		if index >= 0 {
			typ := info.Types[index]
			return nonComparableTypeSpan(pkg, info, pkg.Files[typ.File].File, typ.TypeStart, typ.TypeEnd, depth+1)
		}
	}
	return false
}
