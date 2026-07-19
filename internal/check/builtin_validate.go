package check

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

const (
	builtinTypeUnknown = iota
	builtinTypeOrdered
	builtinTypeString
	builtinTypeMap
	builtinTypeSlice
	builtinTypeInvalid
)

func invalidBuiltinCalls(pkg *load.Package, info *PackageInfo, fileIndex int, fn syntax.FuncDecl, signature *FuncSignature, calls []int) (int, int) {
	file := &pkg.Files[fileIndex].File
	var locals []definiteLocalTypeSpan
	localsReady := false
	for call := 0; call < len(calls); call++ {
		callee := calls[call]
		open := callee + 1
		name := tokenString(file, callee)
		close := findTypeMatching(*file, open, '(', ')')
		if close <= open || close > fn.BodyEnd {
			continue
		}
		args := splitExprList(*file, open+1, close-1)
		if !localsReady {
			locals = collectDefiniteLocalTypes(*file, fn)
			localsReady = true
		}
		if name == "clear" {
			if len(args) != 1 {
				return CheckErrBuiltinArity, callee
			}
			kind := definiteBuiltinExprType(pkg, info, fileIndex, signature, locals, args[0], callee, 0)
			if kind != builtinTypeUnknown && kind != builtinTypeMap && kind != builtinTypeSlice {
				return CheckErrBuiltinOperand, args[0].StartTok
			}
			continue
		}
		if len(args) == 0 {
			return CheckErrBuiltinArity, callee
		}
		common := builtinTypeUnknown
		commonType := ""
		for i := 0; i < len(args); i++ {
			kind := definiteBuiltinExprType(pkg, info, fileIndex, signature, locals, args[i], callee, 0)
			if kind == builtinTypeInvalid || kind == builtinTypeMap || kind == builtinTypeSlice {
				return CheckErrBuiltinOperand, args[i].StartTok
			}
			if kind != builtinTypeUnknown {
				if common != builtinTypeUnknown && common != kind {
					return CheckErrBuiltinOperand, args[i].StartTok
				}
				common = kind
			}
			typ := definiteBuiltinExprTypeName(pkg, info, fileIndex, signature, locals, args[i], callee, 0)
			if typ == "!" || typ != "" && commonType != "" && typ != commonType {
				return CheckErrBuiltinOperand, args[i].StartTok
			}
			if typ != "" {
				commonType = typ
			}
		}
	}
	return CheckOK, -1
}

func definiteBuiltinExprTypeName(pkg *load.Package, info *PackageInfo, fileIndex int, signature *FuncSignature, locals []definiteLocalTypeSpan, span ExprSpan, before int, depth int) string {
	if depth > len(info.Types)+2 {
		return ""
	}
	file := &pkg.Files[fileIndex].File
	start, end := trimExprSpan(*file, span.StartTok, span.EndTok)
	start, end = stripOuterParens(*file, start, end)
	if start < 0 || end <= start {
		return ""
	}
	if end-start == 1 && file.Tokens[start].KindLine&255 == syntax.TokenIdent {
		return definiteBuiltinNamedValueTypeName(pkg, info, fileIndex, signature, locals, start, before, depth+1)
	}
	if file.Tokens[start].KindLine&255 == syntax.TokenIdent && start+1 < end && tokCharIs(file, start+1, '(') {
		name := tokenString(file, start)
		if name != "make" {
			return definiteBuiltinCanonicalTypeName(pkg, info, name, depth+1)
		}
	}
	paren, bracket, brace := 0, 0, 0
	for i := start; i < end; i++ {
		if tokCharIs(file, i, '(') {
			paren++
		} else if tokCharIs(file, i, ')') {
			paren--
		} else if tokCharIs(file, i, '[') {
			bracket++
		} else if tokCharIs(file, i, ']') {
			bracket--
		} else if tokCharIs(file, i, '{') {
			brace++
		} else if tokCharIs(file, i, '}') {
			brace--
		} else if paren == 0 && bracket == 0 && brace == 0 && isExprBinaryOp(*file, i) {
			left := definiteBuiltinExprTypeName(pkg, info, fileIndex, signature, locals, ExprSpan{StartTok: start, EndTok: i}, before, depth+1)
			right := definiteBuiltinExprTypeName(pkg, info, fileIndex, signature, locals, ExprSpan{StartTok: i + 1, EndTok: end}, before, depth+1)
			if left != "" && right != "" && left != right {
				return "!"
			}
			if left != "" {
				return left
			}
			return right
		}
	}
	return ""
}

func definiteBuiltinNamedValueTypeName(pkg *load.Package, info *PackageInfo, fileIndex int, signature *FuncSignature, locals []definiteLocalTypeSpan, nameTok int, before int, depth int) string {
	file := &pkg.Files[fileIndex].File
	name := tokenString(file, nameTok)
	for i := 0; i < len(signature.Receiver); i++ {
		if signature.Receiver[i].Name == name {
			return definiteBuiltinTypeSpanName(pkg, info, fileIndex, signature.Receiver[i].TypeStart, signature.Receiver[i].TypeEnd, depth+1)
		}
	}
	for i := 0; i < len(signature.Params); i++ {
		if signature.Params[i].Name == name {
			return definiteBuiltinTypeSpanName(pkg, info, fileIndex, signature.Params[i].TypeStart, signature.Params[i].TypeEnd, depth+1)
		}
	}
	for i := 0; i < len(signature.Results); i++ {
		if signature.Results[i].Name == name {
			return definiteBuiltinTypeSpanName(pkg, info, fileIndex, signature.Results[i].TypeStart, signature.Results[i].TypeEnd, depth+1)
		}
	}
	if typeStart, typeEnd, ok := findDefiniteLocalType(file, locals, nameTok, before); ok && typeStart >= 0 {
		return definiteBuiltinTypeSpanName(pkg, info, fileIndex, typeStart, typeEnd, depth+1)
	}
	for i := 0; i < len(info.Decls); i++ {
		decl := info.Decls[i]
		if decl.Name == name && decl.Kind == SymbolVar && decl.TypeStart >= 0 {
			return definiteBuiltinTypeSpanName(pkg, info, decl.File, decl.TypeStart, decl.TypeEnd, depth+1)
		}
	}
	return ""
}

func definiteBuiltinTypeSpanName(pkg *load.Package, info *PackageInfo, fileIndex int, start int, end int, depth int) string {
	if fileIndex < 0 || fileIndex >= len(pkg.Files) {
		return ""
	}
	file := &pkg.Files[fileIndex].File
	start, end = trimTypeSpan(*file, start, end)
	if end-start != 1 || file.Tokens[start].KindLine&255 != syntax.TokenIdent {
		return ""
	}
	return definiteBuiltinCanonicalTypeName(pkg, info, tokenString(file, start), depth+1)
}

func definiteBuiltinCanonicalTypeName(pkg *load.Package, info *PackageInfo, name string, depth int) string {
	if name == "string" || name == "float32" || name == "float64" || name == "int" || name == "int8" || name == "int16" || name == "int32" || name == "int64" || name == "uint" || name == "uint8" || name == "uint16" || name == "uint32" || name == "uint64" || name == "uintptr" || name == "byte" || name == "rune" {
		return name
	}
	if depth > len(info.Types)+2 {
		return name
	}
	typeIndex := LookupType(*info, name)
	if typeIndex < 0 {
		return ""
	}
	typ := info.Types[typeIndex]
	if !typ.Alias {
		return name
	}
	return definiteBuiltinTypeSpanName(pkg, info, typ.File, typ.TypeStart, typ.TypeEnd, depth+1)
}

func definiteBuiltinExprType(pkg *load.Package, info *PackageInfo, fileIndex int, signature *FuncSignature, locals []definiteLocalTypeSpan, span ExprSpan, before int, depth int) int {
	if depth > len(info.Types)+2 {
		return builtinTypeUnknown
	}
	file := &pkg.Files[fileIndex].File
	start, end := trimExprSpan(*file, span.StartTok, span.EndTok)
	start, end = stripOuterParens(*file, start, end)
	if start < 0 || end <= start {
		return builtinTypeUnknown
	}
	if end-start == 1 {
		kind := file.Tokens[start].KindLine & 255
		if kind == syntax.TokenString {
			return builtinTypeString
		}
		if kind == syntax.TokenNumber || kind == syntax.TokenChar {
			return builtinTypeOrdered
		}
		if tokenTextIs(file, start, "true") || tokenTextIs(file, start, "false") || tokenTextIs(file, start, "nil") {
			return builtinTypeInvalid
		}
		if kind == syntax.TokenIdent {
			name := tokenString(file, start)
			if category := definiteBuiltinNamedValueType(pkg, info, fileIndex, signature, locals, name, start, before, depth); category != builtinTypeUnknown {
				return category
			}
		}
	}
	if file.Tokens[start].KindLine&255 == syntax.TokenIdent && start+1 < end && tokCharIs(file, start+1, '(') {
		name := tokenString(file, start)
		if name == "make" && start+2 < end {
			return definiteBuiltinTypeSpan(pkg, info, fileIndex, start+2, nextTopLevelComma(*file, start+2, end-1), depth+1)
		}
		return definiteBuiltinTypeName(pkg, info, name, depth+1)
	}
	for i := start; i < end; i++ {
		if isExprBinaryOp(*file, i) {
			left := definiteBuiltinExprType(pkg, info, fileIndex, signature, locals, ExprSpan{StartTok: start, EndTok: i}, before, depth+1)
			right := definiteBuiltinExprType(pkg, info, fileIndex, signature, locals, ExprSpan{StartTok: i + 1, EndTok: end}, before, depth+1)
			if left == right {
				return left
			}
			if left == builtinTypeUnknown {
				return right
			}
			if right == builtinTypeUnknown {
				return left
			}
			return builtinTypeInvalid
		}
	}
	return builtinTypeUnknown
}

func definiteBuiltinNamedValueType(pkg *load.Package, info *PackageInfo, fileIndex int, signature *FuncSignature, locals []definiteLocalTypeSpan, name string, nameTok int, before int, depth int) int {
	for i := 0; i < len(signature.Receiver); i++ {
		if signature.Receiver[i].Name == name {
			return definiteBuiltinTypeSpan(pkg, info, fileIndex, signature.Receiver[i].TypeStart, signature.Receiver[i].TypeEnd, depth+1)
		}
	}
	fields := append(signature.Params, signature.Results...)
	for i := 0; i < len(fields); i++ {
		if fields[i].Name == name {
			return definiteBuiltinTypeSpan(pkg, info, fileIndex, fields[i].TypeStart, fields[i].TypeEnd, depth+1)
		}
	}
	file := fileForPackage(pkg, fileIndex)
	if typeStart, typeEnd, ok := findDefiniteLocalType(file, locals, nameTok, before); ok && typeStart >= 0 {
		return definiteBuiltinTypeSpan(pkg, info, fileIndex, typeStart, typeEnd, depth+1)
	}
	for i := 0; i < len(info.Decls); i++ {
		decl := info.Decls[i]
		if decl.Name == name && decl.Kind == SymbolVar && decl.TypeStart >= 0 {
			return definiteBuiltinTypeSpan(pkg, info, decl.File, decl.TypeStart, decl.TypeEnd, depth+1)
		}
	}
	return builtinTypeUnknown
}

func definiteBuiltinTypeSpan(pkg *load.Package, info *PackageInfo, fileIndex int, start int, end int, depth int) int {
	if fileIndex < 0 || fileIndex >= len(pkg.Files) {
		return builtinTypeUnknown
	}
	file := pkg.Files[fileIndex].File
	start, end = trimTypeSpan(file, start, end)
	if start < 0 || end <= start {
		return builtinTypeUnknown
	}
	if file.Tokens[start].KindLine&255 == syntax.TokenMap {
		return builtinTypeMap
	}
	if tokCharIs(&file, start, '[') {
		if start+1 < end && tokCharIs(&file, start+1, ']') {
			return builtinTypeSlice
		}
		return builtinTypeInvalid
	}
	if file.Tokens[start].KindLine&255 != syntax.TokenIdent {
		return builtinTypeInvalid
	}
	return definiteBuiltinTypeName(pkg, info, tokenString(&file, start), depth+1)
}

func definiteBuiltinTypeName(pkg *load.Package, info *PackageInfo, name string, depth int) int {
	if name == "string" {
		return builtinTypeString
	}
	if name == "byte" || name == "rune" || name == "float32" || name == "float64" || name == "int" || name == "int8" || name == "int16" || name == "int32" || name == "int64" || name == "uint" || name == "uint8" || name == "uint16" || name == "uint32" || name == "uint64" || name == "uintptr" {
		return builtinTypeOrdered
	}
	if name == "bool" || name == "error" {
		return builtinTypeInvalid
	}
	if depth > len(info.Types)+2 {
		return builtinTypeUnknown
	}
	typeIndex := LookupType(*info, name)
	if typeIndex < 0 {
		return builtinTypeUnknown
	}
	typ := info.Types[typeIndex]
	return definiteBuiltinTypeSpan(pkg, info, typ.File, typ.TypeStart, typ.TypeEnd, depth+1)
}

func fileForPackage(pkg *load.Package, fileIndex int) *syntax.File {
	return &pkg.Files[fileIndex].File
}
