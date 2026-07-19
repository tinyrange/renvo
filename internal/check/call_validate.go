package check

import (
	"renvo.dev/internal/arena"
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

const (
	definiteTypeUnknown = iota
	definiteTypePointer
	definiteTypeNonPointer
)

type definiteLocalTypeSpan struct {
	nameTok   int
	typeStart int
	typeEnd   int
	visible   int
}

// invalidDefiniteCallArgumentType rejects representation-changing argument
// mismatches that can be proven from declarations alone. Unknown expressions
// remain for later, richer checking rather than being guessed here.
func invalidDefiniteCallArgumentType(pkg load.Package, info PackageInfo, fileIndex int, caller syntax.FuncDecl) int {
	file := pkg.Files[fileIndex].File
	var callerSignature FuncSignature
	callerSignatureReady := false
	var localTypes []definiteLocalTypeSpan
	for open := caller.BodyStart + 1; open < caller.BodyEnd; open++ {
		if !tokCharIs(&file, open, '(') || open == 0 || file.Tokens[open-1].Kind != syntax.TokenIdent {
			continue
		}
		calleeTok := open - 1
		if calleeTok > caller.BodyStart && tokCharIs(&file, calleeTok-1, '.') {
			continue
		}
		calleeFileIndex, callee, ok := findDefinitePackageFunc(&pkg, &info, &file, calleeTok)
		if !ok {
			continue
		}
		calleeFile := pkg.Files[calleeFileIndex].File
		if !definiteSignatureHasPointer(calleeFile, callee) {
			continue
		}
		if !callerSignatureReady {
			callerSignature = buildFuncSignature(file, caller)
			localTypes = collectDefiniteLocalTypes(file, caller)
			callerSignatureReady = true
		}
		if definiteCallNameShadowed(&file, caller, callerSignature, calleeTok) {
			continue
		}
		close := findTypeMatching(file, open, '(', ')')
		if close <= open || close > caller.BodyEnd {
			continue
		}
		callArenaStart := arena.Mark()
		args := splitExprList(file, open+1, close-1)
		calleeSignature := buildFuncSignature(calleeFile, callee)
		params := calleeSignature.Params
		invalidTok := -1
		for i := 0; i < len(args) && i < len(params); i++ {
			param := params[i]
			if definiteTypeKind(&pkg, &info, calleeFileIndex, param.TypeStart, param.TypeEnd, 0) != definiteTypePointer {
				continue
			}
			arg := args[i]
			if definiteArgumentTypeKind(&pkg, &info, fileIndex, callerSignature, localTypes, arg, calleeTok) == definiteTypeNonPointer {
				invalidTok = arg.StartTok
				break
			}
		}
		arena.Reset(callArenaStart)
		if invalidTok >= 0 {
			return invalidTok
		}
	}
	return -1
}

func definiteSignatureHasPointer(file syntax.File, fn syntax.FuncDecl) bool {
	for i := fn.ParamsStart; i < fn.ParamsEnd; i++ {
		if tokCharIs(&file, i, '*') {
			return true
		}
	}
	return false
}

func findDefinitePackageFunc(pkg *load.Package, info *PackageInfo, callerFile *syntax.File, calleeTok int) (int, syntax.FuncDecl, bool) {
	symbolIndex := lookupPackageSymbolTextCore(info, callerFile, calleeTok)
	if symbolIndex < 0 || symbolIndex >= len(info.Symbols) || info.Symbols[symbolIndex].Kind != SymbolFunc {
		return -1, syntax.FuncDecl{}, false
	}
	symbol := info.Symbols[symbolIndex]
	if symbol.File < 0 || symbol.File >= len(pkg.Files) {
		return -1, syntax.FuncDecl{}, false
	}
	file := pkg.Files[symbol.File].File
	low := 0
	high := len(file.Funcs)
	for low < high {
		mid := low + (high-low)/2
		if file.Funcs[mid].NameTok < symbol.Token {
			low = mid + 1
		} else {
			high = mid
		}
	}
	if low < len(file.Funcs) {
		fn := file.Funcs[low]
		if fn.ReceiverStart < 0 && fn.NameTok == symbol.Token {
			return symbol.File, fn, true
		}
	}
	return -1, syntax.FuncDecl{}, false
}

func definiteArgumentTypeKind(pkg *load.Package, info *PackageInfo, fileIndex int, signature FuncSignature, localTypes []definiteLocalTypeSpan, arg ExprSpan, before int) int {
	file := pkg.Files[fileIndex].File
	start, end := trimExprSpan(file, arg.StartTok, arg.EndTok)
	if start < 0 || end <= start {
		return definiteTypeUnknown
	}
	if tokCharIs(&file, start, '&') {
		return definiteTypePointer
	}
	if end-start != 1 || file.Tokens[start].Kind != syntax.TokenIdent {
		return definiteTypeUnknown
	}
	name := tokenString(&file, start)
	if name == "nil" {
		return definiteTypePointer
	}
	if kind := definiteNamedFieldTypeKind(pkg, info, fileIndex, signature.Receiver, name); kind != definiteTypeUnknown {
		return kind
	}
	if kind := definiteNamedFieldTypeKind(pkg, info, fileIndex, signature.Params, name); kind != definiteTypeUnknown {
		return kind
	}
	if kind := definiteNamedFieldTypeKind(pkg, info, fileIndex, signature.Results, name); kind != definiteTypeUnknown {
		return kind
	}
	if typeStart, typeEnd, ok := findDefiniteLocalType(file, localTypes, start, before); ok {
		if typeStart < 0 || typeEnd <= typeStart {
			return definiteTypeUnknown
		}
		return definiteTypeKind(pkg, info, fileIndex, typeStart, typeEnd, 0)
	}
	symbolIndex := lookupPackageSymbolTextCore(info, &file, start)
	for i := 0; i < len(info.Decls); i++ {
		declInfo := info.Decls[i]
		if declInfo.Symbol != symbolIndex || declInfo.Kind != SymbolVar {
			continue
		}
		return definiteTypeKind(pkg, info, declInfo.File, declInfo.TypeStart, declInfo.TypeEnd, 0)
	}
	return definiteTypeUnknown
}

func definiteNamedFieldTypeKind(pkg *load.Package, info *PackageInfo, fileIndex int, fields []Field, name string) int {
	for i := 0; i < len(fields); i++ {
		if fields[i].Name == name {
			return definiteTypeKind(pkg, info, fileIndex, fields[i].TypeStart, fields[i].TypeEnd, 0)
		}
	}
	return definiteTypeUnknown
}

func findDefiniteLocalType(file syntax.File, locals []definiteLocalTypeSpan, nameTok int, before int) (int, int, bool) {
	foundStart := -1
	foundEnd := -1
	found := false
	for i := 0; i < len(locals); i++ {
		if locals[i].visible <= before && statementTokensEqual(file, locals[i].nameTok, nameTok) {
			foundStart = locals[i].typeStart
			foundEnd = locals[i].typeEnd
			found = true
		}
	}
	return foundStart, foundEnd, found
}

func collectDefiniteLocalTypes(file syntax.File, caller syntax.FuncDecl) []definiteLocalTypeSpan {
	var locals []definiteLocalTypeSpan
	for i := caller.BodyStart + 1; i < caller.BodyEnd; i++ {
		if file.Tokens[i].Kind != syntax.TokenVar {
			continue
		}
		specStart := i + 1
		if specStart < caller.BodyEnd && tokCharIs(&file, specStart, '(') {
			close := findTypeMatching(file, specStart, '(', ')')
			if close <= specStart || close > caller.BodyEnd {
				continue
			}
			for j := specStart + 1; j < close-1; {
				j = skipLocalSeparators(file, j, close-1)
				specEnd := statementSpecEnd(file, j, close-1)
				locals = appendDefiniteLocalSpecTypes(locals, file, j, specEnd)
				if specEnd <= j {
					j++
				} else {
					j = specEnd
				}
			}
			i = close - 1
			continue
		}
		specEnd := statementSpecEnd(file, specStart, caller.BodyEnd)
		locals = appendDefiniteLocalSpecTypes(locals, file, specStart, specEnd)
		i = specEnd - 1
	}
	return locals
}

func appendDefiniteLocalSpecTypes(locals []definiteLocalTypeSpan, file syntax.File, start int, end int) []definiteLocalTypeSpan {
	start, end = trimDeclSpan(file, start, end)
	if start < 0 || end <= start {
		return locals
	}
	names, namesEnd := localDeclNameTokens(file, start, end)
	typeEnd := end
	if valueStart := findDeclAssign(file, namesEnd, end); valueStart >= 0 {
		typeEnd = valueStart
	}
	typeStart, typeEnd := trimDeclSpan(file, namesEnd, typeEnd)
	for i := 0; i < len(names); i++ {
		locals = append(locals, definiteLocalTypeSpan{nameTok: names[i], typeStart: typeStart, typeEnd: typeEnd, visible: end})
	}
	return locals
}

func definiteCallNameShadowed(file *syntax.File, caller syntax.FuncDecl, signature FuncSignature, calleeTok int) bool {
	if definiteFieldsHaveTokenName(file, signature.Receiver, calleeTok) || definiteFieldsHaveTokenName(file, signature.Params, calleeTok) || definiteFieldsHaveTokenName(file, signature.Results, calleeTok) {
		return true
	}
	if definiteLocalNameDeclared(file, caller, calleeTok) {
		return true
	}
	for i := caller.BodyStart + 1; i < calleeTok; i++ {
		if !tokenTextIs(file, i, ":=") {
			continue
		}
		for j := i - 1; j > caller.BodyStart; j-- {
			if tokCharIs(file, j, ';') || tokCharIs(file, j, '{') || tokCharIs(file, j, '}') || file.Tokens[j].Line != file.Tokens[i].Line {
				break
			}
			if file.Tokens[j].Kind == syntax.TokenIdent && statementTokensEqual(*file, j, calleeTok) {
				return true
			}
		}
	}
	return false
}

func definiteFieldsHaveTokenName(file *syntax.File, fields []Field, tok int) bool {
	for i := 0; i < len(fields); i++ {
		name := fields[i].Name
		token := file.Tokens[tok]
		if tokenMatchesCoreSymbol(file.Src, token.Start, token.End-token.Start, name) {
			return true
		}
	}
	return false
}

func definiteLocalNameDeclared(file *syntax.File, caller syntax.FuncDecl, calleeTok int) bool {
	for i := caller.BodyStart + 1; i < calleeTok; i++ {
		kind := file.Tokens[i].Kind
		if kind != syntax.TokenVar && kind != syntax.TokenConst && kind != syntax.TokenType {
			continue
		}
		start := i + 1
		end := statementSpecEnd(*file, start, calleeTok)
		if start < calleeTok && tokCharIs(file, start, '(') {
			close := findTypeMatching(*file, start, '(', ')')
			if close <= start || close > calleeTok {
				continue
			}
			end = close - 1
			start++
		}
		for start < end {
			start = skipLocalSeparators(*file, start, end)
			specEnd := statementSpecEnd(*file, start, end)
			names, _ := localDeclNameTokens(*file, start, specEnd)
			for j := 0; j < len(names); j++ {
				if statementTokensEqual(*file, names[j], calleeTok) {
					return true
				}
			}
			if specEnd <= start {
				start++
			} else {
				start = specEnd
			}
		}
	}
	return false
}

func definiteTypeKind(pkg *load.Package, info *PackageInfo, fileIndex int, start int, end int, depth int) int {
	if fileIndex < 0 || fileIndex >= len(pkg.Files) || depth > 16 {
		return definiteTypeUnknown
	}
	file := pkg.Files[fileIndex].File
	start, end = trimTypeSpan(file, start, end)
	if start < 0 || end <= start {
		return definiteTypeUnknown
	}
	if tokenTextIs(&file, start, "...") {
		start++
	}
	if start >= end {
		return definiteTypeUnknown
	}
	if tokCharIs(&file, start, '*') {
		return definiteTypePointer
	}
	if file.Tokens[start].Kind != syntax.TokenIdent {
		return definiteTypeNonPointer
	}
	if start+1 < end && tokCharIs(&file, start+1, '.') {
		return definiteTypeUnknown
	}
	symbolIndex := lookupPackageSymbolTextCore(info, &file, start)
	if symbolIndex < 0 || symbolIndex >= len(info.Symbols) || info.Symbols[symbolIndex].Kind != SymbolType {
		return definiteTypeUnknown
	}
	for i := 0; i < len(info.Types); i++ {
		typ := info.Types[i]
		if typ.Symbol != symbolIndex {
			continue
		}
		if typ.Kind == TypePointer {
			return definiteTypePointer
		}
		if typ.Kind != TypeNamed {
			return definiteTypeNonPointer
		}
		return definiteTypeKind(pkg, info, typ.File, typ.TypeStart, typ.TypeEnd, depth+1)
	}
	return definiteTypeUnknown
}
