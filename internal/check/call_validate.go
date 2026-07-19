package check

import (
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

type definiteCallTarget struct {
	ready         bool
	pointerParams []bool
}

func prepareDefiniteCallTargets(pkg *load.Package, info *PackageInfo, refs []CoreNameRef, targets []definiteCallTarget) {
	for i := 0; i < len(refs); i++ {
		symbolIndex := refs[i].Index
		if symbolIndex >= 0 && symbolIndex < len(info.Symbols) && symbolIndex < len(targets) {
			prepareDefiniteCallTarget(pkg, info, symbolIndex, &targets[symbolIndex])
		}
	}
}

// invalidDefiniteCallArgumentType rejects representation-changing argument
// mismatches that can be proven from declarations alone. Unknown expressions
// remain for later, richer checking rather than being guessed here.
func invalidDefiniteCallArgumentType(pkg *load.Package, info *PackageInfo, fileIndex int, caller syntax.FuncDecl, callerSignature *FuncSignature, refs []CoreNameRef, targets []definiteCallTarget) int {
	file := &pkg.Files[fileIndex].File
	localTypesReady := false
	var localTypes []definiteLocalTypeSpan
	for refIndex := 0; refIndex < len(refs); refIndex++ {
		ref := refs[refIndex]
		calleeTok := ref.Token
		open := calleeTok + 1
		if open >= caller.BodyEnd || !tokCharIs(file, open, '(') {
			continue
		}
		if ref.Index < 0 || ref.Index >= len(info.Symbols) || ref.Index >= len(targets) {
			continue
		}
		firstArg := file.Tokens[open+1]
		hasLiteral := firstArg.Kind == syntax.TokenNumber || firstArg.Kind == syntax.TokenString
		if !hasLiteral && firstArg.Kind == syntax.TokenIdent {
			size := firstArg.End - firstArg.Start
			hasLiteral = size == 4 && tokenTextIs(file, open+1, "true") || size == 5 && tokenTextIs(file, open+1, "false")
		}
		target := &targets[ref.Index]
		prepareDefiniteCallTarget(pkg, info, ref.Index, target)
		if len(target.pointerParams) == 0 && !hasLiteral {
			continue
		}
		close := findTypeMatching(*file, open, '(', ')')
		if close <= open || close > caller.BodyEnd {
			continue
		}
		invalidTok := -1
		argStart := open + 1
		for i := 0; i < len(target.pointerParams) && argStart < close-1; i++ {
			argEnd := nextDefiniteCallComma(file, argStart, close-1)
			if target.pointerParams[i] &&
				definiteArgumentTypeKind(pkg, info, fileIndex, &caller, callerSignature, &localTypes, &localTypesReady, argStart, argEnd, calleeTok) == definiteTypeNonPointer {
				invalidTok = argStart
				break
			}
			argStart = argEnd + 1
		}
		if invalidTok >= 0 {
			return invalidTok
		}
		if hasLiteral {
			invalidTok = invalidDefinitePrimitiveCallAt(pkg, info, fileIndex, calleeTok, close)
			if invalidTok >= 0 {
				return invalidTok
			}
		}
	}
	return -1
}

func prepareDefiniteCallTarget(pkg *load.Package, info *PackageInfo, symbolIndex int, target *definiteCallTarget) {
	if target.ready {
		return
	}
	target.ready = true
	symbol := info.Symbols[symbolIndex]
	if symbol.Kind != SymbolFunc || symbol.File < 0 || symbol.File >= len(pkg.Files) {
		return
	}
	file := &pkg.Files[symbol.File].File
	fn, ok := findDefinitePackageFuncDecl(*file, symbol.Token)
	if !ok || !definiteSignatureHasPointer(file, fn) {
		return
	}
	target.pointerParams = renvo_runtime_ArenaPersistCheckBools(definitePointerParams(pkg, info, symbol.File, file, fn))
}

func nextDefiniteCallComma(file *syntax.File, start int, end int) int {
	parenDepth := 0
	bracketDepth := 0
	braceDepth := 0
	for i := start; i < end; i++ {
		tok := file.Tokens[i]
		c := byte(0)
		if tok.Kind == syntax.TokenOperator && tok.End == tok.Start+1 {
			c = file.Src[tok.Start]
		}
		if c == '(' {
			parenDepth++
		} else if c == ')' {
			if parenDepth > 0 {
				parenDepth--
			}
		} else if c == '[' {
			bracketDepth++
		} else if c == ']' {
			if bracketDepth > 0 {
				bracketDepth--
			}
		} else if c == '{' {
			braceDepth++
		} else if c == '}' {
			if braceDepth > 0 {
				braceDepth--
			}
		} else if parenDepth == 0 && bracketDepth == 0 && braceDepth == 0 && c == ',' {
			return i
		}
	}
	return end
}

func definiteSignatureHasPointer(file *syntax.File, fn syntax.FuncDecl) bool {
	for i := fn.ParamsStart; i < fn.ParamsEnd; i++ {
		if tokCharIs(file, i, '*') {
			return true
		}
	}
	return false
}

func definitePointerParams(pkg *load.Package, info *PackageInfo, fileIndex int, file *syntax.File, fn syntax.FuncDecl) []bool {
	var params []bool
	pending := 0
	start := fn.ParamsStart + 1
	end := fn.ParamsEnd - 1
	for start < end {
		segmentEnd := nextTopLevelComma(*file, start, end)
		first, last := trimFieldSpan(*file, start, segmentEnd)
		if first >= last {
			start = segmentEnd + 1
			continue
		}
		if isSingleIdent(*file, first, last) {
			pending++
			start = segmentEnd + 1
			continue
		}
		if file.Tokens[first].Kind == syntax.TokenIdent && first+1 < last && !tokCharIs(file, first+1, '.') {
			pointer := definiteTypeKind(pkg, info, fileIndex, first+1, last, 0) == definiteTypePointer
			for i := 0; i <= pending; i++ {
				params = append(params, pointer)
			}
			pending = 0
		} else {
			for pending > 0 {
				params = append(params, false)
				pending--
			}
			params = append(params, definiteTypeKind(pkg, info, fileIndex, first, last, 0) == definiteTypePointer)
		}
		start = segmentEnd + 1
	}
	for pending > 0 {
		params = append(params, false)
		pending--
	}
	return params
}

func findDefinitePackageFuncDecl(file syntax.File, token int) (syntax.FuncDecl, bool) {
	low := 0
	high := len(file.Funcs)
	for low < high {
		mid := low + (high-low)/2
		if file.Funcs[mid].NameTok < token {
			low = mid + 1
		} else {
			high = mid
		}
	}
	if low < len(file.Funcs) {
		fn := file.Funcs[low]
		if fn.ReceiverStart < 0 && fn.NameTok == token {
			return fn, true
		}
	}
	return syntax.FuncDecl{}, false
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
	fn, ok := findDefinitePackageFuncDecl(pkg.Files[symbol.File].File, symbol.Token)
	return symbol.File, fn, ok
}

func definiteArgumentTypeKind(pkg *load.Package, info *PackageInfo, fileIndex int, caller *syntax.FuncDecl, signature *FuncSignature, localTypes *[]definiteLocalTypeSpan, localTypesReady *bool, start int, end int, before int) int {
	file := &pkg.Files[fileIndex].File
	for start < end && (tokCharIs(file, start, ';') || tokCharIs(file, start, ',')) {
		start++
	}
	for end > start && (tokCharIs(file, end-1, ';') || tokCharIs(file, end-1, ',')) {
		end--
	}
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
	if kind := definiteNamedFieldTypeKind(pkg, info, fileIndex, signature.Receiver, name); kind != definiteTypeUnknown {
		return kind
	}
	if kind := definiteNamedFieldTypeKind(pkg, info, fileIndex, signature.Params, name); kind != definiteTypeUnknown {
		return kind
	}
	if kind := definiteNamedFieldTypeKind(pkg, info, fileIndex, signature.Results, name); kind != definiteTypeUnknown {
		return kind
	}
	if !*localTypesReady {
		*localTypes = collectDefiniteLocalTypes(*file, *caller)
		*localTypesReady = true
	}
	if typeStart, typeEnd, ok := findDefiniteLocalType(file, *localTypes, start, before); ok {
		if typeStart < 0 || typeEnd <= typeStart {
			return definiteTypeUnknown
		}
		return definiteTypeKind(pkg, info, fileIndex, typeStart, typeEnd, 0)
	}
	symbolIndex := lookupPackageSymbolTextCore(info, file, start)
	for i := 0; i < len(info.Decls); i++ {
		if info.Decls[i].Symbol != symbolIndex || info.Decls[i].Kind != SymbolVar {
			continue
		}
		return definiteTypeKind(pkg, info, info.Decls[i].File, info.Decls[i].TypeStart, info.Decls[i].TypeEnd, 0)
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

func findDefiniteLocalType(file *syntax.File, locals []definiteLocalTypeSpan, nameTok int, before int) (int, int, bool) {
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
