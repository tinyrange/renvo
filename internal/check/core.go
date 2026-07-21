package check

import (
	"renvo.dev/internal/arena"
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

func CheckGraphCore(graph load.Graph) Program {
	prog := CheckGraphHeadersCore(graph)
	if !prog.Ok {
		return prog
	}
	for i := 0; i < len(graph.Packages); i++ {
		prog = CheckGraphPackageCore(graph, prog, i)
		if !prog.Ok {
			return prog
		}
	}
	return prog
}

// CheckGraphHeadersCore resolves the package surface needed while checking
// every package body. Keeping this phase separate lets the Renvo compiler lower
// and release one body at a time.
func CheckGraphHeadersCore(graph load.Graph) Program {
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
		info.CoreSymbolHash = buildCoreSymbolHash(info.Symbols)
		prog.Packages = append(prog.Packages, info)
		if !ok {
			return checkFail(prog, err, i, file, tok)
		}
	}
	return prog
}

// CheckGraphPackageCore checks one package body against the complete set of
// package headers. The returned arena range becomes dead after that package is
// lowered.
func CheckGraphPackageCore(graph load.Graph, prog Program, pkgIndex int) Program {
	if !prog.Ok || pkgIndex < 0 || pkgIndex >= len(graph.Packages) || pkgIndex >= len(prog.Packages) {
		return checkFail(prog, CheckErrGraph, pkgIndex, -1, -1)
	}
	prog.Packages[pkgIndex].CoreArenaStart = arena.Mark()
	info, ok, err, file, tok := checkPackageBodyCore(graph, pkgIndex, prog.Packages[pkgIndex], prog.Packages)
	info.CoreArenaStart = prog.Packages[pkgIndex].CoreArenaStart
	info.CoreArenaEnd = arena.Mark()
	prog.Packages[pkgIndex] = info
	if !ok {
		return checkFail(prog, err, pkgIndex, file, tok)
	}
	return prog
}

func checkPackageBodyCore(graph load.Graph, pkgIndex int, info PackageInfo, checked []PackageInfo) (PackageInfo, bool, int, int, int) {
	pkg := graph.Packages[pkgIndex]
	info.Decls = make([]DeclInfo, 0, countPackageDeclsCore(pkg))
	info.CoreBodies = make([]CoreFuncBody, 0, countPackageFuncsCore(pkg))
	for fileIndex := 0; fileIndex < len(pkg.Files); fileIndex++ {
		file := pkg.Files[fileIndex].File
		for i := 0; i < len(file.Decls); i++ {
			decl, undefinedTok := buildDeclInfoCore(file, fileIndex, info, checked, file.Decls[i])
			info.Decls = append(info.Decls, decl)
			if undefinedTok >= 0 {
				return info, false, CheckErrUndefined, fileIndex, undefinedTok
			}
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
			typ := buildTypeInfo(file, decl, i)
			if duplicateTok := duplicateStructFieldToken(typ); duplicateTok >= 0 {
				return info, false, CheckErrDuplicate, decl.File, duplicateTok
			}
			info.Types = append(info.Types, typ)
		}
	}
	sortTypes(info.Types)
	if file, tok := undefinedSimplePackageTypeCore(pkg, info); tok >= 0 {
		return info, false, CheckErrUndefined, file, tok
	}
	info.CoreTypeRefs = buildPackageTypeRefsCore(pkg, info, checked)
	callTargets := make([]definiteCallTarget, len(info.Symbols))
	for fileIndex := 0; fileIndex < len(pkg.Files); fileIndex++ {
		file := pkg.Files[fileIndex].File
		for i := 0; i < len(file.Funcs); i++ {
			fn := file.Funcs[i]
			functionArenaStart := arena.Mark()
			body := syntax.ParseFuncBodyStatements(file, fn)
			if !body.Ok {
				arena.Reset(functionArenaStart)
				return info, false, CheckErrBody, fileIndex, body.ErrorTok
			}
			if statementErr, statementTok := invalidDefiniteStatement(file, body); statementErr != CheckOK {
				arena.Reset(functionArenaStart)
				return info, false, statementErr, fileIndex, statementTok
			}
			if indexTok := invalidConstantArrayIndex(&pkg, &info, fileIndex, fn, &body); indexTok >= 0 {
				arena.Reset(functionArenaStart)
				return info, false, CheckErrArrayIndex, fileIndex, indexTok
			}
			arena.Reset(functionArenaStart)
			signature := buildFuncSignature(file, fn)
			if returnErr, returnTok := invalidReturnCount(file, fn, signature); returnErr != CheckOK {
				return info, false, returnErr, fileIndex, returnTok
			}
			if assignmentErr, assignmentTok := invalidDefiniteAssignmentType(file, fn); assignmentErr != CheckOK {
				return info, false, assignmentErr, fileIndex, assignmentTok
			}
			if sliceTok := invalidDefiniteSliceOperand(pkg, info, fileIndex, fn); sliceTok >= 0 {
				return info, false, CheckErrSliceOperand, fileIndex, sliceTok
			}
			scope, ok, scopeTok := buildFuncScopeCore(file, fn)
			if !ok {
				return info, false, CheckErrScope, fileIndex, scopeTok
			}
			bodyStart := fn.BodyStart + 1
			bodyEnd := fn.BodyEnd - 1
			var out CoreFuncBody
			out.Kind = coreFuncKind(fn)
			out.File = fileIndex
			out.Func = i
			out.ErrorToken = fn.NameTok
			refCount, selectorCount := resolutionCapacitiesCore(bodyEnd - bodyStart)
			out.CoreRefs = make([]CoreNameRef, 0, refCount)
			out.CoreSelectors = make([]CoreSelectorRef, 0, selectorCount)
			var builtinCalls []int
			var undefinedTok int
			out.CoreRefs, out.CoreSelectors, undefinedTok = appendResolutionRefsCore(out.CoreRefs, out.CoreSelectors, &file, fileIndex, &info, checked, scope, bodyStart, bodyEnd, &builtinCalls)
			if undefinedTok >= 0 {
				return info, false, CheckErrUndefined, fileIndex, undefinedTok
			}
			if builtinErr, builtinTok := invalidBuiltinCalls(&pkg, &info, fileIndex, fn, &signature, builtinCalls); builtinErr != CheckOK {
				return info, false, builtinErr, fileIndex, builtinTok
			}
			prepareDefiniteCallTargets(&pkg, &info, out.CoreRefs, callTargets)
			callCheckArenaStart := arena.Mark()
			callTypeTok := invalidDefiniteCallArgumentType(&pkg, &info, fileIndex, fn, &signature, out.CoreRefs, callTargets)
			arena.Reset(callCheckArenaStart)
			if callTypeTok >= 0 {
				return info, false, CheckErrCallArgument, fileIndex, callTypeTok
			}
			if unusedTok := unusedCoreLocalToken(scope); unusedTok >= 0 {
				return info, false, CheckErrUnusedLocal, fileIndex, unusedTok
			}
			locals := buildFuncLocalTypeSpansCore(file, fn)
			out.CoreTypeRefs = buildFuncTypeRefsCore(file, fileIndex, info, checked, signature, locals, scope)
			out.CoreRefs = renvo_runtime_ArenaPersistCheckNameRefs(out.CoreRefs)
			out.CoreSelectors = renvo_runtime_ArenaPersistCheckSelectorRefs(out.CoreSelectors)
			out.CoreTypeRefs = renvo_runtime_ArenaPersistCheckTypeRefs(out.CoreTypeRefs)
			arena.Reset(functionArenaStart)
			info.CoreBodies = append(info.CoreBodies, out)
		}
	}
	for i := 0; i < len(info.Imports); i++ {
		imp := info.Imports[i]
		if !imp.Blank && !imp.Dot && !imp.Used {
			return info, false, CheckErrUnusedImport, imp.File, imp.Token
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

func undefinedSimplePackageTypeCore(pkg load.Package, info PackageInfo) (int, int) {
	for i := 0; i < len(info.Decls); i++ {
		decl := info.Decls[i]
		if decl.TypeEnd-decl.TypeStart != 1 {
			continue
		}
		file := &pkg.Files[decl.File].File
		if file.Tokens[decl.TypeStart].KindLine&255 == syntax.TokenIdent && !corePredeclaredToken(file, decl.TypeStart) && lookupPackageSymbolTokenCore(&info, file, decl.File, decl.TypeStart) < 0 && !hasDotImportCore(&info, decl.File) {
			return decl.File, decl.TypeStart
		}
	}
	return -1, -1
}

func buildDeclInfoCore(file syntax.File, fileIndex int, info PackageInfo, checked []PackageInfo, decl syntax.TopDecl) (DeclInfo, int) {
	name := tokenString(&file, decl.NameTok)
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
		if tokenTextIs(&file, typeStart, "=") {
			out.Alias = true
			typeStart++
		}
		out.TypeStart, out.TypeEnd = trimDeclSpan(file, typeStart, decl.EndTok)
		return out, -1
	}
	typeStart := declNameListEnd(file, decl)
	valueStart := findDeclAssign(file, typeStart, decl.EndTok)
	if valueStart >= 0 {
		out.TypeStart, out.TypeEnd = trimDeclSpan(file, typeStart, valueStart)
		out.ValueStart, out.ValueEnd = trimDeclSpan(file, valueStart+1, decl.EndTok)
		refCount, selectorCount := resolutionCapacitiesCore(out.ValueEnd - out.ValueStart)
		out.CoreRefs = make([]CoreNameRef, 0, refCount)
		out.CoreSelectors = make([]CoreSelectorRef, 0, selectorCount)
		var undefinedTok int
		out.CoreRefs, out.CoreSelectors, undefinedTok = appendResolutionRefsCore(out.CoreRefs, out.CoreSelectors, &file, fileIndex, &info, checked, CoreScope{}, out.ValueStart, out.ValueEnd, nil)
		return out, undefinedTok
	} else {
		out.TypeStart, out.TypeEnd = trimDeclSpan(file, typeStart, decl.EndTok)
	}
	return out, -1
}

type CoreScope struct {
	Names []CoreScopeName
}

type CoreScopeName struct {
	Kind  int
	Token int
	Hash  int
}

func resolutionCapacitiesCore(tokens int) (int, int) {
	if tokens < 0 {
		tokens = 0
	}
	// Package references are usually fewer than one per fourteen tokens. Import
	// selectors are rarer still, so let that slice grow only in packages that
	// actually use one. This avoids reserving thousands of empty selector rows
	// while self-hosting the import-free backend package.
	return tokens/14 + 4, 0
}

func appendResolutionRefsCore(refs []CoreNameRef, selectors []CoreSelectorRef, file *syntax.File, fileIndex int, info *PackageInfo, checked []PackageInfo, scope CoreScope, start int, end int, builtinCalls *[]int) ([]CoreNameRef, []CoreSelectorRef, int) {
	undefined := -1
	for i := start; i < end && i < len(file.Tokens); i++ {
		token := file.Tokens[i]
		blank := token.KindLine&255 == syntax.TokenIdent && token.End == token.Start+1 && file.Src[token.Start] == '_'
		scopeIndex := -1
		skipRef := token.KindLine&255 != syntax.TokenIdent || blank || shouldSkipIdentRef(file, i, end)
		if !skipRef {
			scopeIndex = lookupScopeTokenNameCore(scope, file, i)
		} else if token.KindLine&255 == syntax.TokenIdent && !blank && i+1 < end && tokenTextIs(file, i+1, ":") {
			// A leading identifier in a keyed map literal is an expression even
			// though the same token shape denotes a field name in a struct literal.
			scopeIndex = lookupScopeTokenNameCore(scope, file, i)
		}
		if scopeIndex >= 0 && scope.Names[scopeIndex].Kind == NameVariable && i != scope.Names[scopeIndex].Token && !coreLocalWriteOnly(file, i, end) {
			scope.Names[scopeIndex].Kind = NameVariableUsed
		}
		if !skipRef && scopeIndex < 0 && lookupImportTokenNameCore(info, fileIndex, file, i) < 0 {
			symbolIndex := lookupPackageSymbolTokenCore(info, file, fileIndex, i)
			if symbolIndex >= 0 {
				refs = append(refs, CoreNameRef{Token: i, Index: symbolIndex})
			} else if corePredeclaredToken(file, i) {
				if builtinCalls != nil && i+1 < end && tokCharIs(file, i+1, '(') && coreOrdinaryBuiltinToken(file, i) {
					*builtinCalls = append(*builtinCalls, i)
				}
			} else if !hasDotImportCore(info, fileIndex) && undefined < 0 {
				undefined = i
			}
		}
		dot := token.KindLine&255 == syntax.TokenOperator && token.End == token.Start+1 && file.Src[token.Start] == '.'
		if i > start && i+1 < end && i+1 < len(file.Tokens) && dot &&
			file.Tokens[i-1].KindLine&255 == syntax.TokenIdent && file.Tokens[i+1].KindLine&255 == syntax.TokenIdent &&
			!(i >= start+2 && tokenTextIs(file, i-2, ".")) &&
			!(file.Tokens[i-1].End == file.Tokens[i-1].Start+1 && file.Src[file.Tokens[i-1].Start] == '_') &&
			!(file.Tokens[i+1].End == file.Tokens[i+1].Start+1 && file.Src[file.Tokens[i+1].Start] == '_') {
			selector := resolveImportSelectorCore(fileIndex, info, checked, scope, file, i-1, i, i+1)
			if selector.Symbol >= 0 {
				selectors = append(selectors, selector)
			} else if selector.BasePackage >= 0 && !coreSelectorContinues(file, i+1, end) && !coreUnsafeSelector(info, fileIndex, file, i-1) && undefined < 0 {
				undefined = i + 1
			}
		}
	}
	return refs, selectors, undefined
}

func corePredeclaredToken(file *syntax.File, tok int) bool {
	token := file.Tokens[tok]
	// These are the Go predeclared identifiers plus Renvo's syscall surface,
	// ordered by hash. Packing the hashes, offsets, and spelling table into
	// constants keeps this hot lookup allocation-free in a self-hosted compiler;
	// the final spelling comparison makes hash collisions harmless.
	const names = "anycapintlenmaxminnewnilchmodclearcloseerrorfalseint16int32int64panicprintuint8complex128writeO_CREATEstringuint16uint32uint64O_TRUNCuintptrfloat32float64O_RDONLYrecovercomplex64printlnO_WRONLYO_RDWRcomplexappenddeleteboolbytecopyimagint8iotamakeopenreadrealrunetrueuint"
	const hashes = "\x2d\x5e\x88\x0b\xf9\x64\x88\x0b\x30\x80\x88\x0b\xc4\x8b\x88\x0b\x8b\x8f\x88\x0b\x89\x90\x88\x0b\x4f\x94\x88\x0b\xc8\x94\x88\x0b\xd0\x5f\x39\x0f\x8c\x6d\x3b\x0f\x5b\x9a\x3b\x0f\xef\x21\x63\x0f\xf0\xce\x6b\x0f\xb7\x52\xa9\x0f\xf5\x52\xa9\x0f\x5a\x53\xa9\x0f\x30\xcb\x20\x10\x12\x09\x2a\x10\xfd\xa9\x7f\x10\x18\xaa\x8e\x10\x50\xb5\xa8\x10\x47\xf8\x1c\x17\xfc\xaf\x93\x1c\xec\xe8\x74\x20\x2a\xe9\x74\x20\x8f\xe9\x74\x20\xff\xe5\xb0\x28\xdb\x1a\x13\x2f\x20\xbb\x70\x33\x85\xbb\x70\x33\xcb\xb4\x22\x39\x3b\x0e\x06\x3e\x87\xee\x70\x42\xec\xa3\xd0\x42\xfe\xd6\xc9\x45\x32\xca\x0b\x47\x1d\x6c\x65\x53\xdd\x48\x4d\x72\x78\x84\x83\x78\x91\xb3\x94\x7c\xb9\xde\x94\x7c\x20\x40\x95\x7c\xe3\x7f\x98\x7c\x68\x86\x98\x7c\xd2\x8a\x98\x7c\xa3\x7f\x9a\x7c\x77\xd7\x9b\x7c\x41\x4d\x9d\x7c\x49\x4d\x9d\x7c\xff\x92\x9d\x7c\xe5\x9f\x9e\x7c\x25\x05\x9f\x7c"
	const offsets = "\x00\x00\x03\x00\x06\x00\x09\x00\x0c\x00\x0f\x00\x12\x00\x15\x00\x18\x00\x1d\x00\x22\x00\x27\x00\x2c\x00\x31\x00\x36\x00\x3b\x00\x40\x00\x45\x00\x4a\x00\x4f\x00\x59\x00\x5e\x00\x66\x00\x6c\x00\x72\x00\x78\x00\x7e\x00\x85\x00\x8c\x00\x93\x00\x9a\x00\xa2\x00\xa9\x00\xb2\x00\xb9\x00\xc1\x00\xc7\x00\xce\x00\xd4\x00\xda\x00\xde\x00\xe2\x00\xe6\x00\xea\x00\xee\x00\xf2\x00\xf6\x00\xfa\x00\xfe\x00\x02\x01\x06\x01\x0a\x01\x0e\x01"
	hash := hashCoreToken(file.Src, token.Start, token.End-token.Start)
	low, high := 0, len(hashes)/4
	for low < high {
		mid := low + (high-low)/2
		at := mid * 4
		candidate := int(hashes[at]) | int(hashes[at+1])<<8 | int(hashes[at+2])<<16 | int(hashes[at+3])<<24
		if candidate < hash {
			low = mid + 1
		} else {
			high = mid
		}
	}
	if low >= len(hashes)/4 {
		return false
	}
	at := low * 4
	if int(hashes[at])|int(hashes[at+1])<<8|int(hashes[at+2])<<16|int(hashes[at+3])<<24 != hash {
		return false
	}
	start := int(offsets[low*2]) | int(offsets[low*2+1])<<8
	end := int(offsets[low*2+2]) | int(offsets[low*2+3])<<8
	return tokenMatchesCoreSymbol(file.Src, token.Start, token.End-token.Start, names[start:end])
}

func hasDotImportCore(info *PackageInfo, fileIndex int) bool {
	for i := 0; i < len(info.Imports); i++ {
		if info.Imports[i].File == fileIndex && info.Imports[i].Dot {
			return true
		}
	}
	return false
}

func coreSelectorContinues(file *syntax.File, nameTok int, end int) bool {
	return nameTok+1 < end && tokCharIs(file, nameTok+1, '.')
}

func coreUnsafeSelector(info *PackageInfo, fileIndex int, file *syntax.File, baseTok int) bool {
	imp := lookupImportTokenNameCore(info, fileIndex, file, baseTok)
	return imp >= 0 && imp < len(info.Imports) && info.Imports[imp].ImportPath == "unsafe"
}

func coreOrdinaryBuiltinToken(file *syntax.File, tok int) bool {
	return tokenTextIs(file, tok, "min") || tokenTextIs(file, tok, "max") || tokenTextIs(file, tok, "clear")
}

func coreLocalWriteOnly(file *syntax.File, tok int, end int) bool {
	if tok > 0 && tokenTextIs(file, tok-1, "*") {
		return false
	}
	for i := tok + 1; i < end; i++ {
		if syntax.TokenLine(file.Tokens[i]) != syntax.TokenLine(file.Tokens[i-1]) && !tokenTextIs(file, i-1, ",") {
			return false
		}
		if tokenTextIs(file, i, ";") || tokenTextIs(file, i, "{") || tokenTextIs(file, i, "}") {
			return false
		}
		if !tokenTextIs(file, i, "=") && !tokenTextIs(file, i, ":=") {
			continue
		}
		for j := tok + 1; j < i; j++ {
			if file.Tokens[j].KindLine&255 != syntax.TokenIdent && !tokenTextIs(file, j, ",") {
				return false
			}
		}
		return true
	}
	return false
}

func unusedCoreLocalToken(scope CoreScope) int {
	for i := 0; i < len(scope.Names); i++ {
		if scope.Names[i].Kind == NameVariable {
			return scope.Names[i].Token
		}
	}
	return -1
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
		if file.Tokens[i].KindLine&255 != syntax.TokenIdent {
			continue
		}
		if i > start && tokenTextIs(&file, i-1, ".") {
			continue
		}
		if tokenTextIs(&file, i, "_") {
			continue
		}
		if i+2 < end && tokenTextIs(&file, i+1, ".") && file.Tokens[i+2].KindLine&255 == syntax.TokenIdent {
			ref := resolveImportSelectorTypeRefCore(fileIndex, info, checked, scope, file, i, i+1, i+2)
			if ref.Kind == TypeRefImportSelector {
				refs = append(refs, ref)
			}
			i += 2
			continue
		}
		if lookupScopeTokenNameCore(scope, &file, i) < 0 {
			symbol := lookupPackageSymbolTokenCore(&info, &file, fileIndex, i)
			if symbol >= 0 {
				refs = append(refs, CoreTypeRef{Kind: TypeRefPackage, File: fileIndex, Token: i, BaseTok: i, DotTok: i, Package: info.Symbols[symbol].Package, Symbol: symbol})
			}
		}
	}
	return refs
}

func resolveImportSelectorCore(fileIndex int, info *PackageInfo, checked []PackageInfo, scope CoreScope, file *syntax.File, baseTok int, dotTok int, nameTok int) CoreSelectorRef {
	selector := CoreSelectorRef{
		BaseTok:     baseTok,
		DotTok:      dotTok,
		NameTok:     nameTok,
		BaseIndex:   -1,
		BasePackage: -1,
		Symbol:      -1,
	}
	scopeIndex := lookupScopeTokenNameCore(scope, file, baseTok)
	if scopeIndex >= 0 && scope.Names[scopeIndex].Kind != NameLabel {
		selector.BaseIndex = scopeIndex
		return selector
	}
	importIndex := lookupImportTokenNameCore(info, fileIndex, file, baseTok)
	if importIndex < 0 || info.Imports[importIndex].Blank || info.Imports[importIndex].Dot {
		return selector
	}
	info.Imports[importIndex].Used = true
	selector.BaseIndex = importIndex
	selector.BasePackage = info.Imports[importIndex].Package
	if selector.BasePackage < 0 || selector.BasePackage >= len(checked) {
		return selector
	}
	symbol := lookupPackageSymbolTextCore(&checked[selector.BasePackage], file, nameTok)
	if symbol < 0 {
		return selector
	}
	selector.Symbol = symbol
	return selector
}

func resolveImportSelectorTypeRefCore(fileIndex int, info PackageInfo, checked []PackageInfo, scope CoreScope, file syntax.File, baseTok int, dotTok int, nameTok int) CoreTypeRef {
	ref := CoreTypeRef{Kind: TypeRefUnknown, File: fileIndex, Token: nameTok, BaseTok: baseTok, DotTok: dotTok, Package: -1, Symbol: -1}
	scopeIndex := lookupScopeTokenNameCore(scope, &file, baseTok)
	if scopeIndex >= 0 && scope.Names[scopeIndex].Kind != NameLabel {
		ref.Kind = TypeRefScope
		return ref
	}
	importIndex := lookupImportTokenNameCore(&info, fileIndex, &file, baseTok)
	if importIndex < 0 || info.Imports[importIndex].Blank || info.Imports[importIndex].Dot {
		return ref
	}
	info.Imports[importIndex].Used = true
	pkg := info.Imports[importIndex].Package
	if pkg < 0 || pkg >= len(checked) {
		return ref
	}
	symbol := lookupPackageSymbolTextCore(&checked[pkg], &file, nameTok)
	if symbol < 0 {
		return ref
	}
	ref.Kind = TypeRefImportSelector
	ref.Package = pkg
	ref.Symbol = symbol
	return ref
}

func lookupScopeTokenNameCore(scope CoreScope, file *syntax.File, tok int) int {
	if len(scope.Names) == 0 || tok < 0 || tok >= len(file.Tokens) {
		return -1
	}
	token := file.Tokens[tok]
	size := token.End - token.Start
	if size < 0 || token.Start < 0 || token.End > len(file.Src) {
		return -1
	}
	hash := hashCoreToken(file.Src, token.Start, size)
	// References overwhelmingly resolve to parameters and locals declared near
	// the point of use. Search newest declarations first so large compiler
	// functions do not rescan their full scope for every identifier.
	for i := len(scope.Names) - 1; i >= 0; i-- {
		if scope.Names[i].Hash != hash {
			continue
		}
		nameTok := scope.Names[i].Token
		if nameTok < 0 || nameTok >= len(file.Tokens) {
			continue
		}
		name := file.Tokens[nameTok]
		if name.End-name.Start != size {
			continue
		}
		if size > 0 && file.Src[token.Start] != file.Src[name.Start] {
			continue
		}
		matches := true
		for j := 1; j < size; j++ {
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

func lookupImportTokenNameCore(info *PackageInfo, fileIndex int, file *syntax.File, tok int) int {
	if tok < 0 || tok >= len(file.Tokens) {
		return -1
	}
	token := file.Tokens[tok]
	size := token.End - token.Start
	if size < 0 || token.Start < 0 || token.End > len(file.Src) {
		return -1
	}
	for i := 0; i < len(info.Imports); i++ {
		if info.Imports[i].File != fileIndex || len(info.Imports[i].Name) != size {
			continue
		}
		matches := true
		for j := 0; j < size; j++ {
			if file.Src[token.Start+j] != info.Imports[i].Name[j] {
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

func lookupPackageSymbolTokenCore(info *PackageInfo, file *syntax.File, fileIndex int, tok int) int {
	_ = fileIndex
	return lookupPackageSymbolTextCore(info, file, tok)
}

func lookupPackageSymbolTextCore(info *PackageInfo, file *syntax.File, tok int) int {
	if tok < 0 || tok >= len(file.Tokens) {
		return -1
	}
	token := file.Tokens[tok]
	size := token.End - token.Start
	if size < 0 || token.Start < 0 || token.End > len(file.Src) {
		return -1
	}
	if len(info.CoreSymbolHash) > 0 {
		hash := hashCoreToken(file.Src, token.Start, size)
		bucket := hash % len(info.CoreSymbolHash)
		for probes := 0; probes < len(info.CoreSymbolHash); probes++ {
			entry := info.CoreSymbolHash[bucket]
			if entry == 0 {
				return -1
			}
			index := entry - 1
			if index >= 0 && index < len(info.Symbols) && tokenMatchesCoreSymbol(file.Src, token.Start, size, info.Symbols[index].Name) {
				return index
			}
			bucket++
			if bucket == len(info.CoreSymbolHash) {
				bucket = 0
			}
		}
		return -1
	}
	low := 0
	high := len(info.Symbols)
	for low < high {
		mid := low + (high-low)/2
		if compareTokenSymbolCore(file.Src, token.Start, size, info.Symbols[mid].Name) > 0 {
			low = mid + 1
		} else {
			high = mid
		}
	}
	for i := low; i < len(info.Symbols) && compareTokenSymbolCore(file.Src, token.Start, size, info.Symbols[i].Name) == 0; i++ {
		if info.Symbols[i].Kind != SymbolMethod {
			return i
		}
	}
	return -1
}

func buildCoreSymbolHash(symbols []Symbol) []int {
	if len(symbols) == 0 {
		return nil
	}
	buckets := make([]int, len(symbols)*2+1)
	for i := 0; i < len(symbols); i++ {
		if symbols[i].Kind == SymbolMethod {
			continue
		}
		name := symbols[i].Name
		bucket := hashCheckString(name) % len(buckets)
		for probes := 0; probes < len(buckets); probes++ {
			entry := buckets[bucket]
			if entry == 0 {
				buckets[bucket] = i + 1
				break
			}
			if symbols[entry-1].Name == name {
				break
			}
			bucket++
			if bucket == len(buckets) {
				bucket = 0
			}
		}
	}
	return buckets
}

func hashCoreToken(src []byte, start int, size int) int {
	hash := 5381
	for i := 0; i < size; i++ {
		hash = ((hash << 5) + hash + int(src[start+i])) & 2147483647
	}
	return hash
}

func tokenMatchesCoreSymbol(src []byte, start int, size int, name string) bool {
	if size != len(name) {
		return false
	}
	for i := 0; i < size; i++ {
		if src[start+i] != name[i] {
			return false
		}
	}
	return true
}

func compareTokenSymbolCore(src []byte, start int, size int, name string) int {
	limit := size
	if len(name) < limit {
		limit = len(name)
	}
	for i := 0; i < limit; i++ {
		if src[start+i] < name[i] {
			return -1
		}
		if src[start+i] > name[i] {
			return 1
		}
	}
	if size < len(name) {
		return -1
	}
	if size > len(name) {
		return 1
	}
	return 0
}

func buildFuncScopeCore(file syntax.File, fn syntax.FuncDecl) (CoreScope, bool, int) {
	var scope CoreScope
	scope.Names = make([]CoreScopeName, 0, coreScopeCapacity(fn.BodyEnd-fn.BodyStart))
	if fn.ReceiverStart >= 0 {
		tok := receiverNameToken(file, fn)
		if tok >= 0 {
			if !addCoreScopeName(&scope, file, tok, NameReceiver, true, false, false) {
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
	if fn.ResultStart >= 0 && fn.ResultEnd > fn.ResultStart && tokCharIs(&file, fn.ResultStart, '(') {
		end := fn.ResultEnd - 1
		if tokCharIs(&file, end, ')') {
			ok, tok := collectCoreFieldNames(file, fn.ResultStart+1, end, NameResult, &scope)
			if !ok {
				return scope, false, tok
			}
		}
	}
	start := fn.BodyStart + 1
	end := fn.BodyEnd - 1
	for i := start; i < end; i++ {
		token := file.Tokens[i]
		kind := token.KindLine & 255
		if kind == syntax.TokenFunc && i+1 < end && tokCharIs(&file, i+1, '(') {
			paramsEnd := findTypeMatching(file, i+1, '(', ')')
			if paramsEnd > i+1 && paramsEnd <= end {
				var literal CoreScope
				ok, tok := collectCoreFieldNames(file, i+2, paramsEnd-1, NameParam, &literal)
				if !ok {
					return scope, false, tok
				}
				for j := 0; j < len(literal.Names); j++ {
					name := literal.Names[j]
					addCoreScopeName(&scope, file, name.Token, name.Kind, false, false, false)
				}
			}
		}
		if kind == syntax.TokenConst || kind == syntax.TokenVar || kind == syntax.TokenType {
			i = collectCoreDeclScope(file, i, end, &scope)
			continue
		}
		if kind == syntax.TokenOperator && token.End == token.Start+2 && file.Src[token.Start] == ':' && file.Src[token.Start+1] == '=' {
			collectCoreShortDeclScope(file, coreLHSStart(file, i, start), i, &scope)
			continue
		}
		if kind == syntax.TokenIdent && coreTokenLooksLikeLabel(file, i, start, end) {
			if !addCoreScopeName(&scope, file, i, NameLabel, true, true, false) {
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
		if first < segEnd && file.Tokens[first].KindLine&255 == syntax.TokenIdent {
			next := first + 1
			if next >= segEnd {
				pending = append(pending, first)
			} else if tokCharIs(&file, next, '.') {
				pending = pending[:0]
			} else {
				if !addCorePendingNames(file, pending, kind, scope) {
					return false, pending[0]
				}
				pending = pending[:0]
				if !addCoreScopeName(scope, file, first, kind, true, false, false) {
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
		if !addCoreScopeName(scope, file, pending[i], kind, true, false, false) {
			return false
		}
	}
	return true
}

func collectCoreLeadingIdentList(file syntax.File, start int, end int, scope *CoreScope, variable bool) {
	i := start
	for i < end {
		if file.Tokens[i].KindLine&255 != syntax.TokenIdent {
			return
		}
		if !tokenTextIs(&file, i, "_") && lookupScopeTokenNameCore(*scope, &file, i) < 0 {
			addCoreScopeName(scope, file, i, NameLocal, false, false, variable)
		}
		i++
		if i < end && tokCharIs(&file, i, ',') {
			i++
			continue
		}
		return
	}
}

func addCoreScopeName(scope *CoreScope, file syntax.File, tok int, kind int, rejectDup bool, labelsOnly bool, variable bool) bool {
	if tok < 0 || tok >= len(file.Tokens) || tokenTextIs(&file, tok, "_") {
		return true
	}
	token := file.Tokens[tok]
	hash := hashCoreToken(file.Src, token.Start, token.End-token.Start)
	if rejectDup {
		for i := 0; i < len(scope.Names); i++ {
			if scope.Names[i].Hash != hash || !coreTokensEqual(&file, scope.Names[i].Token, tok) {
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
	if variable {
		kind = NameVariable
	}
	scope.Names = append(scope.Names, CoreScopeName{Kind: kind, Token: tok, Hash: hash})
	return true
}

func coreTokensEqual(file *syntax.File, left int, right int) bool {
	if left < 0 || left >= len(file.Tokens) || right < 0 || right >= len(file.Tokens) {
		return false
	}
	leftTok := file.Tokens[left]
	rightTok := file.Tokens[right]
	size := leftTok.End - leftTok.Start
	if size < 0 || rightTok.End-rightTok.Start != size || leftTok.Start < 0 || rightTok.Start < 0 || leftTok.End > len(file.Src) || rightTok.End > len(file.Src) {
		return false
	}
	if size > 0 && file.Src[leftTok.Start] != file.Src[rightTok.Start] {
		return false
	}
	for i := 1; i < size; i++ {
		if file.Src[leftTok.Start+i] != file.Src[rightTok.Start+i] {
			return false
		}
	}
	return true
}

func coreTokenLooksLikeLabel(file syntax.File, tok int, start int, end int) bool {
	if tok < start || tok+1 >= end || file.Tokens[tok].KindLine&255 != syntax.TokenIdent || !tokCharIs(&file, tok+1, ':') || tokenTextIs(&file, tok+1, ":=") {
		return false
	}
	if tok == start {
		return true
	}
	prev := tok - 1
	if tokCharIs(&file, prev, '{') || tokCharIs(&file, prev, ',') {
		return false
	}
	return syntax.TokenLine(file.Tokens[prev]) != syntax.TokenLine(file.Tokens[tok]) || tokCharIs(&file, prev, ';') || tokCharIs(&file, prev, '}')
}

func collectCoreDeclScope(file syntax.File, start int, end int, scope *CoreScope) int {
	variable := file.Tokens[start].KindLine&255 == syntax.TokenVar
	specStart := start + 1
	if specStart < end && tokCharIs(&file, specStart, '(') {
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
			collectCoreLeadingIdentList(file, i, specEnd, scope, variable)
			if specEnd <= i {
				i++
			} else {
				i = specEnd
			}
		}
		return closeTok - 1
	}
	specEnd := statementSpecEnd(file, specStart, end)
	collectCoreLeadingIdentList(file, specStart, specEnd, scope, variable)
	return specEnd - 1
}

func collectCoreShortDeclScope(file syntax.File, start int, end int, scope *CoreScope) {
	for i := start; i < end; i++ {
		if file.Tokens[i].KindLine&255 == syntax.TokenIdent {
			if !tokenTextIs(&file, i, "_") && lookupScopeTokenNameCore(*scope, &file, i) < 0 {
				addCoreScopeName(scope, file, i, NameLocal, false, false, true)
			}
		}
	}
}

func coreLHSStart(file syntax.File, assign int, limit int) int {
	start := assign - 1
	for start > limit {
		if tokCharIs(&file, start, ';') || tokCharIs(&file, start, '{') || tokCharIs(&file, start, '}') {
			return start + 1
		}
		if syntax.TokenLine(file.Tokens[start]) != syntax.TokenLine(file.Tokens[assign]) {
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
		kind := file.Tokens[i].KindLine & 255
		if kind != syntax.TokenConst && kind != syntax.TokenVar && kind != syntax.TokenType {
			continue
		}
		specStart := i + 1
		if specStart < end && tokCharIs(&file, specStart, '(') {
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
	if start < 0 || end <= start || start >= len(file.Tokens) || file.Tokens[start].KindLine&255 != syntax.TokenIdent {
		return decls
	}
	typeStart := -1
	typeEnd := -1
	if kind == SymbolType {
		typeStart = start + 1
		if tokenTextIs(&file, typeStart, "=") {
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
