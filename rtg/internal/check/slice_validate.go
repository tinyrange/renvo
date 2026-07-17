package check

import (
	"j5.nz/rtg/rtg/internal/load"
	"j5.nz/rtg/rtg/internal/syntax"
)

// invalidDefiniteSliceOperand rejects array operands that are provably not
// addressable. Slice and string values do not have Go's addressability
// requirement, so unknown operands deliberately remain for richer checking.
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
		baseStart := exprOperandStartBefore(file, fn.BodyStart+1, open)
		baseStart, baseEnd := unwrapDefiniteSliceOperand(file, baseStart, open)
		if definiteArrayComposite(pkg, info, fileIndex, file, baseStart, baseEnd) || definiteArrayCall(pkg, info, fileIndex, file, baseStart, baseEnd) {
			return open
		}
	}
	return -1
}

func unwrapDefiniteSliceOperand(file syntax.File, start int, end int) (int, int) {
	for start < end && tokCharIs(&file, start, '(') && tokCharIs(&file, end-1, ')') {
		close := findTypeMatching(file, start, '(', ')')
		if close != end {
			break
		}
		start++
		end--
	}
	return start, end
}

func definiteArrayComposite(pkg load.Package, info PackageInfo, fileIndex int, file syntax.File, start int, end int) bool {
	if start >= end || !tokCharIs(&file, end-1, '}') {
		return false
	}
	open := matchingOpenBefore(file, end-1, '{', '}')
	if open <= start {
		return false
	}
	return definiteTypeSpanIsArray(pkg, info, fileIndex, start, open, 0)
}

func definiteArrayCall(pkg load.Package, info PackageInfo, fileIndex int, file syntax.File, start int, end int) bool {
	if end-start < 3 || file.Tokens[start].Kind != syntax.TokenIdent || !tokCharIs(&file, start+1, '(') || findTypeMatching(file, start+1, '(', ')') != end {
		return false
	}
	calleeFileIndex, callee, ok := findDefinitePackageFunc(&pkg, &info, &file, start)
	if !ok {
		return false
	}
	calleeFile := pkg.Files[calleeFileIndex].File
	signature := buildFuncSignature(calleeFile, callee)
	if len(signature.Results) != 1 {
		return false
	}
	result := signature.Results[0]
	return definiteTypeSpanIsArray(pkg, info, calleeFileIndex, result.TypeStart, result.TypeEnd, 0)
}

func definiteTypeSpanIsArray(pkg load.Package, info PackageInfo, fileIndex int, start int, end int, depth int) bool {
	if depth > len(info.Types) || fileIndex < 0 || fileIndex >= len(pkg.Files) {
		return false
	}
	file := pkg.Files[fileIndex].File
	start, end = trimTypeSpan(file, start, end)
	if start < 0 || end <= start {
		return false
	}
	kind := classifyType(file, start, end)
	if kind == TypeArray {
		return true
	}
	if kind != TypeNamed || end != start+1 {
		return false
	}
	typeIndex := LookupType(info, tokenString(&file, start))
	if typeIndex < 0 {
		return false
	}
	typ := info.Types[typeIndex]
	return definiteTypeSpanIsArray(pkg, info, typ.File, typ.TypeStart, typ.TypeEnd, depth+1)
}

func matchingOpenBefore(file syntax.File, close int, left byte, right byte) int {
	depth := 0
	for i := close; i >= 0; i-- {
		if tokCharIs(&file, i, right) {
			depth++
		} else if tokCharIs(&file, i, left) {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}
