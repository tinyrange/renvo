//go:build rtg

package check

import (
	"j5.nz/rtg/rtg/internal/load"
	"j5.nz/rtg/rtg/internal/syntax"
)

func CheckGraphCore(graph load.Graph) Program {
	prog := Program{
		Graph:        graph,
		Ok:           true,
		Error:        CheckOK,
		ErrorPackage: -1,
		ErrorFile:    -1,
		ErrorToken:   -1,
	}
	if !graph.Ok {
		return checkFail(prog, CheckErrGraph, graph.ErrorPackage, -1, -1)
	}
	prog.Packages = make([]PackageInfo, 0, len(graph.Packages))
	for i := 0; i < len(graph.Packages); i++ {
		info, ok, err, file, tok := checkPackageHeader(graph, i)
		prog.Packages = append(prog.Packages, info)
		if !ok {
			return checkFail(prog, err, i, file, tok)
		}
	}
	for i := 0; i < len(graph.Packages); i++ {
		info, ok, err, file, tok := checkPackageBodyCore(graph, i, prog.Packages[i], prog.Packages)
		prog.Packages[i] = info
		if !ok {
			return checkFail(prog, err, i, file, tok)
		}
	}
	return prog
}

func checkPackageBodyCore(graph load.Graph, pkgIndex int, info PackageInfo, checked []PackageInfo) (PackageInfo, bool, int, int, int) {
	pkg := graph.Packages[pkgIndex]
	info.Decls = make([]DeclInfo, 0, countPackageDeclsCore(pkg))
	info.Bodies = make([]FuncBody, 0, countPackageFuncsCore(pkg))
	for fileIndex := 0; fileIndex < len(pkg.Files); fileIndex++ {
		file := pkg.Files[fileIndex].File
		for i := 0; i < len(file.Decls); i++ {
			info.Decls = append(info.Decls, buildDeclInfoCore(file, fileIndex, info, checked, file.Decls[i]))
		}
	}
	sortDecls(info.Decls)
	info.DeclOrder = make([]int, len(info.Decls))
	for i := 0; i < len(info.DeclOrder); i++ {
		info.DeclOrder[i] = i
	}
	info.Types = make([]TypeInfo, 0, countTypeDeclsCore(info.Decls))
	for i := 0; i < len(info.Decls); i++ {
		decl := info.Decls[i]
		if decl.Kind == SymbolType {
			file := pkg.Files[decl.File].File
			info.Types = append(info.Types, buildTypeInfo(file, decl, i))
		}
	}
	sortTypes(info.Types)
	info.TypeRefs = buildPackageTypeRefsCore(pkg, info, checked)
	for fileIndex := 0; fileIndex < len(pkg.Files); fileIndex++ {
		file := pkg.Files[fileIndex].File
		for i := 0; i < len(file.Funcs); i++ {
			fn := file.Funcs[i]
			signature := buildFuncSignature(file, fn)
			scope, ok, scopeTok := buildFuncScopeCore(file, fn)
			if !ok {
				return info, false, CheckErrScope, fileIndex, scopeTok
			}
			bodyStart := fn.BodyStart + 1
			bodyEnd := fn.BodyEnd - 1
			var out FuncBody
			out.Kind = coreFuncKind(fn)
			out.File = fileIndex
			out.Func = i
			out.Signature = signature
			out.Scope = scope
			bodyTokens := bodyEnd - bodyStart
			out.Refs = make([]NameRef, 0, coreRefCapacity(bodyTokens))
			out.Selectors = make([]SelectorRef, 0, coreSelectorCapacity(bodyTokens))
			out.Refs = appendExprPackageRefsCore(out.Refs, file, fileIndex, info, scope, bodyStart, bodyEnd)
			out.Selectors = appendImportSelectorsCore(out.Selectors, file, fileIndex, info, checked, scope, bodyStart, bodyEnd)
			out.Locals = buildFuncLocalTypeSpansCore(file, fn)
			out.TypeRefs = buildFuncTypeRefsCore(file, fileIndex, info, checked, signature, out.Locals, scope)
			info.Bodies = append(info.Bodies, out)
		}
	}
	return info, true, CheckOK, -1, -1
}

func countPackageDeclsCore(pkg load.Package) int {
	count := 0
	for i := 0; i < len(pkg.Files); i++ {
		count += len(pkg.Files[i].File.Decls)
	}
	return count
}

func countPackageFuncsCore(pkg load.Package) int {
	count := 0
	for i := 0; i < len(pkg.Files); i++ {
		count += len(pkg.Files[i].File.Funcs)
	}
	return count
}

func countTypeDeclsCore(decls []DeclInfo) int {
	count := 0
	for i := 0; i < len(decls); i++ {
		if decls[i].Kind == SymbolType {
			count++
		}
	}
	return count
}

func buildDeclInfoCore(file syntax.File, fileIndex int, info PackageInfo, checked []PackageInfo, decl syntax.TopDecl) DeclInfo {
	name := tokenString(file, decl.NameTok)
	out := DeclInfo{
		Name:       name,
		Kind:       declSymbolKind(decl.Kind),
		File:       fileIndex,
		Token:      decl.NameTok,
		Symbol:     LookupPackageSymbol(info, name),
		ValueIndex: declNameIndex(file, decl),
		TypeStart:  -1,
		TypeEnd:    -1,
		ValueStart: -1,
		ValueEnd:   -1,
	}
	if decl.Kind == syntax.TokenType {
		typeStart := decl.NameTok + 1
		if tokenTextIs(file, typeStart, "=") {
			out.Alias = true
			typeStart++
		}
		out.TypeStart, out.TypeEnd = trimDeclSpan(file, typeStart, decl.EndTok)
		return out
	}
	typeStart := declNameListEnd(file, decl)
	valueStart := findDeclAssign(file, typeStart, decl.EndTok)
	if valueStart >= 0 {
		out.TypeStart, out.TypeEnd = trimDeclSpan(file, typeStart, valueStart)
		out.ValueStart, out.ValueEnd = trimDeclSpan(file, valueStart+1, decl.EndTok)
		valueTokens := out.ValueEnd - out.ValueStart
		out.Refs = make([]NameRef, 0, coreRefCapacity(valueTokens))
		out.Selectors = make([]SelectorRef, 0, coreSelectorCapacity(valueTokens))
		out.Refs = appendExprPackageRefsCore(out.Refs, file, fileIndex, info, FuncScope{}, out.ValueStart, out.ValueEnd)
		out.Selectors = appendImportSelectorsCore(out.Selectors, file, fileIndex, info, checked, FuncScope{}, out.ValueStart, out.ValueEnd)
	} else {
		out.TypeStart, out.TypeEnd = trimDeclSpan(file, typeStart, decl.EndTok)
	}
	return out
}

func appendExprPackageRefsCore(refs []NameRef, file syntax.File, fileIndex int, info PackageInfo, scope FuncScope, start int, end int) []NameRef {
	for i := start; i < end && i < len(file.Tokens); i++ {
		if file.Tokens[i].Kind != syntax.TokenIdent || shouldSkipIdentRef(file, i, end) {
			continue
		}
		if tokenTextIs(file, i, "_") || lookupScopeTokenNameCore(scope, file, i) >= 0 || lookupImportTokenNameCore(info, fileIndex, file, i) >= 0 {
			continue
		}
		symbolIndex := lookupPackageSymbolTokenCore(info, file, fileIndex, i)
		if symbolIndex >= 0 {
			refs = append(refs, NameRef{Kind: RefPackage, Token: i, Index: symbolIndex, Package: info.Symbols[symbolIndex].Package})
		}
	}
	return refs
}

func appendImportSelectorsCore(selectors []SelectorRef, file syntax.File, fileIndex int, info PackageInfo, checked []PackageInfo, scope FuncScope, start int, end int) []SelectorRef {
	for i := start + 1; i+1 < end && i+1 < len(file.Tokens); i++ {
		if !tokenTextIs(file, i, ".") {
			continue
		}
		if file.Tokens[i-1].Kind != syntax.TokenIdent || file.Tokens[i+1].Kind != syntax.TokenIdent {
			continue
		}
		if tokenTextIs(file, i-1, "_") || tokenTextIs(file, i+1, "_") {
			continue
		}
		selector := resolveImportSelectorCore(fileIndex, info, checked, scope, file, i-1, i, i+1)
		if selector.Kind == SelectorImport {
			selectors = append(selectors, selector)
		}
	}
	return selectors
}

func buildPackageTypeRefsCore(pkg load.Package, info PackageInfo, checked []PackageInfo) []TypeRef {
	var refs []TypeRef
	for i := 0; i < len(info.Decls); i++ {
		decl := info.Decls[i]
		if decl.TypeStart < 0 || decl.TypeEnd <= decl.TypeStart {
			continue
		}
		file := pkg.Files[decl.File].File
		if decl.Kind == SymbolType {
			typeIndex := LookupType(info, decl.Name)
			if typeIndex >= 0 {
				refs = appendTypeInfoRefsCore(refs, pkg, info, checked, info.Types[typeIndex], i)
				continue
			}
		}
		refs = appendDeclTypeSpanRefsCore(refs, file, decl.File, info, checked, FuncScope{}, i, decl.TypeStart, decl.TypeEnd)
	}
	return refs
}

func appendTypeInfoRefsCore(refs []TypeRef, pkg load.Package, info PackageInfo, checked []PackageInfo, typ TypeInfo, ownerDecl int) []TypeRef {
	file := pkg.Files[typ.File].File
	if typ.Kind == TypeStruct {
		for i := 0; i < len(typ.Fields); i++ {
			field := typ.Fields[i]
			refs = appendDeclTypeSpanRefsCore(refs, file, typ.File, info, checked, FuncScope{}, ownerDecl, field.TypeStart, field.TypeEnd)
		}
		return refs
	}
	if typ.Kind == TypeInterface {
		for i := 0; i < len(typ.InterfaceEmbeds); i++ {
			embed := typ.InterfaceEmbeds[i]
			refs = appendDeclTypeSpanRefsCore(refs, file, typ.File, info, checked, FuncScope{}, ownerDecl, embed.TypeStart, embed.TypeEnd)
		}
		for i := 0; i < len(typ.InterfaceMethods); i++ {
			base := len(refs)
			refs = appendSignatureTypeRefsCore(refs, file, typ.File, info, checked, FuncScope{}, typ.InterfaceMethods[i].Signature)
			markTypeRefOwnerDecl(refs, base, ownerDecl)
		}
		return refs
	}
	return appendDeclTypeSpanRefsCore(refs, file, typ.File, info, checked, FuncScope{}, ownerDecl, typ.TypeStart, typ.TypeEnd)
}

func buildFuncTypeRefsCore(file syntax.File, fileIndex int, info PackageInfo, checked []PackageInfo, signature FuncSignature, locals []LocalDeclInfo, scope FuncScope) []TypeRef {
	var refs []TypeRef
	refs = appendSignatureTypeRefsCore(refs, file, fileIndex, info, checked, scope, signature)
	for i := 0; i < len(locals); i++ {
		local := locals[i]
		if local.TypeStart >= 0 && local.TypeEnd > local.TypeStart {
			refs = appendTypeSpanRefsCore(refs, file, fileIndex, info, checked, scope, local.TypeStart, local.TypeEnd)
		}
	}
	return refs
}

func appendSignatureTypeRefsCore(refs []TypeRef, file syntax.File, fileIndex int, info PackageInfo, checked []PackageInfo, scope FuncScope, signature FuncSignature) []TypeRef {
	for i := 0; i < len(signature.Receiver); i++ {
		field := signature.Receiver[i]
		refs = appendTypeSpanRefsCore(refs, file, fileIndex, info, checked, scope, field.TypeStart, field.TypeEnd)
	}
	for i := 0; i < len(signature.Params); i++ {
		field := signature.Params[i]
		refs = appendTypeSpanRefsCore(refs, file, fileIndex, info, checked, scope, field.TypeStart, field.TypeEnd)
	}
	for i := 0; i < len(signature.Results); i++ {
		field := signature.Results[i]
		refs = appendTypeSpanRefsCore(refs, file, fileIndex, info, checked, scope, field.TypeStart, field.TypeEnd)
	}
	return refs
}

func appendDeclTypeSpanRefsCore(refs []TypeRef, file syntax.File, fileIndex int, info PackageInfo, checked []PackageInfo, scope FuncScope, ownerDecl int, start int, end int) []TypeRef {
	base := len(refs)
	refs = appendTypeSpanRefsCore(refs, file, fileIndex, info, checked, scope, start, end)
	markTypeRefOwnerDecl(refs, base, ownerDecl)
	return refs
}

func appendTypeSpanRefsCore(refs []TypeRef, file syntax.File, fileIndex int, info PackageInfo, checked []PackageInfo, scope FuncScope, start int, end int) []TypeRef {
	for i := start; i < end && i < len(file.Tokens); i++ {
		if file.Tokens[i].Kind != syntax.TokenIdent {
			continue
		}
		if i > start && tokenTextIs(file, i-1, ".") {
			continue
		}
		if tokenTextIs(file, i, "_") {
			continue
		}
		if i+2 < end && tokenTextIs(file, i+1, ".") && file.Tokens[i+2].Kind == syntax.TokenIdent {
			ref := resolveImportSelectorTypeRefCore(fileIndex, info, checked, scope, file, i, i+1, i+2)
			if ref.Kind == TypeRefImportSelector {
				refs = append(refs, ref)
			}
			i += 2
			continue
		}
		if lookupScopeTokenNameCore(scope, file, i) < 0 {
			symbol := lookupPackageSymbolTokenCore(info, file, fileIndex, i)
			if symbol >= 0 {
				refs = append(refs, TypeRef{Kind: TypeRefPackage, File: fileIndex, Token: i, BaseToken: i, DotToken: i, Package: info.Symbols[symbol].Package, Symbol: symbol})
			}
		}
	}
	return refs
}

func resolveImportSelectorCore(fileIndex int, info PackageInfo, checked []PackageInfo, scope FuncScope, file syntax.File, baseTok int, dotTok int, nameTok int) SelectorRef {
	selector := SelectorRef{
		Kind:      SelectorUnknown,
		BaseToken: baseTok,
		DotToken:  dotTok,
		NameToken: nameTok,
		BaseRef:   NameRef{Kind: RefUnknown, Token: baseTok, Index: -1, Package: -1},
		Package:   -1,
		Symbol:    -1,
	}
	scopeIndex := lookupScopeTokenNameCore(scope, file, baseTok)
	if scopeIndex >= 0 && scope.Names[scopeIndex].Kind != NameLabel {
		selector.BaseRef.Kind = RefScope
		selector.BaseRef.Index = scopeIndex
		return selector
	}
	importIndex := lookupImportTokenNameCore(info, fileIndex, file, baseTok)
	if importIndex < 0 || info.Imports[importIndex].Blank || info.Imports[importIndex].Dot {
		return selector
	}
	selector.BaseRef.Kind = RefImport
	selector.BaseRef.Index = importIndex
	selector.BaseRef.Package = info.Imports[importIndex].Package
	if selector.BaseRef.Package < 0 || selector.BaseRef.Package >= len(checked) {
		return selector
	}
	symbol := lookupPackageSymbolTextCore(checked[selector.BaseRef.Package], file, nameTok)
	if symbol < 0 {
		return selector
	}
	selector.Kind = SelectorImport
	selector.Package = selector.BaseRef.Package
	selector.Symbol = symbol
	return selector
}

func resolveImportSelectorTypeRefCore(fileIndex int, info PackageInfo, checked []PackageInfo, scope FuncScope, file syntax.File, baseTok int, dotTok int, nameTok int) TypeRef {
	ref := TypeRef{Kind: TypeRefUnknown, File: fileIndex, Token: nameTok, BaseToken: baseTok, DotToken: dotTok, Package: -1, Symbol: -1}
	scopeIndex := lookupScopeTokenNameCore(scope, file, baseTok)
	if scopeIndex >= 0 && scope.Names[scopeIndex].Kind != NameLabel {
		ref.Kind = TypeRefScope
		return ref
	}
	importIndex := lookupImportTokenNameCore(info, fileIndex, file, baseTok)
	if importIndex < 0 || info.Imports[importIndex].Blank || info.Imports[importIndex].Dot {
		return ref
	}
	pkg := info.Imports[importIndex].Package
	if pkg < 0 || pkg >= len(checked) {
		return ref
	}
	symbol := lookupPackageSymbolTextCore(checked[pkg], file, nameTok)
	if symbol < 0 {
		return ref
	}
	ref.Kind = TypeRefImportSelector
	ref.Package = pkg
	ref.Symbol = symbol
	return ref
}

func lookupScopeTokenNameCore(scope FuncScope, file syntax.File, tok int) int {
	if tok < 0 || tok >= len(file.Tokens) {
		return -1
	}
	token := file.Tokens[tok]
	size := token.End - token.Start
	if size < 0 || token.Start < 0 || token.End > len(file.Src) {
		return -1
	}
	for i := 0; i < len(scope.Names); i++ {
		name := scope.Names[i].Name
		if len(name) != size {
			continue
		}
		matches := true
		for j := 0; j < size; j++ {
			if file.Src[token.Start+j] != name[j] {
				matches = false
				break
			}
		}
		if matches {
			return i
		}
	}
	return -1
}

func lookupImportTokenNameCore(info PackageInfo, fileIndex int, file syntax.File, tok int) int {
	if tok < 0 || tok >= len(file.Tokens) {
		return -1
	}
	token := file.Tokens[tok]
	size := token.End - token.Start
	if size < 0 || token.Start < 0 || token.End > len(file.Src) {
		return -1
	}
	for i := 0; i < len(info.Imports); i++ {
		imp := info.Imports[i]
		if imp.File != fileIndex || len(imp.Name) != size {
			continue
		}
		matches := true
		for j := 0; j < size; j++ {
			if file.Src[token.Start+j] != imp.Name[j] {
				matches = false
				break
			}
		}
		if matches {
			return i
		}
	}
	return -1
}

func lookupPackageSymbolTokenCore(info PackageInfo, file syntax.File, fileIndex int, tok int) int {
	if tok < 0 || tok >= len(file.Tokens) {
		return -1
	}
	token := file.Tokens[tok]
	size := token.End - token.Start
	if size < 0 || token.Start < 0 || token.End > len(file.Src) {
		return -1
	}
	for i := 0; i < len(info.Symbols); i++ {
		symbol := info.Symbols[i]
		if symbol.Kind == SymbolMethod || len(symbol.Name) != size {
			continue
		}
		matches := true
		for j := 0; j < size; j++ {
			if file.Src[token.Start+j] != symbol.Name[j] {
				matches = false
				break
			}
		}
		if matches {
			return i
		}
	}
	return -1
}

func lookupPackageSymbolTextCore(info PackageInfo, file syntax.File, tok int) int {
	if tok < 0 || tok >= len(file.Tokens) {
		return -1
	}
	token := file.Tokens[tok]
	size := token.End - token.Start
	if size < 0 || token.Start < 0 || token.End > len(file.Src) {
		return -1
	}
	for i := 0; i < len(info.Symbols); i++ {
		symbol := info.Symbols[i]
		if symbol.Kind == SymbolMethod || len(symbol.Name) != size {
			continue
		}
		matches := true
		for j := 0; j < size; j++ {
			if file.Src[token.Start+j] != symbol.Name[j] {
				matches = false
				break
			}
		}
		if matches {
			return i
		}
	}
	return -1
}

func buildFuncScopeCore(file syntax.File, fn syntax.FuncDecl) (FuncScope, bool, int) {
	var scope FuncScope
	scope.Names = make([]ScopeName, 0, coreScopeCapacity(fn.BodyEnd-fn.BodyStart))
	if fn.ReceiverStart >= 0 {
		tok := receiverNameToken(file, fn)
		if tok >= 0 {
			if !addScopeName(&scope, tokenString(file, tok), NameReceiver, tok, true, false) {
				return scope, false, tok
			}
		}
	}
	if fn.ParamsStart >= 0 && fn.ParamsEnd > fn.ParamsStart {
		ok, tok := collectFieldNames(file, fn.ParamsStart+1, fn.ParamsEnd-1, NameParam, &scope)
		if !ok {
			return scope, false, tok
		}
	}
	if fn.ResultStart >= 0 && fn.ResultEnd > fn.ResultStart && tokCharIs(file, fn.ResultStart, '(') {
		end := fn.ResultEnd - 1
		if tokCharIs(file, end, ')') {
			ok, tok := collectFieldNames(file, fn.ResultStart+1, end, NameResult, &scope)
			if !ok {
				return scope, false, tok
			}
		}
	}
	start := fn.BodyStart + 1
	end := fn.BodyEnd - 1
	for i := start; i < end; i++ {
		kind := file.Tokens[i].Kind
		if kind == syntax.TokenConst || kind == syntax.TokenVar || kind == syntax.TokenType {
			i = collectCoreDeclScope(file, i, end, &scope)
			continue
		}
		if tokenTextIs(file, i, ":=") {
			collectCoreShortDeclScope(file, coreLHSStart(file, i, start), i, &scope)
			continue
		}
		if coreTokenLooksLikeLabel(file, i, start, end) {
			if !addScopeName(&scope, tokenString(file, i), NameLabel, i, true, true) {
				return scope, false, i
			}
		}
	}
	return scope, true, -1
}

func coreTokenLooksLikeLabel(file syntax.File, tok int, start int, end int) bool {
	if tok < start || tok+1 >= end || file.Tokens[tok].Kind != syntax.TokenIdent || !tokCharIs(file, tok+1, ':') || tokenTextIs(file, tok+1, ":=") {
		return false
	}
	if tok == start {
		return true
	}
	prev := tok - 1
	if tokCharIs(file, prev, '{') || tokCharIs(file, prev, ',') {
		return false
	}
	return file.Tokens[prev].Line != file.Tokens[tok].Line || tokCharIs(file, prev, ';') || tokCharIs(file, prev, '}')
}

func collectCoreDeclScope(file syntax.File, start int, end int, scope *FuncScope) int {
	specStart := start + 1
	if specStart < end && tokCharIs(file, specStart, '(') {
		closeTok := findTypeMatching(file, specStart, '(', ')')
		if closeTok <= specStart || closeTok > end {
			return start
		}
		i := specStart + 1
		for i < closeTok-1 {
			i = skipLocalSeparators(file, i, closeTok-1)
			if i >= closeTok-1 {
				break
			}
			specEnd := statementSpecEnd(file, i, closeTok-1)
			collectLeadingIdentList(file, i, specEnd, scope)
			if specEnd <= i {
				i++
			} else {
				i = specEnd
			}
		}
		return closeTok
	}
	specEnd := statementSpecEnd(file, specStart, end)
	collectLeadingIdentList(file, specStart, specEnd, scope)
	return specEnd
}

func collectCoreShortDeclScope(file syntax.File, start int, end int, scope *FuncScope) {
	for i := start; i < end; i++ {
		if file.Tokens[i].Kind == syntax.TokenIdent {
			name := tokenString(file, i)
			if name != "_" && LookupScopeName(*scope, name) < 0 {
				addScopeName(scope, name, NameLocal, i, false, false)
			}
		}
	}
}

func coreLHSStart(file syntax.File, assign int, limit int) int {
	start := assign - 1
	for start > limit {
		if tokCharIs(file, start, ';') || tokCharIs(file, start, '{') || tokCharIs(file, start, '}') {
			return start + 1
		}
		if file.Tokens[start].Line != file.Tokens[assign].Line {
			return start + 1
		}
		start--
	}
	return start
}

func buildFuncLocalTypeSpansCore(file syntax.File, fn syntax.FuncDecl) []LocalDeclInfo {
	decls := make([]LocalDeclInfo, 0, coreLocalTypeCapacity(fn.BodyEnd-fn.BodyStart))
	start := fn.BodyStart + 1
	end := fn.BodyEnd - 1
	for i := start; i < end; i++ {
		kind := file.Tokens[i].Kind
		if kind != syntax.TokenConst && kind != syntax.TokenVar && kind != syntax.TokenType {
			continue
		}
		specStart := i + 1
		if specStart < end && tokCharIs(file, specStart, '(') {
			closeTok := findTypeMatching(file, specStart, '(', ')')
			if closeTok <= specStart || closeTok > end {
				continue
			}
			j := specStart + 1
			for j < closeTok-1 {
				j = skipLocalSeparators(file, j, closeTok-1)
				if j >= closeTok-1 {
					break
				}
				specEnd := statementSpecEnd(file, j, closeTok-1)
				decls = appendLocalTypeSpanCore(decls, file, declSymbolKind(kind), j, specEnd)
				if specEnd <= j {
					j++
				} else {
					j = specEnd
				}
			}
			i = closeTok
			continue
		}
		specEnd := statementSpecEnd(file, specStart, end)
		decls = appendLocalTypeSpanCore(decls, file, declSymbolKind(kind), specStart, specEnd)
		i = specEnd
	}
	return decls
}

func coreRefCapacity(tokens int) int {
	if tokens <= 0 {
		return 0
	}
	capacity := tokens / 4
	if capacity < 4 {
		return 4
	}
	return capacity
}

func coreSelectorCapacity(tokens int) int {
	if tokens <= 0 {
		return 0
	}
	capacity := tokens / 16
	if capacity < 2 {
		return 2
	}
	return capacity
}

func coreScopeCapacity(tokens int) int {
	if tokens <= 0 {
		return 4
	}
	capacity := tokens / 32
	if capacity < 4 {
		return 4
	}
	return capacity
}

func coreLocalTypeCapacity(tokens int) int {
	if tokens <= 0 {
		return 0
	}
	capacity := tokens / 64
	if capacity < 2 {
		return 2
	}
	return capacity
}

func appendLocalTypeSpanCore(decls []LocalDeclInfo, file syntax.File, kind int, start int, end int) []LocalDeclInfo {
	start, end = trimDeclSpan(file, start, end)
	if start < 0 || end <= start || start >= len(file.Tokens) || file.Tokens[start].Kind != syntax.TokenIdent {
		return decls
	}
	typeStart := -1
	typeEnd := -1
	if kind == SymbolType {
		typeStart = start + 1
		if tokenTextIs(file, typeStart, "=") {
			typeStart++
		}
		typeStart, typeEnd = trimDeclSpan(file, typeStart, end)
	} else {
		_, namesEnd := localDeclNameTokens(file, start, end)
		valueStart := findDeclAssign(file, namesEnd, end)
		if valueStart >= 0 {
			typeStart, typeEnd = trimDeclSpan(file, namesEnd, valueStart)
		} else {
			typeStart, typeEnd = trimDeclSpan(file, namesEnd, end)
		}
	}
	if typeStart < 0 || typeEnd <= typeStart {
		return decls
	}
	return append(decls, LocalDeclInfo{TypeStart: typeStart, TypeEnd: typeEnd})
}

func coreFuncKind(fn syntax.FuncDecl) int {
	if fn.ReceiverStart >= 0 {
		return SymbolMethod
	}
	return SymbolFunc
}
