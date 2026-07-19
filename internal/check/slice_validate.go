package check

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

func invalidDefiniteSliceOperand(pkg load.Package, info PackageInfo, fileIndex int, fn syntax.FuncDecl) int {
	file := pkg.Files[fileIndex].File
	for open := fn.BodyStart + 1; open < fn.BodyEnd; open++ {
		if !tokCharIs(&file, open, '[') {
			continue
		}
		close := findTypeMatching(file, open, '[', ']')
		if close <= open || close > fn.BodyEnd || findTypeTopLevelChar(file, open+1, close-1, ':') < 0 {
			continue
		}
		start, end := stripOuterParens(file, exprOperandStartBefore(file, fn.BodyStart+1, open), open)
		array := false
		if start < end && tokCharIs(&file, end-1, '}') {
			typeEnd := findTypeTopLevelChar(file, start, end, '{')
			array = definiteArrayType(pkg, info, file, start, typeEnd)
		} else if end-start >= 3 && file.Tokens[start].KindLine&255 == syntax.TokenIdent && tokCharIs(&file, end-1, ')') {
			calleeFile, callee, ok := findDefinitePackageFunc(&pkg, &info, &file, start)
			if ok {
				signature := buildFuncSignature(pkg.Files[calleeFile].File, callee)
				if len(signature.Results) == 1 {
					result := signature.Results[0]
					array = definiteArrayType(pkg, info, pkg.Files[calleeFile].File, result.TypeStart, result.TypeEnd)
				}
			}
		}
		if array {
			return open
		}
	}
	return -1
}

func definiteArrayType(pkg load.Package, info PackageInfo, file syntax.File, start int, end int) bool {
	for depth := 0; depth <= len(info.Types); depth++ {
		start, end = trimTypeSpan(file, start, end)
		if classifyType(file, start, end) == TypeArray {
			return true
		}
		if end != start+1 {
			return false
		}
		typeIndex := LookupType(info, tokenString(&file, start))
		if typeIndex < 0 {
			return false
		}
		typ := info.Types[typeIndex]
		if typ.File < 0 || typ.File >= len(pkg.Files) {
			return false
		}
		file = pkg.Files[typ.File].File
		start, end = typ.TypeStart, typ.TypeEnd
	}
	return false
}
