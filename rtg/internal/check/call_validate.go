package check

import (
	"j5.nz/rtg/rtg/internal/arena"
	"j5.nz/rtg/rtg/internal/load"
	"j5.nz/rtg/rtg/internal/syntax"
)

const (
	definiteTypeUnknown = iota
	definiteTypePointer
	definiteTypeNonPointer
)

// invalidDefiniteCallArgumentType rejects representation-changing argument
// mismatches that can be proven from declarations alone. Unknown expressions
// remain for later, richer checking rather than being guessed here.
func invalidDefiniteCallArgumentType(pkg load.Package, info PackageInfo, fileIndex int, caller syntax.FuncDecl) int {
	file := pkg.Files[fileIndex].File
	callerSignature := buildFuncSignature(file, caller)
	for open := caller.BodyStart + 1; open < caller.BodyEnd; open++ {
		if !tokCharIs(file, open, '(') || open == 0 || file.Tokens[open-1].Kind != syntax.TokenIdent {
			continue
		}
		calleeTok := open - 1
		if calleeTok > caller.BodyStart && tokCharIs(file, calleeTok-1, '.') {
			continue
		}
		calleeFileIndex, callee, ok := findDefinitePackageFunc(pkg, info, file, calleeTok)
		if !ok {
			continue
		}
		if definiteCallNameShadowed(file, caller, callerSignature, calleeTok) {
			continue
		}
		close := findTypeMatching(file, open, '(', ')')
		if close <= open || close > caller.BodyEnd {
			continue
		}
		callArenaStart := arena.Mark()
		args := splitExprList(file, open+1, close-1)
		calleeFile := pkg.Files[calleeFileIndex].File
		calleeSignature := buildFuncSignature(calleeFile, callee)
		params := calleeSignature.Params
		invalidTok := -1
		for i := 0; i < len(args) && i < len(params); i++ {
			param := params[i]
			if definiteTypeKind(pkg, calleeFileIndex, param.TypeStart, param.TypeEnd, 0) != definiteTypePointer {
				continue
			}
			arg := args[i]
			if definiteArgumentTypeKind(pkg, fileIndex, caller, callerSignature, arg, calleeTok) == definiteTypeNonPointer {
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

func findDefinitePackageFunc(pkg load.Package, info PackageInfo, callerFile syntax.File, calleeTok int) (int, syntax.FuncDecl, bool) {
	symbolIndex := lookupPackageSymbolTextCore(info, callerFile, calleeTok)
	if symbolIndex < 0 || symbolIndex >= len(info.Symbols) || info.Symbols[symbolIndex].Kind != SymbolFunc {
		return -1, syntax.FuncDecl{}, false
	}
	symbol := info.Symbols[symbolIndex]
	if symbol.File < 0 || symbol.File >= len(pkg.Files) {
		return -1, syntax.FuncDecl{}, false
	}
	file := pkg.Files[symbol.File].File
	for i := 0; i < len(file.Funcs); i++ {
		fn := file.Funcs[i]
		if fn.ReceiverStart < 0 && fn.NameTok == symbol.Token {
			return symbol.File, fn, true
		}
	}
	return -1, syntax.FuncDecl{}, false
}

func definiteArgumentTypeKind(pkg load.Package, fileIndex int, caller syntax.FuncDecl, signature FuncSignature, arg ExprSpan, before int) int {
	file := pkg.Files[fileIndex].File
	start, end := trimExprSpan(file, arg.StartTok, arg.EndTok)
	if start < 0 || end <= start {
		return definiteTypeUnknown
	}
	if tokCharIs(file, start, '&') {
		return definiteTypePointer
	}
	if end-start != 1 || file.Tokens[start].Kind != syntax.TokenIdent {
		return definiteTypeUnknown
	}
	name := tokenString(file, start)
	if name == "nil" {
		return definiteTypePointer
	}
	if kind := definiteNamedFieldTypeKind(pkg, fileIndex, signature.Receiver, name); kind != definiteTypeUnknown {
		return kind
	}
	if kind := definiteNamedFieldTypeKind(pkg, fileIndex, signature.Params, name); kind != definiteTypeUnknown {
		return kind
	}
	if kind := definiteNamedFieldTypeKind(pkg, fileIndex, signature.Results, name); kind != definiteTypeUnknown {
		return kind
	}
	if typeStart, typeEnd, ok := findDefiniteLocalType(file, caller, name, before); ok {
		return definiteTypeKind(pkg, fileIndex, typeStart, typeEnd, 0)
	}
	for declFileIndex := 0; declFileIndex < len(pkg.Files); declFileIndex++ {
		declFile := pkg.Files[declFileIndex].File
		for i := 0; i < len(declFile.Decls); i++ {
			decl := declFile.Decls[i]
			if decl.Kind != syntax.TokenVar || tokenString(declFile, decl.NameTok) != name {
				continue
			}
			typeStart := declNameListEnd(declFile, decl)
			valueStart := findDeclAssign(declFile, typeStart, decl.EndTok)
			if valueStart >= 0 {
				typeStart, valueStart = trimDeclSpan(declFile, typeStart, valueStart)
				return definiteTypeKind(pkg, declFileIndex, typeStart, valueStart, 0)
			}
			typeStart, typeEnd := trimDeclSpan(declFile, typeStart, decl.EndTok)
			return definiteTypeKind(pkg, declFileIndex, typeStart, typeEnd, 0)
		}
	}
	return definiteTypeUnknown
}

func definiteNamedFieldTypeKind(pkg load.Package, fileIndex int, fields []Field, name string) int {
	for i := 0; i < len(fields); i++ {
		if fields[i].Name == name {
			return definiteTypeKind(pkg, fileIndex, fields[i].TypeStart, fields[i].TypeEnd, 0)
		}
	}
	return definiteTypeUnknown
}

func findDefiniteLocalType(file syntax.File, caller syntax.FuncDecl, name string, before int) (int, int, bool) {
	foundStart := -1
	foundEnd := -1
	for i := caller.BodyStart + 1; i < before; i++ {
		if file.Tokens[i].Kind != syntax.TokenVar {
			continue
		}
		specStart := i + 1
		if specStart < before && tokCharIs(file, specStart, '(') {
			close := findTypeMatching(file, specStart, '(', ')')
			if close <= specStart || close > before {
				continue
			}
			for j := specStart + 1; j < close-1; {
				j = skipLocalSeparators(file, j, close-1)
				specEnd := statementSpecEnd(file, j, close-1)
				if start, end, ok := definiteLocalSpecType(file, j, specEnd, name); ok {
					foundStart, foundEnd = start, end
				}
				if specEnd <= j {
					j++
				} else {
					j = specEnd
				}
			}
			i = close - 1
			continue
		}
		specEnd := statementSpecEnd(file, specStart, before)
		if start, end, ok := definiteLocalSpecType(file, specStart, specEnd, name); ok {
			foundStart, foundEnd = start, end
		}
		i = specEnd - 1
	}
	return foundStart, foundEnd, foundStart >= 0
}

func definiteLocalSpecType(file syntax.File, start int, end int, name string) (int, int, bool) {
	start, end = trimDeclSpan(file, start, end)
	if start < 0 || end <= start {
		return -1, -1, false
	}
	names, namesEnd := localDeclNameTokens(file, start, end)
	found := false
	for i := 0; i < len(names); i++ {
		if tokenString(file, names[i]) == name {
			found = true
		}
	}
	if !found {
		return -1, -1, false
	}
	typeEnd := end
	if valueStart := findDeclAssign(file, namesEnd, end); valueStart >= 0 {
		typeEnd = valueStart
	}
	typeStart, typeEnd := trimDeclSpan(file, namesEnd, typeEnd)
	return typeStart, typeEnd, typeStart >= 0 && typeEnd > typeStart
}

func definiteCallNameShadowed(file syntax.File, caller syntax.FuncDecl, signature FuncSignature, calleeTok int) bool {
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
			if file.Tokens[j].Kind == syntax.TokenIdent && statementTokensEqual(file, j, calleeTok) {
				return true
			}
		}
	}
	return false
}

func definiteFieldsHaveTokenName(file syntax.File, fields []Field, tok int) bool {
	for i := 0; i < len(fields); i++ {
		name := fields[i].Name
		token := file.Tokens[tok]
		if tokenMatchesCoreSymbol(file.Src, token.Start, token.End-token.Start, name) {
			return true
		}
	}
	return false
}

func definiteLocalNameDeclared(file syntax.File, caller syntax.FuncDecl, calleeTok int) bool {
	for i := caller.BodyStart + 1; i < calleeTok; i++ {
		kind := file.Tokens[i].Kind
		if kind != syntax.TokenVar && kind != syntax.TokenConst && kind != syntax.TokenType {
			continue
		}
		start := i + 1
		end := statementSpecEnd(file, start, calleeTok)
		if start < calleeTok && tokCharIs(file, start, '(') {
			close := findTypeMatching(file, start, '(', ')')
			if close <= start || close > calleeTok {
				continue
			}
			end = close - 1
			start++
		}
		for start < end {
			start = skipLocalSeparators(file, start, end)
			specEnd := statementSpecEnd(file, start, end)
			names, _ := localDeclNameTokens(file, start, specEnd)
			for j := 0; j < len(names); j++ {
				if statementTokensEqual(file, names[j], calleeTok) {
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

func definiteTypeKind(pkg load.Package, fileIndex int, start int, end int, depth int) int {
	if fileIndex < 0 || fileIndex >= len(pkg.Files) || depth > 16 {
		return definiteTypeUnknown
	}
	file := pkg.Files[fileIndex].File
	start, end = trimTypeSpan(file, start, end)
	if start < 0 || end <= start {
		return definiteTypeUnknown
	}
	if tokenTextIs(file, start, "...") {
		start++
	}
	if start >= end {
		return definiteTypeUnknown
	}
	if tokCharIs(file, start, '*') {
		return definiteTypePointer
	}
	if file.Tokens[start].Kind != syntax.TokenIdent {
		return definiteTypeNonPointer
	}
	if start+1 < end && tokCharIs(file, start+1, '.') {
		return definiteTypeUnknown
	}
	name := tokenString(file, start)
	for declFileIndex := 0; declFileIndex < len(pkg.Files); declFileIndex++ {
		declFile := pkg.Files[declFileIndex].File
		for i := 0; i < len(declFile.Decls); i++ {
			decl := declFile.Decls[i]
			if decl.Kind != syntax.TokenType || tokenString(declFile, decl.NameTok) != name {
				continue
			}
			typeStart := decl.NameTok + 1
			if tokenTextIs(declFile, typeStart, "=") {
				typeStart++
			}
			return definiteTypeKind(pkg, declFileIndex, typeStart, decl.EndTok, depth+1)
		}
	}
	return definiteTypeUnknown
}
