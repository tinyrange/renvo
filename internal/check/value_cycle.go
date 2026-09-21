package check

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

// Value containment must be acyclic even for zero-length arrays and blank
// fields. Pointer, slice, map, channel, and function representations terminate
// containment; their referents may legitimately mention the enclosing type.
func invalidRecursiveValueType(pkg load.Package, info PackageInfo) (int, int) {
	states := make([]byte, len(info.Types))
	for i := 0; i < len(info.Types); i++ {
		if recursiveValueNamedType(pkg, info, i, states) {
			return info.Types[i].File, info.Types[i].Token
		}
	}
	return -1, -1
}

func recursiveValueNamedType(pkg load.Package, info PackageInfo, index int, states []byte) bool {
	if states[index] != 0 {
		return states[index] == 1
	}
	states[index] = 1
	typ := info.Types[index]
	if recursiveValueTypeSpan(pkg, info, pkg.Files[typ.File].File, typ.TypeStart, typ.TypeEnd, states) {
		return true
	}
	states[index] = 2
	return false
}

func recursiveValueTypeSpan(pkg load.Package, info PackageInfo, file syntax.File, start int, end int, states []byte) bool {
	start, end = stripOuterParens(file, start, end)
	if start < 0 || start >= end {
		return false
	}
	kind := classifyType(file, start, end)
	if kind == TypeNamed && end-start == 1 {
		index := LookupType(info, tokenString(&file, start))
		return index >= 0 && recursiveValueNamedType(pkg, info, index, states)
	}
	if kind == TypeArray {
		close := findTypeMatching(file, start, '[', ']')
		return close > start && recursiveValueTypeSpan(pkg, info, file, close, end, states)
	}
	if kind == TypeStruct {
		open := findTypeTopLevelChar(file, start, end, '{')
		if open < 0 {
			return false
		}
		fields := parseStructFields(file, open+1, end-1)
		for i := 0; i < len(fields); i++ {
			if recursiveValueTypeSpan(pkg, info, file, fields[i].TypeStart, fields[i].TypeEnd, states) {
				return true
			}
		}
	}
	return false
}
