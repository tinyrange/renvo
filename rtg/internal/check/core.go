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
	info.CoreTypeRefs = buildPackageTypeRefsCore(pkg, info, checked)
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
			bodyTokens := bodyEnd - bodyStart
			out.CoreRefs = make([]CoreNameRef, 0, coreRefCapacity(bodyTokens))
			out.CoreSelectors = make([]CoreSelectorRef, 0, coreSelectorCapacity(bodyTokens))
			out.CoreRefs = appendExprPackageRefsCore(out.CoreRefs, file, fileIndex, info, scope, bodyStart, bodyEnd)
			out.CoreSelectors = appendImportSelectorsCore(out.CoreSelectors, file, fileIndex, info, checked, scope, bodyStart, bodyEnd)
			locals := buildFuncLocalTypeSpansCore(file, fn)
			out.CoreTypeRefs = buildFuncTypeRefsCore(file, fileIndex, info, checked, signature, locals, scope)
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
		out.CoreRefs = make([]CoreNameRef, 0, coreRefCapacity(valueTokens))
		out.CoreSelectors = make([]CoreSelectorRef, 0, coreSelectorCapacity(valueTokens))
		out.CoreRefs = appendExprPackageRefsCore(out.CoreRefs, file, fileIndex, info, CoreScope{}, out.ValueStart, out.ValueEnd)
		out.CoreSelectors = appendImportSelectorsCore(out.CoreSelectors, file, fileIndex, info, checked, CoreScope{}, out.ValueStart, out.ValueEnd)
	} else {
		out.TypeStart, out.TypeEnd = trimDeclSpan(file, typeStart, decl.EndTok)
	}
	return out
}

type CoreScope struct {
	Names []CoreScopeName
}

type CoreScopeName struct {
	Kind  int
	Token int
}

func appendExprPackageRefsCore(refs []CoreNameRef, file syntax.File, fileIndex int, info PackageInfo, scope CoreScope, start int, end int) []CoreNameRef {
	for i := start; i < end && i < len(file.Tokens); i++ {
		if file.Tokens[i].Kind != syntax.TokenIdent || shouldSkipIdentRef(file, i, end) {
			continue
		}
		if tokenTextIs(file, i, "_") || lookupScopeTokenNameCore(scope, file, i) >= 0 || lookupImportTokenNameCore(info, fileIndex, file, i) >= 0 {
			continue
		}
		symbolIndex := lookupPackageSymbolTokenCore(info, file, fileIndex, i)
		if symbolIndex >= 0 {
			refs = append(refs, CoreNameRef{Kind: RefPackage, Token: i, Index: symbolIndex, Package: info.Symbols[symbolIndex].Package})
		}
	}
	return refs
}

func appendImportSelectorsCore(selectors []CoreSelectorRef, file syntax.File, fileIndex int, info PackageInfo, checked []PackageInfo, scope CoreScope, start int, end int) []CoreSelectorRef {
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

func buildPackageTypeRefsCore(pkg load.Package, info PackageInfo, checked []PackageInfo) []CoreTypeRef {
	var refs []CoreTypeRef
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
		refs = appendDeclTypeSpanRefsCore(refs, file, decl.File, info, checked, CoreScope{}, i, decl.TypeStart, decl.TypeEnd)
	}
	return refs
}

func appendTypeInfoRefsCore(refs []CoreTypeRef, pkg load.Package, info PackageInfo, checked []PackageInfo, typ TypeInfo, ownerDecl int) []CoreTypeRef {
	file := pkg.Files[typ.File].File
	if typ.Kind == TypeStruct {
		for i := 0; i < len(typ.Fields); i++ {
			field := typ.Fields[i]
			refs = appendDeclTypeSpanRefsCore(refs, file, typ.File, info, checked, CoreScope{}, ownerDecl, field.TypeStart, field.TypeEnd)
		}
		return refs
	}
	if typ.Kind == TypeInterface {
		for i := 0; i < len(typ.InterfaceEmbeds); i++ {
			embed := typ.InterfaceEmbeds[i]
			refs = appendDeclTypeSpanRefsCore(refs, file, typ.File, info, checked, CoreScope{}, ownerDecl, embed.TypeStart, embed.TypeEnd)
		}
		for i := 0; i < len(typ.InterfaceMethods); i++ {
			base := len(refs)
			refs = appendSignatureTypeRefsCore(refs, file, typ.File, info, checked, CoreScope{}, typ.InterfaceMethods[i].Signature)
			markCoreTypeRefOwnerDecl(refs, base, ownerDecl)
		}
		return refs
	}
	return appendDeclTypeSpanRefsCore(refs, file, typ.File, info, checked, CoreScope{}, ownerDecl, typ.TypeStart, typ.TypeEnd)
}

type CoreLocalTypeSpan struct {
	TypeStart int
	TypeEnd   int
}

func buildFuncTypeRefsCore(file syntax.File, fileIndex int, info PackageInfo, checked []PackageInfo, signature FuncSignature, locals []CoreLocalTypeSpan, scope CoreScope) []CoreTypeRef {
	var refs []CoreTypeRef
	refs = appendSignatureTypeRefsCore(refs, file, fileIndex, info, checked, scope, signature)
	for i := 0; i < len(locals); i++ {
		local := locals[i]
		if local.TypeStart >= 0 && local.TypeEnd > local.TypeStart {
			refs = appendTypeSpanRefsCore(refs, file, fileIndex, info, checked, scope, local.TypeStart, local.TypeEnd)
		}
	}
	return refs
}

func appendSignatureTypeRefsCore(refs []CoreTypeRef, file syntax.File, fileIndex int, info PackageInfo, checked []PackageInfo, scope CoreScope, signature FuncSignature) []CoreTypeRef {
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

func appendDeclTypeSpanRefsCore(refs []CoreTypeRef, file syntax.File, fileIndex int, info PackageInfo, checked []PackageInfo, scope CoreScope, ownerDecl int, start int, end int) []CoreTypeRef {
	base := len(refs)
	refs = appendTypeSpanRefsCore(refs, file, fileIndex, info, checked, scope, start, end)
	markCoreTypeRefOwnerDecl(refs, base, ownerDecl)
	return refs
}

func markCoreTypeRefOwnerDecl(refs []CoreTypeRef, start int, ownerDecl int) {
	for i := start; i < len(refs); i++ {
		refs[i].OwnerDecl = ownerDecl
	}
}

func appendTypeSpanRefsCore(refs []CoreTypeRef, file syntax.File, fileIndex int, info PackageInfo, checked []PackageInfo, scope CoreScope, start int, end int) []CoreTypeRef {
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
				refs = append(refs, CoreTypeRef{Kind: TypeRefPackage, File: fileIndex, Token: i, BaseTok: i, DotTok: i, Package: info.Symbols[symbol].Package, Symbol: symbol})
			}
		}
	}
	return refs
}

func resolveImportSelectorCore(fileIndex int, info PackageInfo, checked []PackageInfo, scope CoreScope, file syntax.File, baseTok int, dotTok int, nameTok int) CoreSelectorRef {
	selector := CoreSelectorRef{
		Kind:      SelectorUnknown,
		BaseTok:   baseTok,
		DotTok:    dotTok,
		NameTok:   nameTok,
		BaseKind:  RefUnknown,
		BaseIndex: -1,
		Package:   -1,
		Symbol:    -1,
	}
	scopeIndex := lookupScopeTokenNameCore(scope, file, baseTok)
	if scopeIndex >= 0 && scope.Names[scopeIndex].Kind != NameLabel {
		selector.BaseKind = RefScope
		selector.BaseIndex = scopeIndex
		return selector
	}
	importIndex := lookupImportTokenNameCore(info, fileIndex, file, baseTok)
	if importIndex < 0 || info.Imports[importIndex].Blank || info.Imports[importIndex].Dot {
		return selector
	}
	selector.BaseKind = RefImport
	selector.BaseIndex = importIndex
	selector.BasePackage = info.Imports[importIndex].Package
	if selector.BasePackage < 0 || selector.BasePackage >= len(checked) {
		return selector
	}
	symbol := lookupPackageSymbolTextCore(checked[selector.BasePackage], file, nameTok)
	if symbol < 0 {
		return selector
	}
	selector.Kind = SelectorImport
	selector.Package = selector.BasePackage
	selector.Symbol = symbol
	return selector
}

func resolveImportSelectorTypeRefCore(fileIndex int, info PackageInfo, checked []PackageInfo, scope CoreScope, file syntax.File, baseTok int, dotTok int, nameTok int) CoreTypeRef {
	ref := CoreTypeRef{Kind: TypeRefUnknown, File: fileIndex, Token: nameTok, BaseTok: baseTok, DotTok: dotTok, Package: -1, Symbol: -1}
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

func lookupScopeTokenNameCore(scope CoreScope, file syntax.File, tok int) int {
	if tok < 0 || tok >= len(file.Tokens) {
		return -1
	}
	token := file.Tokens[tok]
	size := token.End - token.Start
	if size < 0 || token.Start < 0 || token.End > len(file.Src) {
		return -1
	}
	for i := 0; i < len(scope.Names); i++ {
		nameTok := scope.Names[i].Token
		if nameTok < 0 || nameTok >= len(file.Tokens) {
			continue
		}
		name := file.Tokens[nameTok]
		if name.End-name.Start != size {
			continue
		}
		matches := true
		for j := 0; j < size; j++ {
			if file.Src[token.Start+j] != file.Src[name.Start+j] {
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

func buildFuncScopeCore(file syntax.File, fn syntax.FuncDecl) (CoreScope, bool, int) {
	var scope CoreScope
	scope.Names = make([]CoreScopeName, 0, coreScopeCapacity(fn.BodyEnd-fn.BodyStart))
	if fn.ReceiverStart >= 0 {
		tok := receiverNameToken(file, fn)
		if tok >= 0 {
			if !addCoreScopeName(&scope, file, tok, NameReceiver, true, false) {
				return scope, false, tok
			}
		}
	}
	if fn.ParamsStart >= 0 && fn.ParamsEnd > fn.ParamsStart {
		ok, tok := collectCoreFieldNames(file, fn.ParamsStart+1, fn.ParamsEnd-1, NameParam, &scope)
		if !ok {
			return scope, false, tok
		}
	}
	if fn.ResultStart >= 0 && fn.ResultEnd > fn.ResultStart && tokCharIs(file, fn.ResultStart, '(') {
		end := fn.ResultEnd - 1
		if tokCharIs(file, end, ')') {
			ok, tok := collectCoreFieldNames(file, fn.ResultStart+1, end, NameResult, &scope)
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
			if !addCoreScopeName(&scope, file, i, NameLabel, true, true) {
				return scope, false, i
			}
		}
	}
	return scope, true, -1
}

func collectCoreFieldNames(file syntax.File, start int, end int, kind int, scope *CoreScope) (bool, int) {
	pending := make([]int, 0, 2)
	i := start
	for i < end {
		segStart := i
		segEnd := nextTopLevelComma(file, i, end)
		first := firstNonSeparator(file, segStart, segEnd)
		if first < segEnd && file.Tokens[first].Kind == syntax.TokenIdent {
			next := first + 1
			if next >= segEnd {
				pending = append(pending, first)
			} else if tokCharIs(file, next, '.') {
				pending = pending[:0]
			} else {
				if !addCorePendingNames(file, pending, kind, scope) {
					return false, pending[0]
				}
				pending = pending[:0]
				if !addCoreScopeName(scope, file, first, kind, true, false) {
					return false, first
				}
			}
		} else {
			pending = pending[:0]
		}
		i = segEnd + 1
	}
	return true, -1
}

func addCorePendingNames(file syntax.File, pending []int, kind int, scope *CoreScope) bool {
	for i := 0; i < len(pending); i++ {
		if !addCoreScopeName(scope, file, pending[i], kind, true, false) {
			return false
		}
	}
	return true
}

func collectCoreLeadingIdentList(file syntax.File, start int, end int, scope *CoreScope) {
	i := start
	for i < end {
		if file.Tokens[i].Kind != syntax.TokenIdent {
			return
		}
		if !tokenTextIs(file, i, "_") && lookupScopeTokenNameCore(*scope, file, i) < 0 {
			addCoreScopeName(scope, file, i, NameLocal, false, false)
		}
		i++
		if i < end && tokCharIs(file, i, ',') {
			i++
			continue
		}
		return
	}
}

func addCoreScopeName(scope *CoreScope, file syntax.File, tok int, kind int, rejectDup bool, labelsOnly bool) bool {
	if tok < 0 || tok >= len(file.Tokens) || tokenTextIs(file, tok, "_") {
		return true
	}
	if rejectDup {
		for i := 0; i < len(scope.Names); i++ {
			if !coreTokensEqual(file, scope.Names[i].Token, tok) {
				continue
			}
			if labelsOnly {
				if scope.Names[i].Kind == NameLabel {
					return false
				}
				continue
			}
			if scope.Names[i].Kind != NameLabel {
				return false
			}
		}
	}
	scope.Names = append(scope.Names, CoreScopeName{Kind: kind, Token: tok})
	return true
}

func coreTokensEqual(file syntax.File, left int, right int) bool {
	if left < 0 || left >= len(file.Tokens) || right < 0 || right >= len(file.Tokens) {
		return false
	}
	leftTok := file.Tokens[left]
	rightTok := file.Tokens[right]
	size := leftTok.End - leftTok.Start
	if size < 0 || rightTok.End-rightTok.Start != size || leftTok.Start < 0 || rightTok.Start < 0 || leftTok.End > len(file.Src) || rightTok.End > len(file.Src) {
		return false
	}
	for i := 0; i < size; i++ {
		if file.Src[leftTok.Start+i] != file.Src[rightTok.Start+i] {
			return false
		}
	}
	return true
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

func collectCoreDeclScope(file syntax.File, start int, end int, scope *CoreScope) int {
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
			collectCoreLeadingIdentList(file, i, specEnd, scope)
			if specEnd <= i {
				i++
			} else {
				i = specEnd
			}
		}
		return closeTok
	}
	specEnd := statementSpecEnd(file, specStart, end)
	collectCoreLeadingIdentList(file, specStart, specEnd, scope)
	return specEnd
}

func collectCoreShortDeclScope(file syntax.File, start int, end int, scope *CoreScope) {
	for i := start; i < end; i++ {
		if file.Tokens[i].Kind == syntax.TokenIdent {
			if !tokenTextIs(file, i, "_") && lookupScopeTokenNameCore(*scope, file, i) < 0 {
				addCoreScopeName(scope, file, i, NameLocal, false, false)
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

func buildFuncLocalTypeSpansCore(file syntax.File, fn syntax.FuncDecl) []CoreLocalTypeSpan {
	decls := make([]CoreLocalTypeSpan, 0, coreLocalTypeCapacity(fn.BodyEnd-fn.BodyStart))
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

func appendLocalTypeSpanCore(decls []CoreLocalTypeSpan, file syntax.File, kind int, start int, end int) []CoreLocalTypeSpan {
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
	return append(decls, CoreLocalTypeSpan{TypeStart: typeStart, TypeEnd: typeEnd})
}

func coreFuncKind(fn syntax.FuncDecl) int {
	if fn.ReceiverStart >= 0 {
		return SymbolMethod
	}
	return SymbolFunc
}
