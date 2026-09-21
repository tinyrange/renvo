package check

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

func invalidMakeBuiltinCall(pkg *load.Package, info *PackageInfo, fileIndex int, scope CoreScope, callee, close int, args []ExprSpan) (int, int) {
	file := pkg.Files[fileIndex].File
	if len(args) == 0 || len(args) > 3 || tokenTextIs(&file, close-2, "...") {
		return CheckErrBuiltinArity, callee
	}
	typ := makeAllocationType(*pkg, *info, fileIndex, args[0].StartTok, args[0].EndTok, scope, 0)
	if typ < 0 {
		return CheckErrBuiltinOperand, args[0].StartTok
	}
	if typ == TypeSlice && len(args) < 2 || (typ == TypeMap || typ == TypeChan) && len(args) > 2 {
		return CheckErrBuiltinArity, callee
	}
	context := constantIndexContext{pkg: pkg, info: info, fileIndex: fileIndex, strict: true}
	var length wideConstant
	for i := 1; i < len(args); i++ {
		arg := args[i]
		start, end := stripOuterParens(file, arg.StartTok, arg.EndTok)
		if end-start == 1 && (file.Tokens[start].KindLine&255 == syntax.TokenString || (tokenTextIs(&file, start, "true") || tokenTextIs(&file, start, "false") || tokenTextIs(&file, start, "nil")) && lookupScopeTokenNameCore(scope, &file, start) < 0 && LookupPackageSymbol(*info, tokenString(&file, start)) < 0) {
			return CheckErrBuiltinOperand, arg.StartTok
		}
		if unsafeAddFractionalDecimal(file, start, end) {
			return CheckErrBuiltinOperand, arg.StartTok
		}
		value := arrayLiteralConstant(context, start, end, scope)
		if value.ok && (value.negative || len(value.words) > 5 || len(value.words) == 5 && value.words[4] >= 8) {
			return CheckErrBuiltinOperand, arg.StartTok
		}
		if i == 1 {
			length = value
		}
		if i == 2 && typ == TypeSlice && length.ok && value.ok && wideMagnitudeCompare(length, value) > 0 {
			return CheckErrBuiltinOperand, arg.StartTok
		}
	}
	return CheckOK, -1
}

// Zero denotes an unresolved type, positive values the permitted allocation
// families, and -1 a definitely invalid first argument.
func makeAllocationType(pkg load.Package, info PackageInfo, fileIndex, start, end int, scope CoreScope, depth int) int {
	if depth > len(info.Types)+1 || start < 0 || start >= end {
		return 0
	}
	file := pkg.Files[fileIndex].File
	start, end = stripOuterParens(file, start, end)
	kind := classifyType(file, start, end)
	if kind == TypeSlice || kind == TypeMap || kind == TypeChan {
		for open := start; open < end; open++ {
			if !tokCharIs(&file, open, '{') {
				continue
			}
			if !isCompositeTypeBodyOpen(file, open) {
				return -1
			}
			close := findTypeMatching(file, open, '{', '}')
			if close <= open {
				return 0
			}
			open = close - 1
		}
		return kind
	}
	if kind == TypeArray || kind == TypePointer || kind == TypeStruct || kind == TypeInterface || kind == TypeFunc {
		return -1
	}
	if end-start != 1 {
		return 0
	}
	if file.Tokens[start].KindLine&255 != syntax.TokenIdent {
		return -1
	}
	if lookupScopeTokenNameCore(scope, &file, start) >= 0 {
		return 0
	}
	name := tokenString(&file, start)
	index := LookupType(info, name)
	if index >= 0 {
		typ := info.Types[index]
		return makeAllocationType(pkg, info, typ.File, typ.TypeStart, typ.TypeEnd, CoreScope{}, depth+1)
	}
	if definiteBuiltinType(name) || name == "float32" || name == "float64" || name == "complex64" || name == "complex128" || name == "any" || name == "error" || name == "true" || name == "false" || name == "nil" || LookupPackageSymbol(info, name) >= 0 {
		return -1
	}
	return 0
}
